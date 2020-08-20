package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
)

type BlockRespMsg struct {
	*Head
	Blocks []*Block
}

func NewBlockRespMsg(blocks []*Block) *BlockRespMsg {
	return &BlockRespMsg{
		Head:   NewHeadV1(MsgBlockResp),
		Blocks: blocks,
	}
}

func (brm *BlockRespMsg) String() string {
	res := new(bytes.Buffer)
	res.WriteString("{")
	n := len(brm.Blocks)
	for i:=0; i<n; i++ {	// 只取前2个字节作短名
		if i!=n-1 {
			res.WriteString(encoding.ToHex(brm.Blocks[i].BlockHeader.PrevHash[:2]) + ", ")
		} else {
			res.WriteString(encoding.ToHex(brm.Blocks[i].BlockHeader.PrevHash[:2]))
		}
	}
	res.WriteString("}")

	return fmt.Sprintf("%d block: %s", n, res.String())
}


func (brm *BlockRespMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Head
	binary.Write(buf, binary.BigEndian, brm.Head.Encode())
	// Blocks
	blocksL := uint16(len(brm.Blocks))
	binary.Write(buf, binary.BigEndian, blocksL)
	for i:=uint16(0); i<blocksL; i++ {
		bB := brm.Blocks[i].Encode()
		binary.Write(buf, binary.BigEndian, uint16(len(bB)))
		binary.Write(buf, binary.BigEndian, bB)
	}

	return buf.Bytes()
}

// 注意：如果存在成员结构不是使用binary编码而是使用其他方式编码结构体，那么外层就不能直接使用binary解码
// 必须先得到[]byte，再去解码

func (brm *BlockRespMsg) Decode(data io.Reader) error {
	// Head
	brm.Head = &Head{}
	if err := binary.Read(data, binary.BigEndian, brm.Head); err != nil {
		return errors.Wrap(err, "BlockRespMsg_Decode")
	}
	// Blocks
	blocksL := uint16(0)
	if err := binary.Read(data, binary.BigEndian, &blocksL); err != nil {
		return errors.Wrap(err, "BlockRespMsg_Decode: blocksL")
	}
	brm.Blocks = make([]*Block, blocksL)
	for i:=uint16(0); i<blocksL; i++ {
		bBL := uint16(0)
		if err := binary.Read(data, binary.BigEndian, &bBL); err != nil {
			return errors.Wrapf(err, "BlockRespMsg_Decode: Blocks[%d] bBL", i)
		}
		bB := make([]byte, bBL)
		if err := binary.Read(data, binary.BigEndian, bB); err != nil {
			return errors.Wrapf(err, "BlockRespMsg_Decode: Blocks[%d] bB", i)
		}
		brm.Blocks[i] = &Block{}
		if err := brm.Blocks[i].Decode(bytes.NewReader(bB)); err != nil {
			return errors.Wrapf(err, "BlockRespMsg_Decode: Blocks[%d]", i)
		}
	}

	return nil
}


func (brm *BlockRespMsg) Verify() error {
	if brm.Version != V1 {
		return fmt.Errorf("invalid version %d", brm.Version)
	}

	if brm.Type != MsgBlockResp {
		return fmt.Errorf("invalid type %d", brm.Type)
	}

	for _, block := range brm.Blocks {
		if err := block.Verify(); err != nil {
			return fmt.Errorf("invalid block %v", err)
		}
	}

	return nil
}

