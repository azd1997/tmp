/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/7 12:35
* @Description: The file is for
***********************************************************************/

package view

import (
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/protocol/core"
)

type BlockInfo struct {
	*core.Block
	Height    uint64
}

type TxInfo struct {
	*core.Tx
	Height    uint64
	BlockHash crypto.Hash
	BlockTime      int64	// 区块构建时间（交易构建时间在core.Tx有）
}

type AccountInfo struct {
	TxsHashFrom []crypto.Hash	// 由该账户构建的交易
	TxsHashTo []crypto.Hash	// 由该账户接收的交易
	Balance    uint64
	// TODO: Credit等
}
