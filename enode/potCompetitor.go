/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/3 0:35
* @Description: The file is for
***********************************************************************/

package enode

import (
	"fmt"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/enode/bc"
	"github.com/azd1997/ecoin/enode/bc/merkle"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ego/epattern"
	"time"
)

// scheduler 负责循环 参与POT竞争，尝试出块
// scheduler 只对worker节点生效
// 其从本地区块链以及交易池收集信息，参与竞争并尝试出块
// 当POT获胜时将区块广播出去
type potCompetitor struct {

	HalfEP int64	// 半个出块周期 ns

	potStart *time.Timer	// pot竞争开始的定时器
	potEnd *time.Timer		// pot竞争结束的定时器
	waitBlockTimeout *time.Timer	// 等待新区块的超时定时器

	tbtxp []*core.Tx	// 待出块的交易池
	selfProof *core.PoTProof

	proofs map[string]*core.PoTProof	// 证明列表
	recvProof chan *core.PoTProof	// 从net模块接收网络中得到的证明消息
	winnerProof *core.PoTProof

	winnerHistory map[string]*core.PoTProof		// 历史证明，键为hex(base)
	waitingBlockProof *core.PoTProof	// 等待的区块

	txPool    *txPool
	chain   *bc.Chain
	network *net
	workerID crypto.ID

	lm *epattern.LoopMode
}


func newPotCompetitor(p *txPool, c *bc.Chain,
	n *net, workerID crypto.ID) *potCompetitor {
	pc := &potCompetitor{
		txPool:    p,
		chain:   c,
		network: n,
		workerID: workerID,
		lm:      epattern.NewLoop(1),
	}

	pc.recvProof = pc.network.potProofCollect	// 设置接收通道

	return pc
}

func (pc *potCompetitor) start() {
	go pc.compete()	// 启动竞争循环
	pc.lm.StartWorking()
}

func (pc *potCompetitor) stop() {
	pc.lm.Stop()
}

// 循环参与POT共识竞争
func (pc *potCompetitor) compete() {
	pc.lm.Add()
	defer pc.lm.Done()

	select {

	case <-pc.network.InitFinishC:	// 网络模块初始化完成（本地区块链已达网络最新状态）
		logger.Info("blocks sync finished, start mining...")
		// 初始化完成后，定时器开始设置，时间一到就广播PoT证明消息，参与共识
		t1 := time.Unix(0, pc.chain.GetLatestBlockTime() + pc.HalfEP)
		t2 := t1.Add(time.Duration(pc.HalfEP) * time.Nanosecond)
		t3 := t1.Add(time.Duration(2 * pc.HalfEP) * time.Nanosecond)
		// 这里注意time.AfterFunc会将注册的函数在定时器到期后，在单独的goroutine中执行
		// 由于这里PoT竞争器部分应尽量避免并发，保证时间的串行，所以不采用这种做法
		//pc.potStart = time.AfterFunc(
		//	t1.Sub(time.Now()), pc.genAndBroadcastPoTProof)
		//pc.potEnd = time.AfterFunc(
		//	t2.Sub(time.Now()), pc.judgeCompetitionAndHandle)
		//pc.waitBlockTimeout = time.AfterFunc(
		//	t3.Sub(time.Now()), pc.waitNewBlock)

		pc.potStart = time.NewTimer(t1.Sub(time.Now()))
		pc.potEnd = time.NewTimer(t2.Sub(time.Now()))
		pc.waitBlockTimeout = time.NewTimer(t3.Sub(time.Now()))
	case <-pc.lm.D:
		logger.Info("program is terminated before blocks sync finished")
		return
	}

	newRound := false
	for {

		select {
		case <-pc.lm.D:
			logger.Info("stop competing and exist")
			return
		case lastestBlockTime := <-pc.chain.PassiveChangeNotify:
			// 在PoT的竞争阶段已经接收到了新区块：可能是由于本地节点网速过慢导致
			logger.Debug("terminate competing, start next turn\n")
			// 归还交易
			pc.returnTxs()
			newRound = true		// 接收到新区块则将之置为true
			// 更新定时器
			pc.potStart.Reset(time.Unix(0, lastestBlockTime + pc.HalfEP).Sub(time.Now()))
			pc.potEnd.Stop()
			pc.waitBlockTimeout.Stop()
		case <-pc.potStart.C:				// pot竞争开始
			pc.genAndBroadcastPoTProof()
		case <-pc.potEnd.C:					// pot竞争结束
			pc.judgeCompetitionAndHandle()
		case <-pc.waitBlockTimeout.C:
			if !newRound {	// 在超时时间到时仍未收到新区块（开启下一轮）
				// 重新开始新round
				// 惩罚原winner
				newRound = true
				pc.waitWinnerBlockTimeout()

			}
		case proof := <- pc.recvProof:		// 接收到新证明
			pc.proofs[proof.From.ToHex()] = proof
			if pc.winnerProof == nil {
				pc.winnerProof = proof
			} else if proof.GreaterThan(pc.winnerProof) {
				pc.winnerProof = proof
			}
		}
	}
}

