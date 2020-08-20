package storage

import (
	"bytes"
	"testing"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/protocol/core"
)

func TestBlockHeader(t *testing.T) {
	headerParams := core.NewBlockHeaderParams()
	header := core.GenBlockHeaderFromParams(headerParams)
	height := uint64(100)

	blockHeader := NewBlockHeader(header, height)
	blockHeaderBytes := blockHeader.Encode()

	rBlockHeader := &BlockHeader{}
	err := rBlockHeader.Decode(bytes.NewReader(blockHeaderBytes))
	if err != nil {
		t.Fatalf("decode block header failed: %v\n", err)
	}

	if err := core.CheckBlockHeader(rBlockHeader.BlockHeader, headerParams); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint64("height", height, rBlockHeader.Height); err != nil {
		t.Fatal(err)
	}
}

func TestBlock(t *testing.T) {
	txHashes := [][]byte{
		crypto.Hash([]byte("1111")),
		crypto.Hash([]byte("22222")),
		crypto.Hash([]byte("333333")),
	}

	block := NewBlock(txHashes)
	blockBytes := block.Encode()

	rBlock := &Block{}
	err := rBlock.Decode(bytes.NewReader(blockBytes))
	if err != nil {
		t.Fatalf("decode block failed: %v\n", err)
	}

	if err := utils.TCheckInt("tx size", len(txHashes), len(rBlock.TxHashes)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < len(txHashes); i++ {
		if err := utils.TCheckBytes("tx hash", txHashes[i], rBlock.TxHashes[i]); err != nil {
			t.Fatal(err)
		}
	}
}
