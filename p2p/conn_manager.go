package p2p

import (
	"fmt"
	"github.com/azd1997/ecoin/common/crypto"
	"net"
	"sync"

	"github.com/azd1997/ego/epattern"

	"github.com/azd1997/ecoin/p2p/peer"
)

// connManager manages all the conn
type connManager interface {
	// 启动链接管理器
	start()
	// 停止链接管理器
	stop()
	// 链接个数
	size() int
	// 获取所有链接的对端节点peer.ID
	getIDs() []peer.ID
	// 判断某个对端节点peer.ID是否存在
	isExist(peerID peer.ID) bool
	// 向某个对端节点发送某种协议的数据。 对端节点和数据封装在PeerData。 Protocol代表了通信的协议：handshake/discover/storage/core
	send(p Protocol, pd *PeerData) error
	// 添加一个新链接。 传入的是创建新链接所需的参数
	add(peer *peer.Peer, conn TCPConn, ec codec, handler recvHandler) error
	// 用于打印一些信息
	String() string
}

func newConnManager(maxPeerNum int) connManager {
	return &connManagerImp{
		conns:      make(map[peer.ID]*conn),
		maxPeerNum: maxPeerNum,
		removing:   make(chan peer.ID, maxPeerNum),
		lm:         epattern.NewLoop(1),
	}
}

type connManagerImp struct {
	sync.RWMutex
	// 链接表 <peer ID, conn>
	conns      map[peer.ID]*conn
	// 最大链接节点数量
	maxPeerNum int
	// 移除链接的有缓冲通道，缓冲容量为maxPeerNum
	removing   chan peer.ID
	// 循环模式
	lm         *epattern.LoopMode
}

func (c *connManagerImp) start() {
	go c.loop()
	c.lm.StartWorking()
}

func (c *connManagerImp) stop() {
	c.Lock()
	defer c.Unlock()

	// 关闭所有链接
	if c.lm.Stop() {
		for _, conn := range c.conns {
			conn.stop()
		}
	}
}

func (c *connManagerImp) size() int {
	c.RLock()
	defer c.RUnlock()

	return len(c.conns)
}

func (c *connManagerImp) getIDs() []peer.ID {
	c.RLock()
	defer c.RUnlock()

	var result []peer.ID
	for key := range c.conns {
		result = append(result, key)
	}
	return result
}

func (c *connManagerImp) isExist(peerID peer.ID) bool {
	c.RLock()
	defer c.RUnlock()

	_, ok := c.conns[peerID]
	return ok
}

// 发送数据。 根据PeerData所含有的PeerID，如果为空，则向链接管理器内所有Peer广播数据
func (c *connManagerImp) send(p Protocol, data *PeerData) error {
	if c.size() == 0 {
		return ErrNoPeers
	}

	// broadcast 广播
	if data.Peer == crypto.ZeroID {
		c.Lock()
		for _, conn := range c.conns {
			conn.send(p.ID(), data.Data)
		}
		c.Unlock()
		return nil
	}

	// unicast 单播
	c.Lock()
	conn, ok := c.conns[data.Peer]
	c.Unlock()
	if !ok {
		return ErrPeerNotFound{Peer: data.Peer}
	}
	conn.send(p.ID(), data.Data)

	return nil
}

func (c *connManagerImp) add(peer *peer.Peer, conn TCPConn, ec codec, handler recvHandler) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.conns[peer.ID]; ok {
		return fmt.Errorf("already exist a connection with %s", peer.ID)
	}

	if len(c.conns) >= c.maxPeerNum {
		return fmt.Errorf("over max peer(%d) limits", len(c.conns))
	}

	connection := newConn(peer, conn, ec, handler)
	c.conns[peer.ID] = connection
	// 设置链接断开时的回调函数：打印信息，并从链接管理器移除该链接
	conn.SetDisconnectCb(func(addr net.Addr) {
		logger.Info("disconnect peer %v, address %v\n", peer.ID, addr)
		c.removeConn(peer.ID)
	})
	connection.start()

	logger.Info("add conn of %v\n", peer)
	return nil
}

func (c *connManagerImp) String() string {
	c.RLock()
	defer c.RUnlock()

	var result string
	for k, v := range c.conns {
		result += "[" + string(k) + " " + v.p.Address() + "] "
	}
	return result
}

func (c *connManagerImp) loop() {
	c.lm.Add()
	defer c.lm.Done()

	for {
		select {
		case <-c.lm.D:	// 循环结束
			return
		case rmID := <-c.removing:	// 移除某个/些链接
			c.Lock()
			delete(c.conns, rmID)
			c.Unlock()
		}
	}
}

func (c *connManagerImp) removeConn(peerID peer.ID) {
	c.removing <- peerID
}
