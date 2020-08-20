package bc

import (
	"fmt"
	"sync"

	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/protocol/core"
)

// 这里的Block结构用于程序内部方便查询，构成既具有双链表特性又具有多叉树特性的结构
type block struct {
	*core.Block
	height uint64	// 区块高度
	stored bool		// 该区块是否已被存储到区块链存储(BadgerDB/blocks.db)中

	// 之前的区块，或者称父区块，只有一个
	prev *block
	// 接下来的区块，或者称子区块，有若干个
	nexts sync.Map		// 如果考虑分叉的话，Next需要使用sync.Map<blockhash, *Block>
}


func newBlock(b *core.Block, height uint64, stored bool) *block {
	return &block{
		Block:  b,
		height: height,
		stored: stored,
	}
}

// 区块构建时间
func (b *block) time() int64 {
	return b.Time
}

// 是否已存储到数据库中
func (b *block) isStored() bool {
	return b.stored
}

// 设置父区块
func (b *block) setPrev(prev *block) {
	b.prev = prev
}

// 移除对父区块的引用
func (b *block) removePrev() {
	b.prev = nil
}

// 添加子区块
func (b *block) addNext(next *block) {
	key := encoding.ToHex(next.Hash)
	b.nexts.Store(key, next)
}

// 移除对某个子区块的引用
func (b *block) removeNext(next *block) {
	key := encoding.ToHex(next.Hash)
	b.nexts.Delete(key)
}

// 是否是cb的父区块
func (b *block) isPrevOf(cb *core.Block) bool {
	key := encoding.ToHex(cb.Hash)
	_, ok := b.nexts.Load(key)
	return ok
}

// 子区块的数量
func (b *block) nextsNum() int {
	result := 0
	b.nexts.Range(func(k, v interface{}) bool {
		result++
		return true
	})
	return result
}

// 当一个区块没有子区块时，又触发了删除条件（注意：必须是没有子区块的时候才能删除）
// 这时需要将该区块从该区块的父区块的子区块列表移除，并且返回其父区块
func (b *block) remove() (*block, error) {
	if b.nextsNum() != 0 {
		return nil, fmt.Errorf("fordward reference is not zero, can't be removed")
	}

	prev := b.prev
	prev.removeNext(b)
	b.removePrev()
	return prev, nil
}

