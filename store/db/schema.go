package db

import (
	"bytes"
	"encoding/binary"
	"github.com/azd1997/ecoin/common/crypto"
)

var (
	// height为uint64直接转存为的[]byte
	headerPrefix         = []byte("H")     // headerPrefix + height + hash -> header
	hashSuffix           = []byte("h")     // headerPrefix + height + hashSuffix -> header hash
	headerHeightPrefix   = []byte("n")     // headerHeightPrefix + hash -> height
	blockPrefix          = []byte("B")     // blockPrefix + height + hash -> block
	txPrefix       = []byte("T")     // txPrefix + height + hash -> tx
	txHeightPrefix = []byte("N")     // txHeightPrefix + hash -> height
	balanceSuffix          = []byte("b") // id + balanceSuffix -> balance
	creditSuffix          = []byte("c") // id + creditSuffix -> credit
	txFromSuffix       = []byte("f")     // id + txFromSuffix + txHash -> height
	txToSuffix       = []byte("t")     // id + txToSuffix + txHash -> height

	// meta data key should begin with 'm'
	mLatestHeight = []byte("mLatestHeight")
	mGenesis      = []byte("mGenesis")
)

// hbyte 将uint64整型转为字节数组
func hbyte(height uint64) []byte {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, height)
	return result
}

// byteh 将8B字节数组转为uint64
func byteh(data []byte) uint64 {
	var result uint64
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.BigEndian, &result)
	return result
}

// H..
// HeaderKey用来根据区块高度和区块哈希查询Header
func getHeaderKey(height uint64, hash crypto.Hash) []byte {
	return append(headerPrefix, append(hbyte(height), hash...)...)
}

// H..h
// HashKey用来根据区块高度查询区块哈希
func getHashKey(height uint64) []byte {
	return append(headerPrefix, append(hbyte(height), hashSuffix...)...)
}

// n..
// HeaderHeightKey用来根据区块头哈希查询区块高度
func getHeaderHeightKey(hash crypto.Hash) []byte {
	return append(headerHeightPrefix, hash...)
}

// B..
// getBlockKey用来根据区块高度和区块哈希查询区块数据
func getBlockKey(height uint64, hash crypto.Hash) []byte {
	return append(blockPrefix, append(hbyte(height), hash...)...)
}

// T..
// getTxKey用来根据区块高度和区块哈希查询交易
func getTxKey(height uint64, hash crypto.Hash) []byte {
	return append(txPrefix, append(hbyte(height), hash...)...)
}

// N..
// TxHeightKey用来根据交易哈希查询交易所在区块高度
func getTxHeightKey(hash crypto.Hash) []byte {
	return append(txHeightPrefix, hash...)
}

// ..b
// getBalanceKey用来根据用户ID查询余额
func getBalanceKey(id crypto.ID) []byte {
	return append([]byte(id), balanceSuffix...)
}

// ..c
// getCreditKey用来根据用户ID查询信用积分
func getCreditKey(id crypto.ID) []byte {
	return append([]byte(id), creditSuffix...)
}

// ..f
// getAccountTxFromKeyPrefix 获取账户交易前缀
func getAccountTxFromKeyPrefix(id crypto.ID) []byte {
	return append([]byte(id), txFromSuffix...)
}

// ..t
// getAccountTxToKeyPrefix 获取账户交易前缀
func getAccountTxToKeyPrefix(id crypto.ID) []byte {
	return append([]byte(id), txToSuffix...)
}
