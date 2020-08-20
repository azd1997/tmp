/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/3 0:02
* @Description: The file is for
***********************************************************************/

package enode

import (
	"strings"
	"sync"
	"time"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/enode/bc"
	"github.com/azd1997/ecoin/protocol/view"
	"github.com/azd1997/ecoin/store/db"
)

// qCache 缓存所有未持久化的区块数据
// 它和Chain结构中的区块缓存不同，这里用作查询，只读
type qCache struct {
	c *bc.Chain

	// all the map keys are upper case
	blockInfos      map[string]*view.BlockInfo		// <hex(hash), *BlockInfo>
	txInfos         map[string]*view.TxInfo			// <hex(hash), *TxInfo>
	accountInfos    map[crypto.ID]*view.AccountInfo		// <ID, *AccountInfo>
	sortedBlocks    *sortedBlocks
	lastRefreshTime time.Time
	refreshLock     sync.Mutex
}

// 这些哈希表等结构，用时再初始化
func newQCache(c *bc.Chain) *qCache {
	return &qCache{
		c: c,
	}
}

type sortedBlocks struct {
	blocks []*view.BlockInfo
	begin  uint64
	end    uint64
}

func (qc *qCache) getBlockViaHash(hexHash string) *view.BlockInfo {
	qc.refresh()
	blocks := qc.blockInfos
	if v, ok := blocks[strings.ToUpper(hexHash)]; ok {
		return v
	}
	return nil
}

func (qc *qCache) getBlockViaHeight(height uint64) *view.BlockInfo {
	// 先刷新
	qc.refresh()

	// 查看是否在缓存中有
	sbs := qc.sortedBlocks
	if height >= sbs.begin && height <= sbs.end {
		diff := sbs.end - height
		return sbs.blocks[diff]
	}

	// 缓存未命中，查询数据库
	cb, _, err := db.GetBlockViaHeight(height)
	if err != nil {
		return nil
	}


	return &view.BlockInfo{
		Block:     cb,
		Height:    height,
	}
}

// 获取最新区块
func (qc *qCache) getLatestBlock() *view.BlockInfo {
	// 刷新缓存
	qc.refresh()

	// 查询缓存
	sbs := qc.sortedBlocks
	if len(sbs.blocks) != 0 {
		return sbs.blocks[0]
	}

	// 数据库查找
	latestHeight, err := db.GetLatestHeight()
	if err != nil {
		return nil
	}
	latestBlock, _, err := db.GetBlockViaHeight(latestHeight)
	if err != nil {
		return nil
	}

	return &view.BlockInfo{
		Block:     latestBlock,
		Height:    latestHeight,
	}
}

// 根据十六进制哈希字符串列表，查找交易信息
func (qc *qCache) getTx(hexHashes []string) []*view.TxInfo {
	// 刷新缓存
	qc.refresh()

	cacheTxs := qc.txInfos

	var result []*view.TxInfo
	for _, hash := range hexHashes {
		// 缓存中查找
		if e, ok := cacheTxs[strings.ToUpper(hash)]; ok {
			result = append(result, e)
			continue
		}

		// 数据库查找
		h, err := encoding.FromHex(hash)
		if err != nil {
			continue
		}
		tx, height, err := db.GetTxViaHash(h)
		if err != nil {
			continue
		}
		header, blockHash, err := db.GetHeaderViaHeight(height)
		if err != nil {
			continue
		}

		result = append(result, &view.TxInfo{
			Tx:tx,
			Height:    height,
			BlockHash: blockHash,
			BlockTime:      header.Time,
		})
	}
	return result
}

// 获取账户信息：账户相关的交易与账户余额
func (qc *qCache) getAccountInfo(id crypto.ID) ([]crypto.Hash, []crypto.Hash, uint64, int64) {
	// 刷新缓存
	qc.refresh()

	cacheAccounts := qc.accountInfos

	var txsFrom, txsTo []crypto.Hash
	fromAdded, toAdded := make(map[string]bool), make(map[string]bool)	// 用于去重
	balance := uint64(0)

	// 缓存中查询
	if acc, ok := cacheAccounts[id]; ok {
		txsFrom = append(txsFrom, acc.TxsHashFrom...)
		for _, v := range acc.TxsHashFrom {
			fromAdded[encoding.ToHex(v)] = true
		}
		txsTo = append(txsTo, acc.TxsHashTo...)
		for _, v := range acc.TxsHashTo {
			toAdded[encoding.ToHex(v)] = true
		}
		balance = acc.Balance
	}

	// 数据库中查找. 同时去除与缓存中重复的数据
	if txsHashFrom, _, err := db.GetTxFromHashesViaID(id); err == nil {
		for _, hash := range txsHashFrom {
			if !fromAdded[encoding.ToHex(hash)] {
				txsFrom = append(txsFrom, hash)
			}
		}
	}
	if txsHashTo, _, err := db.GetTxFromHashesViaID(id); err == nil {
		for _, hash := range txsHashTo {
			if !toAdded[encoding.ToHex(hash)] {
				txsTo = append(txsTo, hash)
			}
		}
	}

	// TODO： Balance的处理逻辑
	if dbBalance, err := db.GetBalanceViaID(id); err == nil {
		balance += dbBalance
	}

	return txsFrom, txsTo, balance, 0	// TODO: 信誉分
}

// 刷新查询缓存状态
func (qc *qCache) refresh() {
	qc.refreshLock.Lock()
	defer qc.refreshLock.Unlock()

	const refreshInterval = 20 * time.Second
	now := time.Now()
	// 时间到，刷新
	if now.Sub(qc.lastRefreshTime) > refreshInterval {
		// 获取未存储的区块
		blocks, heights := qc.c.GetUnstoredBlocks()

		latestSortedBlocks := &sortedBlocks{}
		latestAccounts := make(map[crypto.ID]*view.AccountInfo)
		latestTxs := make(map[string]*view.TxInfo)
		latestBlocks := make(map[string]*view.BlockInfo)

		if len(blocks) != 0 {
			latestSortedBlocks.end = heights[0]
			latestSortedBlocks.begin = heights[len(heights)-1]
		}

		for i := 0; i < len(blocks); i++ {
			h := blocks[i].Hash

			blockInfo := &view.BlockInfo{
				Block:     blocks[i],
				Height:    heights[i],
			}
			latestBlocks[encoding.ToHex(h)] = blockInfo
			latestSortedBlocks.blocks = append(latestSortedBlocks.blocks, blockInfo)

			for _, tx := range blocks[i].Txs {
				latestTxs[encoding.ToHex(tx.Id)] = &view.TxInfo{
					Tx:  tx,
					Height:    heights[i],
					BlockHash: h,
					BlockTime:      blocks[i].Time,
				}

				id := tx.From
				var account *view.AccountInfo
				var ok bool
				if account, ok = latestAccounts[id]; !ok {
					account = &view.AccountInfo{}
					latestAccounts[id] = account
				}
				account.TxsHashFrom = append(account.TxsHashFrom, tx.Id)
			}

			minerID := blocks[i].CreateBy
			var miner *view.AccountInfo
			var ok bool
			if miner, ok = latestAccounts[minerID]; !ok {
				miner = &view.AccountInfo{}
				latestAccounts[minerID] = miner
			}
			miner.Balance++
		}

		qc.sortedBlocks = latestSortedBlocks
		qc.accountInfos = latestAccounts
		qc.txInfos = latestTxs
		qc.blockInfos = latestBlocks
		qc.lastRefreshTime = now
	}

	return
}

