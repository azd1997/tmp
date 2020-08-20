/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/12 16:01
* @Description: The file is for
***********************************************************************/

package raw

import (
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
)

// RawTx 原始交易指的是用户侧创建交易时所要提供的信息
// 在协议层会对RawTx进行处理得到Tx，
// 重新读出时再根据其交易类型附上交易权重（认为某些交易
// 比如远程诊断是更优先的）来决定谁更优先被装入新区块
type Tx struct {
	Type uint8		// 交易类型
	Uncompleted uint8	// 交易活动未完成? 0表示false(完成)， 1表示true(未完成)
	To crypto.ID		// 接收者账户ID
	Amount uint32	// 转账数额
	Description []byte	// 描述

	// Payload 关于Payload，见core.Tx
	// TODO: Payload 暂时不具体设计
	Payload []byte

	// 来源交易Id
	// 如果为nil，则说明没有来源交易
	// rawTx -> core.tx 过程中需要将prevTxId不断找到对应的*Tx，形成链表
	PrevTxId crypto.Hash
}

// Hash 用于计算RawTx的哈希
func (tx *Tx) Hash() crypto.Hash {
	data, _ := encoding.GobEncode(tx)
	return crypto.HashD(data)
}
