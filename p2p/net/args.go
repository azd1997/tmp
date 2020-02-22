package net

import (
	"github.com/azd1997/ecoin/account"
	eaccount "github.com/azd1997/ecoin/account/ecoinaccount"
	"github.com/azd1997/ecoin/core/bc"
	"github.com/azd1997/ecoin/p2p/eaddr"
)

// Args 外部数据结构的参数，如区块链、数据存储...，注入到P2P节点中以方便调用
type Args struct {

	// 节点版本
	NodeVersion uint8

	// Server参数
	Ip string
	Port int
	Name string


	Account *account.Account

	Chain    *singlechain.Chain
	EAccouts eaccount.IEcoinAccounts
	EAddrs   *eaddr.EAddrs
}
