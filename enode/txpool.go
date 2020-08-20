/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/2 9:28
* @Description: The file is for
***********************************************************************/

package enode

import (
	"container/heap"
	"github.com/azd1997/ecoin/protocol/raw"
	"sync"
	"time"

	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ego/epattern"
)

const txsCacheSize  = 1024


// 带权重的交易
// 考虑当交易量升到比较大时，如果仍是固定间隔出块
// 那么块的体积会越来越大，这会不断增大单个块的传输时延
// 最终会影响POT的策略
// 考虑交易排队，权重高者优先出块
// 权重考虑因素目前考虑：
// TODO: 交易类型 * factor + 交易时间（越早越大）
// 假设交易时间单位s，考虑其他类型交易与诊断类交易相比
// 要落后1分钟，那么交易类型 * 60 +交易时间权重　为总体权重
// 权重大者先打包
// TODO: 使用优先队列
type weightedTx struct {
	*core.Tx
}

type txPool struct {
	acc *account.Account
	raws      chan *raw.Tx
	txs *txsPriorityQueue	// UBTXP 相对应的TBTXP是POT竞争时临时出现的

	txsLock sync.RWMutex
	broadcast chan<- []*core.Tx
	lm *epattern.LoopMode
}

func newTxPool(acc *account.Account) *txPool {
	tp := &txPool{
		acc : acc,
		raws: make(chan *raw.Tx, txsCacheSize),
		txs:new(txsPriorityQueue),
		lm:   epattern.NewLoop(1),
	}
	return tp
}

func (tp *txPool) setBroadcastChan(c chan<- []*core.Tx) {
	tp.broadcast = c
}

func (tp *txPool) start() {
	go func() {
		tp.lm.Add()
		defer tp.lm.Done()
		for {
			select {
			case <-tp.lm.D:
				return
			case rawTx := <-tp.raws:
				// TODO: 处理原始交易，调用core.Tx的协议方法，转为core.Tx
				tp.processRaw(rawTx) // TODO
			}
		}
	}()

	tp.lm.StartWorking()
}


func (tp *txPool) stop() {
	tp.lm.Stop()
}

func (tp *txPool) addRawTx(txs []*raw.Tx) {
	for _, tx := range txs {
		select {
		case tp.raws <- tx:
		default:
			logger.Warn("tx raw queue is full, drop raw tx %X",
				tx.Hash())
		}
	}
}


func (tp *txPool) addTx(txs []*core.Tx, fromBroadcast bool) {
	// 将txs插入到优先队列中，排队
	for _, tx := range txs {
		tp.insert(&weightedTx{tx})
	}

	// 如果不是来自广播的交易，那么需要将其广播出去
	if !fromBroadcast {
		tp.broadcast <- txs
	}
}


// 从交易池的交易队列取出优先级最高的一个交易，如果没有则返回nil
func (tp *txPool) nextTx() *core.Tx {
	tp.txsLock.Lock()
	defer tp.txsLock.Unlock()

	if tp.txs.len() == 0 {
		return nil
	}

	return tp.txs.pop().Tx
}

func (tp *txPool) txsSize() int {
	tp.txsLock.RLock()
	defer tp.txsLock.RUnlock()

	return tp.txs.len()
}

// TODO
func (tp *txPool) processRaw(raw *raw.Tx) {
	// 1. 检查raw是否引用了一个未完成交易
		// 1.1 如果没有，直接往下边走
		// 1.2 如果有，需要不断地找到交易链路，核查下交易链路的正确性（应该认为之前的交易链路是正确的）


	// 2. 新建core.Tx
	tx := core.NewTx(raw.Type, tp.acc.UserId(), raw.To, raw.Amount, raw.Payload, raw.PrevTxId, raw.Uncompleted, raw.Description)
	if err := tx.Sign(tp.acc.PrivateKey); err != nil {
		logger.Warn("sign tx failed:%v\n", err)
		return
	}

	// 3.加入本地的交易队列，并且广播出去
	tp.txs.push(&weightedTx{tx})
	select {
	case tp.broadcast <- []*core.Tx{tx}: 	// 推到广播队列去
	default:
		logger.Warn("tx ask to broadcast failed\n")
	}
}

func (tp *txPool) insert(wtx *weightedTx) {
	tp.txsLock.Lock()
	defer tp.txsLock.Unlock()

	if tp.txs.len() >= txsCacheSize {
		return
	}
	tp.txs.push(wtx)
}

/////////////////////////////////////////////

// 优先队列

type txsPriorityQueue []*weightedTx

// Less函数。
// NOTICE: 调整权重设计需要修改此处
// TODO: 是否引入交易费机制参与排队，这需要在core.Tx增加字段
func (q *txsPriorityQueue) Less(i, j int) bool {
	weightI := txTypeWeight[(*q)[i].Type] * weightFactor + int(time.Now().Unix() - (*q)[i].TimeUnix)
	weightJ := txTypeWeight[(*q)[j].Type] * weightFactor + int(time.Now().Unix() - (*q)[j].TimeUnix)
	return weightI > weightJ
}

func (q *txsPriorityQueue) Len() int {
	return len(*q)
}

func (q *txsPriorityQueue) Swap(i, j int) {
	(*q)[i], (*q)[j] = (*q)[j], (*q)[i]
}

func (q *txsPriorityQueue) Push(wtx interface{}) {
	*q = append(*q, wtx.(*weightedTx))
}

func (q *txsPriorityQueue) Pop() (wtx interface{}) {
	wtx, *q = (*q)[len(*q)-1], (*q)[:len(*q)-1]
	return
}

// 上面的Push/Pop是为了实现heap.Interface接口，为了方便使用，定义以下方法进行包装

// init 对现有的q数组进行堆化(heapify)
func (q *txsPriorityQueue) init() {
	heap.Init(q)
}

func (q *txsPriorityQueue) push(wtx *weightedTx) {
	heap.Push(q, wtx)
}

func (q *txsPriorityQueue) pop() *weightedTx {
	return heap.Pop(q).(*weightedTx)
}

func (q *txsPriorityQueue) peek() *weightedTx {
	return (*q)[0]
}

// 为了保持这些方法的一致性，尽管有个Len()，还是再搞个len()方法
// 这样使用该队列只使用小写字母开头的方法
func (q *txsPriorityQueue) len() int {
	return len(*q)
}


//////////////////////////////////////////////

// 交易类型 - 权重 映射表

// 这里假定

var txTypeWeight = map[uint8]int{
	// coinbase有些特殊，他是创块时生成的，其实并不会出现在优先队列中
	core.TX_COINBASE: 0,		// coinbase交易，用于给出块者发放奖励
	core.TX_GENERAL:0,		// 通用转账交易
	core.TX_R2P:0,			// 研究机构->病人的数据购买请求交易
	core.TX_P2R:0,			// 病人->研究机构的数据购买交易回复
	core.TX_P2H:1,			// 病人->医院的诊断请求交易
	core.TX_H2P:1,			// 医院->病人的诊断回复交易
	core.TX_P2D:1,			// 病人->医生的诊断请求交易
	core.TX_D2P:1,			// 医生->病人的诊断回复交易
	core.TX_ARBITRATE:0,	// 仲裁交易纠纷用
	core.TX_UPLOAD:0,		// 上传数据摘要用
	core.TX_REGREQ:0,		// 注册账号用	register request
	core.TX_REGRESP:0,		// 注册账号用 register response
}

const weightFactor = 120	// 120相当于2min