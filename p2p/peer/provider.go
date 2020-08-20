package peer

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/azd1997/ego/epattern"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/protocol/discover"
)


const (
	msgDiscardTime      int64 = 8 //8s
	maxNeighboursRspNum       = 8
	pingHashExpiredTime       = peerExpiredTime
)

// Provider 节点提供者接口
type Provider interface {
	Start()
	Stop()

	// GetPeers 向调用者返回可用的节点
	GetPeers(expect int, exclude map[ID]bool) ([]*Peer, error)

	// AddSeeds 添加seed种子节点，用于provider初始化
	AddSeeds(seeds []*Peer)
}


func NewProvider(ipstr string, port int, id ID) Provider {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		logger.Error("invalid ip: %s\n", ipstr)
		os.Exit(1)
	}

	p := &provider{
		ip:            ip,
		port:          port,
		peerId:id,
		table:         newTable(id),
		pingHash:      make(map[string]time.Time),
		lm:            epattern.NewLoop(1),
	}
	p.udp = NewUDPServer(ip, port)

	return p
}


// Provider接口实现者
type provider struct {
	ip            net.IP
	port          int
	peerId ID
	udp           UDPServer
	table         table
	pingHash      map[string]time.Time // hash为键

	lm *epattern.LoopMode
}

func (p *provider) Start() {
	if !p.udp.Start() {
		logger.Errorln("start udp server failed")
		os.Exit(1)
	}

	go p.loop()
	p.lm.StartWorking()
}

func (p *provider) Stop() {
	if p.lm.Stop() {
		p.udp.Stop()
	}
}

func (p *provider) AddSeeds(seeds []*Peer) {
	p.table.addPeers(seeds, true)
}

func (p *provider) GetPeers(expect int, exclude map[ID]bool) ([]*Peer, error) {
	return p.table.getPeers(expect, exclude), nil
}

func (p *provider) String() string {
	return fmt.Sprintf("[provider] id:%s, with %s:%d\n",
		p.peerId, p.ip.String(), p.port)
}

// Provider的工作循环
func (p *provider) loop() {
	p.lm.Add()
	defer p.lm.Done()

	refreshTicker := time.NewTicker(peerExpiredTime * 2)
	taskTicker := time.NewTicker(2 * time.Second)
	recvQ := p.udp.GetRecvChannel()

	for {
		select {
		case <-p.lm.D:
			return
		case <-taskTicker.C:
			p.ping()
			p.getNeighbours()
		case pkt := <-recvQ:
			p.handleRecv(pkt)
		case <-refreshTicker.C:
			p.refresh()
		}
	}
}

func (p *provider) handleRecv(pkt *UDPPacket) {
	head := &discover.Head{}
	err := head.Decode(bytes.NewReader(pkt.Data))
	if err != nil {
		logger.Warnln("receive error data")
		return
	}

	now := time.Now().Unix()
	if head.Time+msgDiscardTime < now {
		logger.Info("expired Packet from %v", pkt.Addr)
		return
	}

	switch head.Type {
	case discover.MSG_PING:
		p.handlePing(pkt.Data, pkt.Addr)
	case discover.MSG_PONG:
		p.handlePong(pkt.Data, pkt.Addr)
	case discover.MSG_GET_NEIGHBERS:
		p.handleGetNeighbours(pkt.Data, pkt.Addr)
	case discover.MSG_NEIGHBERS:
		p.handleNeighbours(pkt.Data, pkt.Addr)
	default:
		logger.Warn("unknown op: %d\n", head.Type)
		return
	}
}

func (p *provider) send(msg []byte, addr *net.UDPAddr) {
	pkt := &UDPPacket{
		Data: msg,
		Addr: addr,
	}
	p.udp.Send(pkt)
}

func (p *provider) ping() {
	targets := p.table.getPeersToPing()

	for _, peer := range targets {
		pkt := discover.NewPingMsg(p.peerId).Encode()
		if addr, err := net.ResolveUDPAddr("udp", peer.Address()); err == nil {
			p.send(pkt, addr)
			p.pingHash[encoding.ToHex(crypto.HashD(pkt))] = time.Now()
		}
	}
}

func (p *provider) getNeighbours() {
	targets := p.table.getPeersToGetNeighbours()

	for _, peer := range targets {
		pkt := discover.NewGetNeighboursMsg(p.peerId).Encode()

		if addr, err := net.ResolveUDPAddr("udp", peer.Address()); err == nil {
			p.send(pkt, addr)
		}
	}
}

