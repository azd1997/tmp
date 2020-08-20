/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/1 17:14
* @Description: The file is for
***********************************************************************/

package bc

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/enode/bc/merkle"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/store/db"
)

// branch用来处理分叉
// 加入从第100号区块开始产生了三条分叉，那么会产生三个branch
//

type branch struct {
	// 分支的末端位置
	head *block
	// 分支的起始位置
	tail *block

	// 交易缓存
	txCache sync.Map // <hex(tx.Id), *core.Tx>
	// 区块缓存
	blockCache sync.Map // <hex(block.Hash), *block>
}

// 新建分支
func newBranch(begin *block) *branch {
	result := &branch{
		head: begin,
		tail: begin,
	}

	iter := begin
	for {
		bKey := encoding.ToHex(iter.Hash)
		result.blockCache.Store(bKey, iter)

		for _, tx := range iter.Txs {
			txKey := encoding.ToHex(tx.Id)
			result.txCache.Store(txKey, tx)
		}

		iter = iter.prev
		// 当向前迭代至空时则停止
		// 这是因为区块链会分时段处理分叉，
		// 并将确定的部分写入数据库，从这里（内存）删去
		// 因此当前溯至空时，就说明到了分支起点
		if iter == nil {
			break
		}
	}

	return result
}

// 不再具有Next区块（子区块）的分支应该将其所有区块删除，删除之后该分支不再使用
// eg.
// A --> B --> C --> D           (main branch)
//             | --> E --> F     (fork branch)
// if the fork branch call remove(), then the C will not point to E, the E will not point to F,
// but the C still points to D
func (b *branch) remove() {
	iter := b.head
	var err error
	for {
		iter, err = iter.remove()
		if err != nil {
			break
		}

		if iter == nil {
			break
		}
	}
}

// 分支末端添加新区块
func (b *branch) add(newBlock *block) error {
	oldHead := b.head
	oldHead.addNext(newBlock)

	newBlock.setPrev(oldHead)
	nbKey := encoding.ToHex(newBlock.Hash)
	b.head = newBlock
	b.blockCache.Store(nbKey, newBlock)

	for _, tx := range newBlock.Txs {
		key := encoding.ToHex(tx.Id)
		b.txCache.Store(key, tx)
	}

	return nil
}

// 分支的哈希标识，取的是分支末端的区块哈希
func (b *branch) hash() crypto.Hash {
	return b.head.Hash
}

// 分支高度
func (b *branch) height() uint64 {
	return b.head.height
}

// 在该分支搜索区块
func (b *branch) getBlock(hash crypto.Hash) *block {
	bKey := encoding.ToHex(hash)
	v, ok := b.blockCache.Load(bKey)
	if ok {
		b := v.(*block)
		return b
	}

	return nil
}

// 在该分支搜索交易
func (b *branch) getTx(hash crypto.Hash) *core.Tx {
	eKey := encoding.ToHex(hash)
	v, ok := b.txCache.Load(eKey)
	if ok {
		tx := v.(*core.Tx)
		return tx
	}
	return nil
}

// 根据该分支，以及已经存储（固化）的过往区块链，校验新来的区块是否合法
func (b *branch) verifyBlock(cb *core.Block) error {
	// 检查区块结构的有效性
	if err := cb.Verify(); err != nil {
		return fmt.Errorf("block struct verify failed:%v", err)
	}

	// 根据区块链上下文（该分支及已固化的区块链）检查区块

	// 1. time
	t := time.Unix(cb.Time, 0)
	if t.Sub(time.Now()) > 3*time.Second {
		return fmt.Errorf("invalid future time")
	}
	if t.Before(time.Unix(b.head.time(), 0)) {
		return fmt.Errorf("invalid past time")
	}

	// 2. PotMsg Proof

	// 3. last block hash
	if !bytes.Equal(b.hash(), cb.PrevHash) {
		return fmt.Errorf("mismatch last hash")
	}

	// 4. pot

	// 5. tx
	if cb.IsEmptyMerkleRoot() {
		return nil
	}

	var leafs merkle.MerkleLeafs
	for _, tx := range cb.Txs {
		if err := b.verifyTx(tx); err != nil {
			return err
		}
		leafs = append(leafs, tx.Id)
	}

	root, _ := merkle.ComputeRoot(leafs)
	if !bytes.Equal(root, cb.MerkleRoot) {
		return fmt.Errorf("mismatch merkle root")
	}

	return nil
}

// 根据该分支和已固化的区块链去检查交易的有效性
// TODO
func (b *branch) verifyTx(tx *core.Tx) error {
	// 检查交易结构的有效性
	if err := tx.Verify(); err != nil {
		return fmt.Errorf("tx struct verify failed:%v", err)
	}

	// 根据该分支和已固化的区块链深度检查该交易

	// 该分支的交易缓存
	if v := b.getTx(tx.Id); v != nil {
		return ErrTxAlreadyExist{tx.Id}
	}

	// 数据库（已固化区块链）检查
	if db.HasTx(tx.Id) {
		return ErrTxAlreadyExist{tx.Id}
	}

	return nil
}

// TODO
func (b *branch) potCheck() bool {
	return true
}

// 从缓存移除某区块及其所包含交易
func (b *branch) removeFromCache(rmBlock *block) {
	bKey := encoding.ToHex(rmBlock.Hash)
	b.blockCache.Delete(bKey)

	for _, tx := range rmBlock.Block.Txs {
		txKey := encoding.ToHex(tx.Id)
		b.txCache.Delete(txKey)
	}
}

func (b *branch) String() string {
	var result string
	iter := b.head
	for {
		if iter == nil {
			break
		}
		result += fmt.Sprintf("%X(%d)->",
			iter.Hash[crypto.HASH_LENGTH-2:], iter.height)
		iter = iter.prev
	}
	result += "..."

	blocksSize := 0
	b.blockCache.Range(func(key interface{}, value interface{}) bool {
		blocksSize++
		return true
	})
	txsSize := 0
	b.txCache.Range(func(key interface{}, value interface{}) bool {
		txsSize++
		return true
	})
	cacheInfo := fmt.Sprintf("cache info: %d blocks, %d evidences", blocksSize, txsSize)
	result = fmt.Sprintf("%s\n%s", result, cacheInfo)

	return result
}
