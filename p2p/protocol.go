package p2p

import (
	"fmt"
	"github.com/pkg/errors"

	"github.com/azd1997/ecoin/p2p/peer"
)

const defaultPeerDataChanSize = 2048

// Protocol Protocol接口，handshake/discover/storage/core这四种p2p协议必须实现这个接口
type Protocol interface {
	ID() uint8
	Name() string
}

// ProtocolRunner 协议运行器。 必须实现该接口才能访问p2p网络
type ProtocolRunner interface {
	// Send 向p2p网络发送数据
	// 成功则返回nil，失败返回ErrPeerNotFound或者ErrNoPeers
	Send(dp *PeerData) error

	// GetRecvChan 返回一个通道用于获取网络中的数据
	GetRecvChan() <-chan *PeerData
}

// PeerData 用于在p2p网络中发送或接收数据。 封装了对端节点ID
type PeerData struct {
	// the Peer is the send target or the receive source node ID
	// if it equals peer.ZeroID, means broadcast to every nodes
	// Peer是发送的对端节点ID或者是接受的源节点ID
	Peer peer.ID

	// payload负载数据
	Data []byte
}

// ErrPeerNotFound 节点未找到
type ErrPeerNotFound struct {
	Peer peer.ID
}

func (p ErrPeerNotFound) Error() string {
	return fmt.Sprintf("Peer:%s not found", p.Peer)
}

// ErrNoPeers 网络中找不到可通信节点
var ErrNoPeers = errors.New("Not found any peers on the network yet")

//////////////////////////////////////////////////////////////////////////////////////

// 协议运行器，某一种协议的运行载体
type protocolRunner struct {
	protocol Protocol	// 运行的协议
	Data     chan *PeerData	// 接收或发送数据的有缓冲通道
	sendFunc func(p Protocol, dp *PeerData) error	// 发送函数
	n        *node	// TCP节点
}

func newProtocolRunner(protocol Protocol, sendFunc func(p Protocol, dp *PeerData) error) *protocolRunner {
	runner := &protocolRunner{
		protocol: protocol,
		Data:     make(chan *PeerData, defaultPeerDataChanSize),
		sendFunc: sendFunc,
	}
	return runner
}

func (p *protocolRunner) Send(dp *PeerData) error {
	return p.sendFunc(p.protocol, dp)
}

func (p *protocolRunner) GetRecvChan() <-chan *PeerData {
	return p.Data
}
