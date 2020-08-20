package crypto

import (
	"crypto/sha256"
	"math"
	"math/rand"
)

type Hash = []byte	// 统一使用SHA256 32B长度. 当然要改的话只需要在这里改就行了

const HASH_LENGTH = sha256.Size

var ZeroHash = make([]byte, HASH_LENGTH)	// 这基于几乎不可能出现这种全0哈希的情况而设置，用来表示未设置哈希值

// 哈希
func HashD(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// 双重哈希
func HashH(data []byte) Hash {
	hash := HashD(data)
	return HashD(hash)
}

func RandHash() Hash {
	h := make([]byte, HASH_LENGTH)
	for i:=0; i<len(h); i++ {
		h[i] = uint8(rand.Intn(math.MaxUint8))
	}
	return h
}
