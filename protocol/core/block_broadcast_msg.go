package core

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
)

type BlockBroadcastMsg struct {
	*Head
	Block *Block
}

func NewBlockBroadcastMsg(block *Block) *BlockBroadcastMsg {
	return &BlockBroadcastMsg{
		Head:  NewHeadV1(MsgBlockBroadcast),
		Block: block,
	}
}

func (b *BlockBroadcastMsg) Decode(data io.Reader) error {
	b.Head, b.Block = &Head{}, &Block{}

	// Head
	if err := b.Head.Decode(data); err != nil {
		return errors.Wrap(err, "BlockBroadcastMsg_Decode")
	}
	// Block
	if err := b.Block.Decode(data); err != nil {
		return errors.Wrap(err, "BlockBroadcastMsg_Decode")
	}

	return nil
}

func (b *BlockBroadcastMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Head
	binary.Write(buf, binary.BigEndian, b.Head.Encode())
	// Block
	binary.Write(buf, binary.BigEndian, b.Block.Encode())

	return buf.Bytes()
}

func (b *BlockBroadcastMsg) Verify() error {
	if b.Version != V1 {
		return fmt.Errorf("invalid version %d", b.Version)
	}

	if b.Type != MsgBlockBroadcast {
		return fmt.Errorf("invlaid type %d", b.Type)
	}

	if b.Block == nil {
		return fmt.Errorf("nil block")
	}

	if err := b.Block.Verify(); err != nil {
		return fmt.Errorf("invalid block %v", err)
	}

	return nil
}

func (b *BlockBroadcastMsg) String() string {
	return fmt.Sprintf("Block %X", b.Block.Hash)
}
