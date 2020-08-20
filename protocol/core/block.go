package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/utils"
)

type Block struct {
	*BlockHeader
	Txs []*Tx
}

func NewBlock(header *BlockHeader, txs []*Tx) *Block {
	return &Block{
		BlockHeader: header,
		Txs:txs,
	}
}

func (b *Block) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// BlockHeader
	binary.Write(buf, binary.BigEndian, b.BlockHeader.Encode())
	// Txs
	binary.Write(buf, binary.BigEndian, uint16(len(b.Txs)))
	for i:=0; i<len(b.Txs); i++ {
		txBytes := b.Txs[i].Encode()
		binary.Write(buf, binary.BigEndian, uint16(len(txBytes)))
		binary.Write(buf, binary.BigEndian, txBytes)
	}

	return buf.Bytes()
}

func (b *Block) Decode(data io.Reader) error {
	// BlockHeader
	b.BlockHeader = &BlockHeader{}
	if err := b.BlockHeader.Decode(data); err != nil {
		return errors.Wrap(err, "Block_Decode")
	}

	// Txs
	txsL := uint16(0)
	if err := binary.Read(data, binary.BigEndian, &txsL); err != nil {
		return errors.Wrap(err, "Block_Decode: txsL")
	}
	txs := make([]*Tx, txsL)
	for i:=uint16(0); i<txsL; i++ {
		txBL := uint16(0)
		if err := binary.Read(data, binary.BigEndian, &txBL); err != nil {
			return errors.Wrapf(err, "Block_Decode: tx[%d] txBL", i)
		}
		txB := make([]byte, txBL)
		if err := binary.Read(data, binary.BigEndian, txB); err != nil {
			return errors.Wrapf(err, "Block_Decode: tx[%d] txB", i)
		}
		txs[i] = &Tx{}
		if err := txs[i].Decode(bytes.NewReader(txB)); err != nil {
			return errors.Wrapf(err, "Block_Decode: tx[%d]", i)
		}
	}
	b.Txs = txs

	return nil
}

func (b *Block) Verify() error {
	var err error

	if b.BlockHeader == nil {
		return fmt.Errorf("nil header")
	}

	if err = b.BlockHeader.Verify(); err != nil {
		return fmt.Errorf("block header verify failed:%v", err)
	}

	if b.IsEmptyMerkleRoot() && len(b.Txs) != 0 {
		return fmt.Errorf("expect 0 tx, but %d", len(b.Txs))
	}

	if !b.IsEmptyMerkleRoot() && len(b.Txs) == 0 {
		return fmt.Errorf("expect tx, but empty")
	}

	for _, tx := range b.Txs {
		if err = tx.Verify(); err != nil {
			return fmt.Errorf("tx verify failed:%v", err)
		}
	}

	return nil
}

func (b *Block) ShallowCopy(onlyHeader bool) *Block {
	var txs []*Tx = nil
	if !onlyHeader {
		txs = b.Txs
	}
	return &Block{
		BlockHeader: b.BlockHeader,
		Txs:txs,
	}
}
