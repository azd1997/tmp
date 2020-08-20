package db

import (
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/protocol/core"
)

type db interface {
	Init(path string) error

	HasGenesis() bool
	PutGenesis(block *core.Block) error
	PutBlock(block *core.Block, height uint64) error

	GetHash(height uint64) ([]byte, error)

	GetHeaderViaHeight(height uint64) (*core.BlockHeader, []byte, error)
	GetHeaderViaHash(h crypto.Hash) (*core.BlockHeader, uint64, error)

	GetBlockViaHeight(height uint64) (*core.Block, []byte, error)
	GetBlockViaHash(h crypto.Hash) (*core.Block, uint64, error)

	GetTxViaHash(h crypto.Hash) (*core.Tx, uint64, error)
	GetTxFromHashesViaID(id crypto.ID) ([]crypto.Hash, []uint64, error)
	GetTxToHashesViaID(id crypto.ID) ([]crypto.Hash, []uint64, error)


	HasTx(h crypto.Hash) bool

	GetBalanceViaID(id crypto.ID) (uint64, error)

	GetLatestHeight() (uint64, error)
	GetLatestHeader() (*core.BlockHeader, uint64, []byte, error)

	Close()
}

var (
	instance db
)

// Init 初始化数据库模块
func Init(path string) error {
	instance = newBadger()
	return instance.Init(path)
}

// HashGenesis 判断是否存在Genesis区块
func HasGenesis() bool {
	return instance.HasGenesis()
}

// PutGenesis 存入Genesis区块
func PutGenesis(block *core.Block) error {
	return instance.PutGenesis(block)
}

// PutBlock 存入区块
func PutBlock(block *core.Block, height uint64) error {
	return instance.PutBlock(block, height)
}

// GetHash 根据区块高度查询区块哈希
func GetHash(height uint64) (crypto.Hash, error) {
	return instance.GetHash(height)
}

// GetHeaderViaHeight 通过高度查询区块头信息
func GetHeaderViaHeight(height uint64) (*core.BlockHeader, crypto.Hash, error) {
	return instance.GetHeaderViaHeight(height)
}

// GetHeaderViaHash 根据区块哈希查询区块头信息
func GetHeaderViaHash(h crypto.Hash) (*core.BlockHeader, uint64, error) {
	return instance.GetHeaderViaHash(h)
}

// GetBlockViaHeight 根据高度查询完整区块信息
func GetBlockViaHeight(height uint64) (*core.Block, crypto.Hash, error) {
	return instance.GetBlockViaHeight(height)
}

// GetBlockViaHash 根据哈希查询区块数据
func GetBlockViaHash(h crypto.Hash) (*core.Block, uint64, error) {
	return instance.GetBlockViaHash(h)
}

// GetTxViaHash 根据区块哈希查询交易
func GetTxViaHash(h crypto.Hash) (*core.Tx, uint64, error) {
	return instance.GetTxViaHash(h)
}

// GetTxFromHashesViaID 根据账户ID查询其作为发送方的交易
func GetTxFromHashesViaID(id crypto.ID) ([]crypto.Hash, []uint64, error) {
	return instance.GetTxFromHashesViaID(id)
}

// GetTxToHashesViaID 根据账户ID查询其作为接收方的交易
func GetTxToHashesViaID(id crypto.ID) ([]crypto.Hash, []uint64, error) {
	return instance.GetTxToHashesViaID(id)
}

// HasTx 根据交易哈希判断是否存在某个交易
func HasTx(h crypto.Hash) bool {
	return instance.HasTx(h)
}

// GetBalanceViaID 查询账户的余额
func GetBalanceViaID(id crypto.ID) (uint64, error) {
	return instance.GetBalanceViaID(id)
}

// TODO
// GetCreditViaID 查询账户的信誉分
//func GetCreditViaID(id crypto.ID) (uint64, error) {
//	return instance.GetCreditViaID(id)
//}

// GetLatestHeight 获取最高高度
func GetLatestHeight() (uint64, error) {
	return instance.GetLatestHeight()
}

// GetLatestHeader 获取最高高度的区块头
func GetLatestHeader() (*core.BlockHeader, uint64, crypto.Hash, error) {
	return instance.GetLatestHeader()
}

// Close 关闭数据库模块
func Close() {
	if instance != nil {
		instance.Close()
	}
}
