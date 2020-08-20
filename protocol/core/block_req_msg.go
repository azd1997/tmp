package core

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
)


const (
	onlyHeaderFlag    = 1
	notOnlyHeaderFlag = 0
)

// 获取区块请求
type BlockReqMsg struct {
	*Head
	Base       crypto.Hash	// 自身最高区块哈希
	End        crypto.Hash	// 自己请求的对方节点的最高区块哈希  (当然也可以不一下子要这么多)
	OnlyHeader uint8		// 是否只要区块头
}

func NewBlockReqMsg(base crypto.Hash, end crypto.Hash, onlyHeader bool) *BlockReqMsg {
	brm := &BlockReqMsg{
		Head:       NewHeadV1(MsgBlockReq),
		Base:       base,
		End:        end,
		OnlyHeader: notOnlyHeaderFlag,
	}
	if onlyHeader {
		brm.OnlyHeader = onlyHeaderFlag
	}
	return brm
}

func (brm *BlockReqMsg) String() string {
	return fmt.Sprintf("Base %X End %X OnlyHeader %v", brm.Base, brm.End, brm.OnlyHeader)
}

func (brm *BlockReqMsg) IsOnlyHeader() bool {
	return brm.OnlyHeader == onlyHeaderFlag
}

func (brm *BlockReqMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Head
	binary.Write(buf, binary.BigEndian, brm.Head.Encode())
	// Base
	binary.Write(buf, binary.BigEndian, brm.Base)
	// End
	binary.Write(buf, binary.BigEndian, brm.End)
	// OnlyHeader
	binary.Write(buf, binary.BigEndian, brm.OnlyHeader)

	return buf.Bytes()
}

func (brm *BlockReqMsg) Decode(data io.Reader) error {
	// Head
	brm.Head = &Head{}
	if err := brm.Head.Decode(data); err != nil {
		return errors.Wrap(err, "BlockReqMsg_Decode")
	}
	// Base
	brm.Base = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, brm.Base); err != nil {
		return errors.Wrap(err, "BlockReqMsg_Decode: Base")
	}
	// End
	brm.End = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, brm.End); err != nil {
		return errors.Wrap(err, "BlockReqMsg_Decode: End")
	}
	// OnlyHeader
	if err := binary.Read(data, binary.BigEndian, &brm.OnlyHeader); err != nil {
		return errors.Wrap(err, "BlockReqMsg_Decode: OnlyHeader")
	}

	return nil
}

func (brm *BlockReqMsg) Verify() error {
	if brm.Version != V1 {
		return fmt.Errorf("invalid version %d", brm.Version)
	}

	if brm.Type != MsgBlockReq {
		return fmt.Errorf("invalid type %d", brm.Type)
	}

	if len(brm.Base) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid base %X", brm.Base)
	}

	if len(brm.End) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid end %X", brm.End)
	}

	return nil
}

