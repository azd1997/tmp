/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/9/13 17:17
* @Description: node处理输入输出，驱动pot变更状态
***********************************************************************/

package pot

import (
	"github.com/pkg/errors"
	"os"
	"time"

	"github.com/azd1997/ecoin/common/log"
	"github.com/azd1997/ecoin/consensus/implement/pot/potpb"
	"github.com/azd1997/ecoin/protocol/core"
)


type Peer struct {
	id string	// 由账户公钥转换而来的id
	//addr string	// ip:port
	//name string	// 账户所代表的单位/个体名称
}


// Node 节点接口
type Node interface {
	// 启动节点
	Start(peers []*Peer, tickMs int32)

	// 停止节点
	Stop()

	// run
	run()

	// tick 其实就是HalfEP甚至更细粒度的时钟
	// 用于驱动pot状态机变更
	tick()

	// 节点内部的时钟驱动应该是：
	// 合适的时间到达，则发起pot竞争，需要有proof，
	// 状态机应该向外传递一个包含一个无缓冲chanel的结构体， 类似于 chan chan Proof，
	// 进一步地，如何做到内部状态机向外传递通知呢？
	// 提供一个类似 BlockChannel的函数，外部监听。
	// 内部状态机除了proof还有block，都需要传入，
	// 但要注意，确定proof时其实确定了Block，Block能够包含proof信息，因此只需要传送Block
	// 那么使用Node时，需要先调用BlockChannel获取到 blockc := BlockChannel()，
	// 而后需要在for{select{}}中监听blockc，收到chan Block后准备最新的Block塞进去
	// 这个BlockChannel由区块链模块持有
	// 外部只写
	BlockChannel() chan chan core.Block


	// 另外一个需要注意的是，Block需要被检验，但并不希望共识协议中去涉及这些，因此还需要验证器模块，
	// 同样的，是内部状态机主动需要调用验证器能力，所以验证器也可以做成上面BlockChannel的使用模式
	// 但是，由于验证器与区块链、交易池、账户表三个模块高度相关；同时Pot状态机必然依赖于一份区块存储
	// 为了满足这两个依赖关系，需要将区块链分为两个层级：底层些的只负责区块的存储与持久化，上层的则需
	// 要关注当区块链变化时其他相关组件的变更
	// 尽管通过channel作数据传输会影响效率，但是有助于将模块拆分开来，简化总体的编码难度
	// 考虑上述组件都使用channel（chan chan xxx）解耦
	// 则验证器也需要解耦 chan reqValidate{data, result (chan Result) }
	// 这个ValidateChannel由validator模块持有
	ValidateChannel()  chan ReqValidate

	// 发起Pot竞争后，pot状态机需要收集其他节点的证明，则需要网络模块的支持，
	// 网络模块在收到某个证明之后，先传给验证器检查，没有问题，再传给pot状态机
	// 因此，Node需要循环收集proof，并交给pot状态机处理
	CollectProofsChannel() chan potpb.Proof

	//
}

// NewNode 新建节点
func NewNode() Node {
	return &node{}
}

// node Node的实现
type node struct {
	/*
		节点列表相关
	*/
	// 邻居节点列表
	peers []*Peer
	// 邻居节点列表大小上限
	maxPeerNum int32


	// 滴答间隔（时间尺度）. 默认500ms
	tickMs int32


	/*
		节点列表相关
	*/


	/*
	Node对外提供的chan
	*/

	// pot状态机
	pot *Pot
}



// Start
// peers 外部传进来的（由文件加载而来的）邻居节点列表
// tickMs 一次“滴答”的时长，单位ms。默认tickMs=500ms
func (n *node) Start(peers []*Peer, tickMs int32) {
	if len(peers) == 0 {
		log.Error("node.Start requires at least 1 peer")
		os.Exit(1)
		//return errors.New("node.Start requires at least 1 peer")
	}

	// 从本地文件加载原先持久化的邻居节点列表 {id, ipaddr, 其他信息}
	if len(n.peers) != 0 {
		n.peers = peers
	} else {
		log.Error("node.Start requires empty node.peers, but got sizeof(node.peers) = %d", len(n.peers))
		os.Exit(1)
		//return errors.New("node.Start requires empty node.peers")
	}

	// 设置滴答间隔
	if tickMs <= 100 {
		tickMs = 500
		log.Warn("node.Start got a tickMs <= 100, correct it to default value (500ms)")
	}
	n.tickMs = tickMs

	// 运行节点
	go n.run()

}

func (n *node) Stop() {
	// ...
}

func (n *node) run() {
	ticker := time.Tick(time.Duration(n.tickMs) * time.Millisecond)

	for {
		select {
		// “逻辑时钟”
		case <- ticker:
			n.tick()	// 驱动pot状态转换

		//
		}
	}
}

// tick 向内部状态机pot发送tick信号。内部状态机处理tick信号(虚拟信号)，自行切换状态
func (n *node) tick() {
	n.pot.tick()
}

func (n *node) BlockChannel() chan chan core.Block {

}

func (n *node) ValidateChannel() chan interface{} {
	panic("implement me")
}

func (n *node) CollectProofsChannel() chan potpb.Proof {
	panic("implement me")
}



