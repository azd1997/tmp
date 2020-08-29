/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/21 22:49
* @Description: The file is for
***********************************************************************/

package consensus

import (
	"context"
	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/protocol/core"
)

// 想要比较好的设计一个可插拔的共识模块，非常困难，尤其是PoW/PoT/Raft/PBFT这四类情况差别比较大
// PoW存在延迟确认与分叉情况
// PoT先投票后出块（相当于写请求），而且需要好好考虑网络分区问题
// Raft和PBFT是强一致性共识，Raft只做到CFT（崩溃容错），PBFT则是拜占庭容错
// 按网络拓扑划分，PoT和PoW都是对称网络共识，而Raft/PBFT非对称

// 因此将consensus进行抽象然后交给Node使用很困难，
// 只有将consensus看作是比较大的模块，包含网络和区块链操作才行

// consensus其实是共识节点间的问题。考虑将consensus定位共识节点包，外部使用者调consensus时屏蔽共识过程
// 对于外部用户，其实只关心交易层，而不关心区块层，区块看作是交易的批处理
// 外部用户只需要进行：添加交易；查询区块链数据

// 对于区块链结构，可以统一使用带分叉的结构，Raft/PBFT时不理睬分支即可

// 既然共识模块设计为可替换，那么就是尽量在consensus下作实现，而避免其他包实现consensus包的接口

type RawNode struct {
	server Server
	client Client
}

// Client 客户端
type Client interface {

	// SendTx 发送交易
	// ctx 存储一个map[string]interface{}，
	// 	目前使用map["result"] = chan *Result来起到回调通知的作用
	//	map["txchan"] = chan *Tx 用于Client向Server传递交易
	// tx 构建的交易，发送到共识集群中
	// servers 发送到的server的ID，可以指定发给哪些server，空着的话则是广播
	SendTx(ctx context.Context, tx *core.Tx, servers ...string)
}


// Server 服务端
// 对于共识节点来说，Server就是共识节点参与方
// 对于非共识节点，Server只是代理服务器，负责转发至真正的共识节点集群
type Server interface {

	// CollectLoop 收集循环
	// 	来源一： map["txchan"] = chan *Tx
	//  来源二： 其他共识节点传递过来的
	// 收集交易之后需要有触发条件，将之打包为区块，再将区块进行同步
	CollectLoop(ctx context.Context)
}






// Consensuser 共识器。 描述了一个共识节点。
// 接收外部客户端传来的请求（交易），
type Consensuser interface {
	// WriteTx
	WriteTx(ctx context.Context, tx core.Tx, to ...account.ID)
}

// TxBroadcaster 交易广播器
// 通过定制交易广播器，可以制定接收到交易后的广播策略，不广播? 广播? 多播?
type TxBroadcaster interface {

}

// Peer P2P共识节点
// 只有纯粹的收发能力，并不关系具体细节
type Peer interface {

}