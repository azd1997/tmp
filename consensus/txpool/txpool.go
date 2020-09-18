/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/9/8 9:59
* @Description: The file is for
***********************************************************************/

package txpool

import (
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/protocol/core"
)

type TxPool interface {
	// PrepareTxs 准备好要出块的交易列表，及其证明
	// 对于具体实现来说，其实就是将ubtxp倒到tbtxp，然后求MerkleHash，返回以作证明
	PrepareTxs() crypto.Hash
}

type poolImpl struct {
	ubtxp Pool
	tbtxp Pool
	uctxp Pool
}


/////////////////// 池结构 ///////////////////////
// 可以是数组或者是其他高级结构

type Pool interface {
	Append(txs ...core.Tx) error
	RemoveAll() []core.Tx

}

type ArrayPool struct {

}

type MerklePool struct {

}
