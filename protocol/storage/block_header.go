package storage

import (
	"encoding/gob"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/protocol/core"
)

// BlockHeader 区块头
type BlockHeader struct {
	*core.BlockHeader
	Height uint64
}

func NewBlockHeader(h *core.BlockHeader, height uint64) *BlockHeader {
	return &BlockHeader{
		BlockHeader: h,
		Height:      height,
	}
}

func (b *BlockHeader) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(b)
	if err != nil {
		return errors.Wrap(err, "BlockHeader_Decode")
	}
	return nil
}

func (b *BlockHeader) Encode() []byte {
	res, _ := encoding.GobEncode(b)
	return res
}
