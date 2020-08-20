package peer

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/protocol/discover"
)

var providerTestVar = &struct {
	p *provider

	// provider self infomation
	ip          net.IP
	port        int
	addr        *net.UDPAddr
	id  ID

	// remote peer infomation
	remoteIP          net.IP
	remotePort        int
	remoteAddr        *net.UDPAddr
	remoteID ID
}{
	ip:        net.ParseIP("192.168.1.1"),
	port:      10000,
	id:crypto.RandID(),

	remoteIP:        net.ParseIP("192.168.1.2"),
	remotePort:      10081,
	remoteID:crypto.RandID(),
}

func init() {
	tv := providerTestVar

	tv.addr = &net.UDPAddr{IP: tv.ip, Port: tv.port}

	tv.remoteAddr = &net.UDPAddr{IP: tv.remoteIP, Port: tv.remotePort}

	tv.p = &provider{
		ip:            tv.ip,
		port:          tv.port,
		peerId:tv.id,
		udp:           newUDPServerMock(),
		table:         newTableStub(),
		pingHash:      make(map[string]time.Time),
	}
}

func TestPing(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p		// 模拟provider

	p.ping()

	// verify reqeust
	udpMock := p.udp.(*udpServerMock)	// 模拟provider的模拟udp服务器，provider发送的包会经过udpserver真正发出。
										// 而这里则相当于在模拟udp的sendQ处可以截获这个包
	if err := udpMock.checkSendQSize(1); err != nil {
		t.Fatal(err)
	}
	reqPkt, _ := udpMock.pop()	// 把sendQ的请求包弹出来校验

	if err := utils.TCheckAddr("request address", tv.remoteAddr, reqPkt.Addr); err != nil {
		t.Fatal(err)
	}

	pingPkt := reqPkt.Data
	ping := &discover.PingMsg{}
	err := ping.Decode(bytes.NewReader(pingPkt))
	if err != nil {
		t.Fatal("decode Ping failed\n")
	}
	if err := utils.TCheckString("request peerID", string(tv.id), string(ping.From)); err != nil {
		t.Fatal(err)
	}

	// check pingHash and cleanup
	pingHashKey := encoding.ToHex(crypto.HashD(pingPkt))
	if _, ok := p.pingHash[pingHashKey]; !ok {
		t.Fatalf("expect existing pingHash %s\n", pingHashKey)
	}
	delete(p.pingHash, pingHashKey)
}

func TestGetNeighbours(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	p.getNeighbours()

	// verify reqeust
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(1); err != nil {
		t.Fatal(err)
	}
	reqPkt, _ := udpMock.pop()

	if err := utils.TCheckAddr("request address", tv.remoteAddr, reqPkt.Addr); err != nil {
		t.Fatal(err)
	}

	getNeighbourPkt := reqPkt.Data
	getNeighbour := &discover.GetNeighboursMsg{}
	err := getNeighbour.Decode(bytes.NewReader(getNeighbourPkt))
	if err != nil {
		t.Fatal("decode GetNeighbour failed\n")
	}
	if err := utils.TCheckString("request id", string(tv.id), string(getNeighbour.From)); err != nil {
		t.Fatal(err)
	}
}