func (p *provider) handlePing(data []byte, remoteAddr *net.UDPAddr) {
	//fmt.Println("555", encoding.ToHex(crypto.HashD(data)))
	ping := &discover.PingMsg{}
	err := ping.Decode(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error ping: %v\n", err)
		return
	}

	p.table.recvPing(NewPeer(remoteAddr.IP, remoteAddr.Port, ping.From))

	// response ping
	pingHash := crypto.HashD(data)
	//fmt.Println("000", encoding.ToHex(pingHash))
	//fmt.Println("777", encoding.ToHex(crypto.HashD(data)))
	pong := discover.NewPongMsg(pingHash, p.peerId)
	//fmt.Println("999", encoding.ToHex(crypto.HashD(data)))
	pongB := pong.Encode()
	//fmt.Println("xxx", encoding.ToHex(crypto.HashD(data)))
	if pongB == nil {
		logger.Warnln("generate Pong failed")
		return
	}
	//fmt.Println("888", encoding.ToHex(crypto.HashD(data)))
	//fmt.Println("111", encoding.ToHex(pong.PingHash))
	p.send(pongB, remoteAddr)
	//fmt.Println("666", encoding.ToHex(crypto.HashD(data)))
}

func (p *provider) handlePong(data []byte, remoteAddr *net.UDPAddr) {
	pong := &discover.PongMsg{}
	err := pong.Decode(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error Pong:%v\n", err)
		return
	}

	pingHash := encoding.ToHex(pong.PingHash)
	if _, ok := p.pingHash[pingHash]; !ok {
		return
	}
	delete(p.pingHash, pingHash)

	p.table.recvPong(NewPeer(remoteAddr.IP, remoteAddr.Port, pong.From))
}

func (p *provider) handleGetNeighbours(data []byte, remoteAddr *net.UDPAddr) {
	getNeighbours := &discover.GetNeighboursMsg{}
	err := getNeighbours.Decode(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error GetNeighbours: %v\n", err)
		return
	}

	fromId := getNeighbours.From

	if !p.table.exists(fromId) {
		logger.Warn("query is not from my peer and ignore it: %v\n", remoteAddr)
		return
	}

	// response
	exclude := make(map[ID]bool)
	exclude[fromId] = true

	neighbours := p.table.getPeers(maxNeighboursRspNum, exclude)
	neighboursMsg := p.genNeighbours(neighbours)
	p.send(neighboursMsg, remoteAddr)
	//fmt.Println("p.send 1", encoding.ToHex(crypto.HashD(neighboursMsg)))

	// 通知自己的邻居们，有个新家伙来了(可能自己的邻居们已经知道，但仍要发，确保节点同步)
	putMsg := p.genNeighbours([]*Peer{NewPeer(remoteAddr.IP, remoteAddr.Port, fromId)})
	for _, n := range neighbours {
		if neighbourAddr, err := net.ResolveUDPAddr("udp", n.Address()); err == nil {
			p.send(putMsg, neighbourAddr)
			//fmt.Println("p.send 2", encoding.ToHex(crypto.HashD(putMsg)))
		}
	}
}

func (p *provider) handleNeighbours(data []byte, remoteAddr *net.UDPAddr) {
	neighbours := &discover.NeighboursMsg{}
	err := neighbours.Decode(bytes.NewReader(data))
	if err != nil {
		logger.Warn("receive error Neighbours: %v\n", err)
		return
	}

	var peers []*Peer
	for _, n := range neighbours.Nodes {
		peers = append(peers, NewPeer(n.Addr.IP, int(n.Addr.Port), n.ID))
	}

	p.table.addPeers(peers, false)
}

func (p *provider) refresh() {
	p.table.refresh()

	curr := time.Now()
	for k, v := range p.pingHash {
		if curr.Sub(v) > pingHashExpiredTime {
			delete(p.pingHash, k)
		}
	}
}

func (p *provider) genNeighbours(peers []*Peer) []byte {
	var nodes []*discover.Node
	for _, peer := range peers {
		addr := discover.NewAddress(peer.IP.String(), int32(peer.Port))
		node := discover.NewNode(addr, peer.ID)
		nodes = append(nodes, node)
	}

	neighbours := discover.NewNeighboursMsg(p.peerId, nodes)
	//fmt.Println("gennei", encoding.ToHex(crypto.HashD(neighbours.Encode())))
	return neighbours.Encode()
}

// only used in test
func (p *provider) getAllPeersForTest() map[ID]*Peer {
	result := make(map[ID]*Peer)
	table := p.table.(*tableImp)

	for _, pst := range table.peers {
		if pst.isAvaible() {
			result[pst.ID] = pst.Peer
		}
	}
	return result
}
