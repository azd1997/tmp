package p2p

import (
	"fmt"
	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/log"
	"github.com/azd1997/ego/epattern"
	"net"
	"os"
	"sync"
	"time"

	"github.com/azd1997/ecoin/p2p/peer"
)

// Config P2P网络节点配置
type Config struct {
	// TCP通信地址（ip/port）
	NodeIP     string
	NodePort   int
	// UDP节点（提供器）
	Provider   peer.Provider
	// 最大连接节点数量
	MaxPeerNum int
	// 本机结点账户
	Account *account.Account	// Account包含了类型角色类型信息
	// 本机区块链的ID
	ChainID    uint8
}

// Node P2P网络节点
type Node interface {
	AddProtocol(p Protocol) ProtocolRunner
	Start()
	Stop()
}

// NewNode 新建一个节点
func NewNode(c *Config) Node {
	if !role.IsRole(c.Account.RoleNo) {
		logger.Error("invalid node role %d\n", c.Account.RoleNo)
		return nil
	}

	n := &node{
		account:        c.Account,
		chainID:        c.ChainID,
		maxPeersNum:    c.MaxPeerNum,
		peerProvider:   c.Provider,
		protocols:      make(map[uint8]*protocolRunner),
		ngBlackList:    make(map[peer.ID]time.Time),
		tcpConnectFunc: TCPConnectTo,
		connectTask:    make(chan *peer.Peer, c.MaxPeerNum),
		connMgr:        newConnManager(c.MaxPeerNum),
		lm:             epattern.NewLoop(1),
	}
	n.ng = newNegotiator(n.account, n.chainID)

	var ip net.IP
	if ip = net.ParseIP(c.NodeIP); ip == nil {
		log.Error("parse ip for tcp server failed:%s\n", c.NodeIP)
	}
	n.tcpServer = NewTCPServer(ip, c.NodePort)

	return n
}

type node struct {
	// TCP服务器
	tcpServer TCPServer

	// 账户
	account *account.Account

	// 本地区块链标识
	chainID   uint8

	// 最大节点连接数量
	maxPeersNum  int

	// 节点提供器
	peerProvider peer.Provider

	// 协议表
	protocolsMutex sync.Mutex
	protocols      map[uint8]*protocolRunner //<Protocol ID, ProtocolRunner>

	// 协商器
	ng          negotiator
	ngMutex     sync.Mutex
	ngBlackList map[peer.ID]time.Time	// 黑名单

	// TCP连接函数
	tcpConnectFunc func(ip net.IP, port int) (TCPConn, error)
	// 连接任务（channel中传来的peer待与之建立链接）
	connectTask    chan *peer.Peer
	// 连接管理器
	connMgr        connManager

	lm *epattern.LoopMode
}

// AddProtocol 添加一个运行时的P2P协议： shakehand/discover/storage/core
// TODO: 关于PROTOCOL的描述。 其实Protocol主要是为了core protocol准备的
func (n *node) AddProtocol(p Protocol) ProtocolRunner {
	n.protocolsMutex.Lock()
	defer n.protocolsMutex.Unlock()

	if v, ok := n.protocols[p.ID()]; ok {
		logger.Error("protocol conflicts in ID:%s, exists:%s, wanted to add:%s\n",
			p.Name(), v.protocol.Name(), v.protocol.Name())
		os.Exit(1)
	}
	// 协议运行器都需要借助发送通道来发送数据包，再由TCP服务器去异步地发送数据
	runner := newProtocolRunner(p, n.send)
	n.protocols[p.ID()] = runner
	return runner
}

func (n *node) Start() {
	if !n.tcpServer.Start() {
		logger.Errorln("start node's tcp server failed")
		os.Exit(1)
	}
	n.connMgr.start()

	go n.loop()
	n.lm.StartWorking()
}

func (n *node) Stop() {
	if n.lm.Stop() {
		n.tcpServer.Stop()
		n.connMgr.stop()
	}
}

func (n *node) String() string {
	return fmt.Sprintf("[node] listen on %v", n.tcpServer.Addr())
}

func (n *node) loop() {
	n.lm.Add()
	defer n.lm.Done()

	// 获取结点进行连接 Ticker
	getPeersToConnectTicker := time.NewTicker(10 * time.Second)
	// 状态报告 Ticker
	statusReportTicker := time.NewTicker(15 * time.Second)
	// 协商器黑名单
	ngBlackListCleanTicker := time.NewTicker(1 * time.Minute)

	// 获取TCP服务器的链接接收通道
	acceptConn := n.tcpServer.GetTCPAcceptConnChannel()
	for {
		select {
		case <-n.lm.D:
			return
		case <-getPeersToConnectTicker.C:
			n.getPeersToConnect()
		case <-statusReportTicker.C:
			n.statusReport()
		case <-ngBlackListCleanTicker.C:
			n.cleanNgBlackList()
		case newPeer := <-n.connectTask:	// 有新节点待建立链接
			go func() {
				n.lm.Add()
				n.setupConn(newPeer)
				n.lm.Done()
			}()
		case newPeerConn := <-acceptConn:	// 有新TCP链接，那么让其到一个go程自己去跑
			go func() {
				n.lm.Add()
				newPeerConn.SetSplitFunc(splitTCPStream)
				n.recvConn(newPeerConn)
				n.lm.Done()
			}()
		}
	}
}

