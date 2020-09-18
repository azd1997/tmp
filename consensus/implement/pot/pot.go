/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/8/26 17:28
* @Description: The file is for
***********************************************************************/

// PoT基于以下假设：
// 每个节点持有其他所有节点的地址信息。（实际情况是，可能缺少部分）

package pot

import (
	"errors"
	"time"

	pb "github.com/azd1997/ecoin/consensus/implement/pot/potpb"
	"github.com/azd1997/ecoin/consensus/txpool"
	"github.com/azd1997/ecoin/consensus/validate"
)

// Pot 代表一个Pot共识节点
type Pot struct {
	// 当前处于哪个阶段
	Stage Stage

	// PoT节点标识
	id string

	// PotLog 日志存储
	PotLog *PotLog

	// 日志复制进度
	// 广播区块之后
	// 用于所有节点，维护的也是节点自身的邻居表的进度
	Prs map[string]*Progress


	// proofs 竞赛分数证明
	proofs map[string]*pb.Proof

	// msgs 待发送的消息，本地消息以及网络消息都写到该队列
	msgs []*pb.Message

	// Winner 当前轮pot竞赛的胜者，在下一轮竞赛开启时重置
	Winner string

	// validator 验证模块，外部提供。validator须提供一个validate(data, datatype) bool 函数
	Validator validate.Validator

	// TxPool
	TxPool txpool.TxPool

	HalfEP int64	// 半个出块周期 ns

	potStart *time.Timer	// pot竞争开始的定时器
	potEnd *time.Timer		// pot竞争结束的定时器
	waitBlockTimeout *time.Timer	// 等待新区块的超时定时器

	// nextIndex 表示接下来将要生产哪一个区块（条目）的阶段
	nextIndex uint64
	// 当前允许接收证明消息
	collectProof bool

	// 等待新区块（索引）； 0说明没在等区块
	waitBlock uint64

	// 停止模块的通知
	closing chan struct{}
}

func newPot(c Config) *Pot {
	return &Pot{}
}

// Pot
// 逻辑时钟
// 竞争
// 出块

// Start 启动
func (pot *Pot) Start() {
	go pot.potLoop()
}

// Stop 停止
func (pot *Pot) Stop() {
	close(pot.closing)
}

// Pot循环
func (pot *Pot) potLoop() {
	// 时钟


	// 时钟切换
	for {
		select {
		case <- pot.potStart.C:
			// 开始竞争
			pot.startCompetition()
		case <- pot.potEnd.C:
			// 结束证明收集
			pot.finishCompetition()
		}
	}

}

// 参与竞争
func (pot *Pot) startCompetition() {
	// 控制交易池，取出证明
	merkle := pot.TxPool.PrepareTxs()
	proof := &pb.Proof{
		TxsMerkle: merkle,
	}
	msg := &pb.Message{}

	// 广播证明消息
	err := pot.broadcast(msg)

	// 设置允许收集证明
	pot.collectProof = true
}

// 处理证明消息
func (pot *Pot) handleMsgProof(msg *pb.Message) error {
	// 如果当前不允许接收证明消息
	if !pot.collectProof {
		return errors.New("no collect proof")
	}

	// 验证消息，验证消息身份和哈希格式等
	if res := pot.Validator.Validate(msg); res != validate.OK {
		return errors.New(string(res))
	}

	// 从msg提取proof
	proof := &pb.Proof{}

	// 对于验证通过的消息，需要加入到proofs
	pot.proofs[proof.From] = proof

	// 更新winner
	if pot.Winner == "" {
		pot.Winner = proof.From
	} else if proof.GreaterThan(pot.proofs[pot.Winner]) {
		pot.Winner = proof.From
	}
}

func (pot *Pot) finishCompetition() {
	// 禁止收集证明消息
	pot.collectProof = false

	// 固定获胜者信息
	winner := pot.Winner

	// 是自己获胜吗？
	if winner == pot.id {
		// 构建区块并广播
		blockBytes := pot.genEntry()
		msg := &pb.Message{}
		pot.broadcast(msg)
	} else {
		// 等待接收新区块
		pot.waitBlock = pot.proofs[pot.Winner].Index
	}
}

// 生成区块
func (pot *Pot) genEntry() []byte {

}

func (pot *Pot) handleMsgEntry(msg *pb.Message) error {
	if msg.MessageType != pb.MessageType_Block {
		return errors.New("not msg block")
	}
	// 检查消息有效性
	pot.Validator.Validate(msg)

}



// 广播消息
func (pot *Pot) broadcast(msg *pb.Message) error {

}