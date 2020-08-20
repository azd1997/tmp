package storage

import (
	"encoding/gob"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/encoding"
)

// Block 区块体，存放所有交易哈希
type Block struct {
	TxHashes []crypto.Hash
}

func NewBlock(txHashes []crypto.Hash) *Block {
	return &Block{
		TxHashes: txHashes,
	}
}

func (b *Block) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(b)
	if err != nil {
		return errors.Wrap(err, "Block_Decode")
	}
	return nil
}

func (b *Block) Encode() []byte {
	res, _ := encoding.GobEncode(b)
	return res
}