func TestHandlePing(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	remotePingPkt := discover.NewPingMsg(tv.remoteID).Encode()		// 模拟的ping请求
	rc := append([]byte{}, remotePingPkt...)	// 不知道为什么不复制一份的话，会在p.handlePing中的pong.Encode修改，导致remotePingPkt发生变化
	//fmt.Println("444", encoding.ToHex(crypto.HashD(remotePingPkt)))
	p.handlePing(remotePingPkt, tv.remoteAddr)		// 处理ping请求，把pong回应发到sendQ

	// verify response
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(1); err != nil {
		t.Fatal(err)
	}
	respPkt, _ := udpMock.pop()


	if err := utils.TCheckAddr("response address", tv.remoteAddr, respPkt.Addr); err != nil {
		t.Fatal(err)
	}

	pongPkt := respPkt.Data

	pong := &discover.PongMsg{}
	err := pong.Decode(bytes.NewReader(pongPkt))
	if err != nil {
		t.Fatalf("decode pong failed: %v\n", err)
	}
	//fmt.Println("222", encoding.ToHex(pong.PingHash))
	//fmt.Println("333", encoding.ToHex(crypto.HashD(remotePingPkt)))


	//fmt.Println(len(pong.PingHash), len(crypto.HashD(remotePingPkt)))
	if err := utils.TCheckBytes("ping hash", pong.PingHash, crypto.HashD(rc)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckString("response ID", string(p.peerId), string(pong.From)); err != nil {
		t.Fatal(err)
	}
}

func TestHandlePong(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	pingHash := crypto.RandHash()
	pingHashKey := encoding.ToHex(pingHash)
	pongPkt := discover.NewPongMsg(pingHash, tv.remoteID).Encode()

	p.pingHash[pingHashKey] = time.Now()
	p.handlePong(pongPkt, tv.remoteAddr)

	if len(p.pingHash) != 0 {
		t.Fatal("expect clean pingHash after handle pong\n")
	}
}

func TestHandleGetNeighboursNotFromMyPeers(t *testing.T) {
	p := providerTestVar.p

	unknownPeerID := crypto.RandID()
	getNeighboursPkt := discover.NewGetNeighboursMsg(unknownPeerID).Encode()
	p.handleGetNeighbours(getNeighboursPkt, &net.UDPAddr{IP: net.ParseIP("1.2.3.4"), Port: 999})

	// verify response
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(0); err != nil {
		t.Fatal(err)
	}
}

//TODO: 莫名其妙的BUG
func TestHandleGetNeighbours(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	getNeighboursPkt := discover.NewGetNeighboursMsg(tv.remoteID).Encode()
	p.handleGetNeighbours(getNeighboursPkt, tv.remoteAddr)
	// 测试时，p应该是会发两个内容完全相同的NeighboursMsg到p.udp.sendQ

	// verify response
	udpMock := p.udp.(*udpServerMock)
	if err := udpMock.checkSendQSize(2); err != nil {
		t.Fatal(err)
	}
	pkt, err := udpMock.pop()
	if err != nil {
		log.Printf("pop pkt: %v\n", err)
	}
	pkt2, err := udpMock.pop()
	if err != nil {
		log.Printf("pop pkt2: %v\n", err)
	}
	pkt3, err := udpMock.pop()
	pkt3 = pkt3
	if err != nil {
		log.Printf("pop pkt3: %v\n", err)
	}

	if err := utils.TCheckAddr("response address", tv.remoteAddr, pkt.Addr); err != nil {
		t.Fatal(err)
	}

	neighbours := &discover.NeighboursMsg{}
	fmt.Println("decode pkt2", encoding.ToHex(crypto.HashD(pkt2.Data)), pkt2.Addr)
	fmt.Println("decode", encoding.ToHex(crypto.HashD(pkt.Data)), pkt.Addr)
	err = neighbours.Decode(bytes.NewReader(pkt2.Data))
	if err != nil {
		t.Fatalf("decode Neighbours failed: %v\n", err)
	}

	if len(neighbours.Nodes) != 1 {
		t.Fatal("expect 1 node in Neighbours\n")
	}

	node := neighbours.Nodes[0]
	if err := utils.TCheckString("node ID", string(tv.remoteID), string(node.ID)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckIP("node IP", tv.remoteIP, node.Addr.IP); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("node port", tv.remotePort, int(node.Addr.Port)); err != nil {
		t.Fatal(err)
	}
}

func TestHandleNeighbours(t *testing.T) {
	tv := providerTestVar
	p := providerTestVar.p

	neighbourPkt := discover.NewNeighboursMsg(p.peerId, []*discover.Node{
		discover.NewNode(discover.NewAddress(tv.remoteIP.String(), int32(tv.remotePort)), tv.remoteID),
	}).Encode()
	p.handleNeighbours(neighbourPkt, tv.remoteAddr)

	// verify add result
	table := p.table.(*tableMock)
	if err := utils.TCheckInt("table add list size", 1, len(table.add)); err != nil {
		t.Fatal(err)
	}

	node := table.add[0]
	if err := utils.TCheckIP("node IP", tv.remoteIP, node.IP); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("node port", tv.remotePort, node.Port); err != nil {
		t.Fatal(err)
	}
}

/////////////////////////////////////////////////tableMock

type tableMock struct {
	peer *Peer
	add  []*Peer
}

func newTableStub() *tableMock {
	return &tableMock{
		peer: NewPeer(providerTestVar.remoteIP, providerTestVar.remotePort, providerTestVar.remoteID),
	}
}
func (t *tableMock) addPeers(p []*Peer, isSeed bool) {
	t.add = p
}
// 在Mock的table中exclude不起作用
func (t *tableMock) getPeers(expect int, exclude map[ID]bool) []*Peer {
	return []*Peer{t.peer}
}
func (t *tableMock) exists(id crypto.ID) bool {
	return id == t.peer.ID
}
func (t *tableMock) getPeersToPing() []*Peer {
	return []*Peer{t.peer}
}
func (t *tableMock) getPeersToGetNeighbours() []*Peer {
	return []*Peer{t.peer}
}
func (t *tableMock) recvPing(p *Peer) {}
func (t *tableMock) recvPong(p *Peer) {}
func (t *tableMock) refresh()         {}

////////////////////////////////////////////////udpServerMock

type udpServerMock struct {
	sendQ []*UDPPacket
}

func newUDPServerMock() *udpServerMock {
	return &udpServerMock{}
}
func (u *udpServerMock) GetRecvChannel() <-chan *UDPPacket {
	return nil
}
func (u *udpServerMock) Send(packet *UDPPacket) {
	log.Printf("有一个包待发送：%s|%s\n", packet.Addr, encoding.ToHex(crypto.HashD(packet.Data)))
	u.sendQ = append(u.sendQ, packet)
}
func (u *udpServerMock) Start() bool {
	return true
}
func (u *udpServerMock) Stop() {}
func (u *udpServerMock) checkSendQSize(expect int) error {
	return utils.TCheckInt("udp send queue size", expect, len(u.sendQ))
}
func (u *udpServerMock) pop() (*UDPPacket, error) {
	if len(u.sendQ) == 0 {
		return nil, fmt.Errorf("empty sendQ")
	}

	result := u.sendQ[0]
	u.sendQ = u.sendQ[1:]
	log.Printf("弹出发送队列首部的包：%s|%s\n", result.Addr, encoding.ToHex(crypto.HashD(result.Data)))
	return result, nil
}