// 每个节点的链接管理器有最大结点数的限制
// 不满这个最大值时，还可以从provider获取expectNum个结点来建立起链接
func (n *node) getPeersToConnect() {
	peersNum := n.connMgr.size()
	if peersNum >= n.maxPeersNum {
		return
	}

	expectNum := n.maxPeersNum - peersNum
	excludePeers := n.getExcludePeers()
	newPeers, err := n.peerProvider.GetPeers(expectNum, excludePeers)
	if err != nil {
		logger.Warn("get peers from provider failed:%v\n", err)
		return
	}

	// 将获得的新结点放入待建立链接的connectTask通道
	for _, newPeer := range newPeers {
		n.connectTask <- newPeer
	}
}

// 打印当前连接管理器所维护的对端结点列表
func (n *node) statusReport() {
	logger.Info("current address book:%v\n", n.connMgr)
}

// 与对端结点建立链接
// 这里假设对端结点也在与自己发起链接。这种时候根据结点ID的字典序大小来决定谁作客户端，谁作服务端。（小者作客户端）
func (n *node) setupConn(newPeer *peer.Peer) {
	// 自己比对端大则睡10秒，等对方向自己发起链接请求
	if crypto.PrivateKey2ID(n.account.PrivateKey, n.account.RoleNo) > newPeer.ID {
		time.Sleep(10 * time.Second)
	}
	// 如果自己检查到对端结点已经在连接管理器中（意味着已经建立了链接）
	if n.connMgr.isExist(newPeer.ID) {
		return
	}

	// 否则向对方发起链接，先建立正儿八经的TCP链接
	conn, err := n.tcpConnectFunc(newPeer.IP, newPeer.Port)
	if err != nil {
		logger.Warn("setup conection to %v failed:%v", newPeer, err)
		return
	}
	// 设置数据流分割方法
	conn.SetSplitFunc(splitTCPStream)

	// 与对方进行握手协商，获取消息加密模块
	ec, err := n.ng.handshakeTo(conn, newPeer)
	if err != nil {
		logger.Warn("handshake to %v failed:%v", newPeer, err)
		conn.Disconnect()
		n.addNgBlackList(newPeer.ID)
		return
	}

	// 将对方添加到本机节点的链接管理器当中
	n.addConn(newPeer, conn, ec)
}

// 接受链接
// 当某个结点（对端结点）向本机发起连接时（此时已经建立了TCP链接，但是我们要决定是否建立P2P链接），
// 要查看是否连接管理器还有位置、 尝试接受握手。 都没有问题，才将该链接添加到本机结点的连接管理器
func (n *node) recvConn(conn TCPConn) {
	accept := false
	if n.connMgr.size() < n.maxPeersNum {
		accept = true
	}

	peer1, ec, err := n.ng.recvHandshake(conn, accept)
	if err != nil {
		logger.Warn("handle handshake from remote failed:%v\n", err)
		conn.Disconnect()
		return
	}

	if !accept {
		conn.Disconnect()
		return
	}

	n.addConn(peer1, conn, ec)
}

// 将某个链接添加到连接管理器
func (n *node) addConn(peer *peer.Peer, conn TCPConn, ec codec) {
	if err := n.connMgr.add(peer, conn, ec, n.recv); err != nil {
		logger.Info("addConn failed:%v\n", err)
		conn.Disconnect()
	}
}

// 以某种协议向对端结点发送数据
func (n *node) send(p Protocol, dp *PeerData) error {
	return n.connMgr.send(p, dp)
}

// 接受数据，将其丢给对应的protocolRunner去处理
func (n *node) recv(peer peer.ID, protocolID uint8, data []byte) {
	logger.Debug("recv a protocol[%d] packet, size %d\n", protocolID, len(data))
	if runner, ok := n.protocols[protocolID]; ok {
		select {
		case runner.Data <- &PeerData{
			Peer: peer,
			Data: data,
		}:
		default:
			logger.Warn("protocol %s recv packet queue full, drop it\n",
				runner.protocol.Name())
		}
	}
}

// 协商黑名单
func (n *node) addNgBlackList(peerID peer.ID) {
	n.ngMutex.Lock()
	defer n.ngMutex.Unlock()
	n.ngBlackList[peerID] = time.Now()
}

// 协商黑名单30分钟解封
func (n *node) cleanNgBlackList() {
	n.ngMutex.Lock()
	defer n.ngMutex.Unlock()

	curr := time.Now()
	for k, v := range n.ngBlackList {
		if curr.Sub(v) > 30*time.Minute {
			delete(n.ngBlackList, k)
		}
	}
}

// 黑名单与已建立链接的结点都应该被排除，不应该加入到connectTask通道
func (n *node) getExcludePeers() map[peer.ID]bool {
	result := make(map[peer.ID]bool)

	n.ngMutex.Lock()
	for k := range n.ngBlackList {
		result[k] = true
	}
	n.ngMutex.Unlock()

	connectedID := n.connMgr.getIDs()
	for _, id := range connectedID {
		result[id] = true
	}

	return result
}
