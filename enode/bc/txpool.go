package bc

import (
	"sync"

	"github.com/azd1997/ecoin/account"
)

// Pool 交易池(tx.Pool)
type TxPool struct {
	*UBTXP
	*TBTXP
	*UCTXP
}

// UBTXP UnBlocked Transaction Pool 未出块交易池
// (UnBlocked指还未包含在区块链)
// UBTXP是传统意义的交易缓存池
type UBTXP struct {
	self *account.Account	// 自己的账户



	sync.RWMutex
}

// TBTXP To Blocked Transaction Pool 待出块交易池
// 用于当共识协议确认自己拥有出块权利后，将UBTXP的交易倒入TBTXP
// 等待出块
type TBTXP struct {


	sync.RWMutex
}

// UCTXP UnCompleted Transaction Pool 未完成交易池
// 指的是当接收到新区块，将其中没有完成的交易(指二段交易的一段或多段交易的非最终段)
// 存入
type UCTXP struct {
	sync.RWMutex
}