// 从交易池取出有效交易
func (pc *potCompetitor) getTxs() {
	txs := make(map[string]*core.Tx)
	//txSize := 0

	// TODO: 总大小限制。暂时不限制

	//for txSize < params.BlockSize {
	//	tx := pc.txPool.nextTx()
	//	if tx == nil {
	//		break
	//	}
	//
	//	if err := pc.chain.VerifyTx(tx); err == nil {
	//		// exclude the same tx
	//		txs[encoding.ToHex(tx.Id)] = tx
	//		txSize += tx.Size()		// tx采取了gob编码，因此没办法预估大小
	//	}
	//}

	for pc.txPool.txsSize() > 0 {
		tx := pc.txPool.nextTx()
		if tx == nil {
			break
		}

		if err := pc.chain.VerifyTx(tx); err == nil {
			// exclude the same tx
			txs[encoding.ToHex(tx.Id)] = tx
		}
	}

	var result []*core.Tx
	for _, tx := range txs {
		result = append(result, tx)
	}
	pc.tbtxp = result
}

// 向交易池归还交易（POT竞争失败）
func (pc *potCompetitor) returnTxs()  {
	pc.txPool.addTx(pc.tbtxp, false)
	pc.tbtxp = nil
	pc.selfProof = nil
}

// 传入得交易列表不含coinbase交易
func (pc *potCompetitor) genBlock() *core.Block {
	if len(pc.tbtxp) == 0 {
		return pc.genEmptyBlock()
	}

	// TODO：coinbase也需要增加签名项
	// 构造coinbase交易
	coinbase := core.NewTx(
		core.TX_COINBASE,
		crypto.ZeroID,		// ZeroID不能被个人使用，一方面作为判空条件，一方面作为发币来源
		pc.workerID,
		100,			// amount之后需要进行定义，暂且写100
		nil,
		crypto.ZeroHash,	// 作为哈希的零值
		0,
		[]byte(fmt.Sprintf("THIS IS COINBASE FOR [%s]", pc.workerID.ToHex())),
		)
	txs := append([]*core.Tx{coinbase}, pc.tbtxp...)

	// 构造区块
	header := core.NewBlockHeaderV1(pc.chain.LatestBlockHash(), pc.workerID, pc.selfProof.TxsMerkle)
	block := core.NewBlock(header, txs)

	return block
}

func (pc *potCompetitor) genEmptyBlock() *core.Block {
	// TODO：coinbase也需要增加签名项
	// 构造coinbase交易
	coinbase := core.NewTx(
		core.TX_COINBASE,
		crypto.ZeroID,		// ZeroID不能被个人使用，一方面作为判空条件，一方面作为发币来源
		pc.workerID,
		100,			// amount之后需要进行定义，暂且写100
		nil,
		crypto.ZeroHash,	// 作为哈希的零值
		0,
		[]byte(fmt.Sprintf("THIS IS COINBASE FOR [%s]", pc.workerID.ToHex())),
	)

	// 构造区块
	header := core.NewBlockHeaderV1(pc.chain.LatestBlockHash(), pc.workerID, core.EmptyMerkleRoot)
	block := core.NewBlock(header, []*core.Tx{coinbase})

	return block
}

func (pc *potCompetitor) genAndBroadcastPoTProof() {
	// 1. 收集交易
	pc.getTxs()
	// 2. 计算默克尔根
	var txLeafs merkle.MerkleLeafs
	for _, tx := range pc.tbtxp {
		txLeafs = append(txLeafs, tx.Id)
	}
	txRoot, _ := merkle.ComputeRoot(txLeafs)
	// 3. 自己的pot证明
	pc.selfProof = &core.PoTProof{
		From:pc.workerID,
		TxsNum:uint32(len(pc.tbtxp)),
		TxsMerkle:txRoot,
		Base:pc.chain.LatestBlockHash(),
	}
	// 4. 通过网络模块发送出去
	pc.network.potProof <- pc.selfProof

	logger.Debug("start competing... my proof is: [%d|%x|%x]\n",
		pc.selfProof.TxsNum, pc.selfProof.TxsMerkle, pc.selfProof.Base)
}

func (pc *potCompetitor) judgeCompetitionAndHandle() bool {
	if pc.selfProof.From == pc.winnerProof.From {
		logger.Debug("end competing, I win... my proof is: [%d|%x|%x]\n",
			pc.selfProof.TxsNum, pc.selfProof.TxsMerkle, pc.selfProof.Base)

		// 自己出块
		b := pc.genBlock()
		pc.network.potWinnerBlock <- b	// 发给网络模块
		// 清理掉临时状态
		pc.winnerHistory[encoding.ToHex(pc.winnerProof.Base)] = pc.winnerProof
		pc.winnerProof = nil
		pc.selfProof = nil
		pc.proofs = make(map[string]*core.PoTProof)

		return true
	} else {
		logger.Debug("end competing, I lose... winner proof is: [%d|%x|%x]\n",
			pc.winnerProof.TxsNum, pc.winnerProof.TxsMerkle, pc.winnerProof.Base)
		// 别人出块，等待这个区块
		pc.waitingBlockProof = pc.winnerProof
		pc.winnerHistory[encoding.ToHex(pc.winnerProof.Base)] = pc.winnerProof
		pc.winnerProof = nil
		pc.selfProof = nil
		pc.proofs = make(map[string]*core.PoTProof)
		// 归还交易
		pc.returnTxs()

		return false
	}
}

func (pc *potCompetitor) waitWinnerBlockTimeout() {
	// 重新开始新round


	// 惩罚原winner
}



