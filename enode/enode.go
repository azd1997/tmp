package enode

import (
	"fmt"
	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/enode/bc"
	"github.com/azd1997/ecoin/p2p"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/protocol/raw"
	"github.com/azd1997/ecoin/protocol/view"
	"sort"
)

type Config struct {
	Node         p2p.Node	// p2p 节点
	Account       *account.Account	// 代表账户的私钥(有了私钥和橘色编号可以得到完整账户)

	*bc.Config
}

// ENode 总结构，代表了ecoin系统中的一个节点
type Enode struct {

	// account 账户
	acc *account.Account
	// id 节点ID以及用户ID
	id crypto.ID

	// chain 区块链
	chain *bc.Chain

	// 交易池
	tp *txPool

	// 证明池
	pp *proofPool

	// 网络模块
	net *net

	// 查询缓存
	qc *qCache

	// potCompetitor。共识竞争器。轮询/同步区块；构造区块
	pc *potCompetitor

	// workerNode 工人节点 账户角色为A类账户，则有义务承担worker的职责，负责出块
	workerNode bool
}

func NewEnode(conf *Config) *Enode {
	// chain
	chain := bc.NewChain()
	if err := chain.Init(conf.Config); err != nil {
		logger.Fatal("init enode module failed: %v\n", err)
	}
	chain.Start()

	// txPool
	txPool := newTxPool(conf.Account)

	// proofPool
	proofPool := NewProofPool()

	// net
	network := newNet(conf.Node, chain, txPool, conf.Account.RoleNo)
	// txPool设置
	txPool.setBroadcastChan(network.txsToBroadcast)
	// network启动
	network.start()
	// txPool启动
	txPool.start()

	// queryCache
	queryCache := newQCache(chain)

	// potCompetitor
	var pot *potCompetitor
	workerNode := false
	if role.IsARole(conf.Account.RoleNo) {
		logger.Info("the enode instance is running with a worker ID\n")
		pot = newPotCompetitor(txPool, chain, network,
			crypto.PrivateKey2ID(conf.Account.PrivateKey, conf.Account.RoleNo))
		pot.start()
		workerNode = true
	} else {
		logger.Info("the enode instance is running with a non-worker ID\n")
	}

	return &Enode{
		acc:conf.Account,
		id:crypto.PrivateKey2ID(conf.Account.PrivateKey, conf.Account.RoleNo),
		chain:chain,
		tp:txPool,
		pp: proofPool,
		qc:queryCache,
		net:network,
		pc:pot,
		workerNode:workerNode,
	}
}

func (en *Enode) Stop() {
	// 对于工人节点，首先关闭竞争器
	if en.workerNode {
		en.pc.stop()
	}
	// 交易池关闭
	en.tp.stop()
	// 证明池关闭
	en.pp.stop()
	// p2p网络模块关闭
	en.net.stop()
	// 区块链关闭
	en.chain.Stop()
}

/////////////////////////////////////////////////////

// 构建交易
func (en *Enode) BuildTxByRaw(rtxs []*raw.Tx) error {
	// TODO 检查rtxs的每一个rtx格式

	en.tp.addRawTx(rtxs)
	return nil
}

func (en *Enode) BuildTx(txs []*core.Tx) error {
	for _, tx := range txs {
		if err := en.chain.VerifyTx(tx); err != nil {
			if _, ok := err.(bc.ErrTxAlreadyExist); ok {
				return err
			}
			return fmt.Errorf("verify tx failed: [%s]", tx.String())
		}
	}

	en.tp.addTx(txs, false)	// 自己上传，当然不是来自广播
	return nil
}

// 查询交易
func (en *Enode) QueryTx(hexHashes []string) []*view.TxInfo {
	return en.qc.getTx(hexHashes)
}

// 查询账户
func (en *Enode) QueryAccount(id crypto.ID) ([]crypto.Hash, []crypto.Hash, uint64, int64) {
	return en.qc.getAccountInfo(id)
}

// 查询区块
func (en *Enode) QueryBlockViaHeights(heights []uint64) []*view.BlockInfo {
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] > heights[j] // 从高到低
	})

	var result []*view.BlockInfo
	for _, height := range heights {
		info := en.qc.getBlockViaHeight(height)
		if info != nil {
			result = append(result, info)
		}
	}

	return result
}

func (en *Enode) QueryLatestBlock() *view.BlockInfo {
	return en.qc.getLatestBlock()
}

func (en *Enode) QueryBlockViaRange(begin, end uint64) []*view.BlockInfo {
	var result []*view.BlockInfo
	for i := end; i >= begin; i-- {
		info := en.qc.getBlockViaHeight(i)
		if info != nil {
			result = append(result, info)
		}
	}

	return result
}

func (en *Enode) QueryBlockViaHash(hexHashes []string) []*view.BlockInfo {
	var result []*view.BlockInfo
	for _, h := range hexHashes {
		info := en.qc.getBlockViaHash(h)
		if info != nil {
			result = append(result, info)
		}
	}

	return result
}