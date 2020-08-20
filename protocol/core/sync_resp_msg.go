package core

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
)


type SyncRespMsg struct {
	*Head
	Base       crypto.Hash	// 请求者最新区块哈希
	End        crypto.Hash	// 被请求者响应的自身最高区块哈希
	HeightDiff uint32	// 高度差。高度差为0说明 请求者的链长 >= 被请求者的链长。，某种意义上就是请求者的链是"相对最新up to date"的。
						// 真正判断最新，是要把所有人都问过一遍之后才可以确定，这个过程是动态的
}

func NewSyncRespMsg(base crypto.Hash, end crypto.Hash, heightDiff uint32) *SyncRespMsg {
	return &SyncRespMsg{
		Head:       NewHeadV1(MsgSyncResp),
		Base:       base,
		End:        end,
		HeightDiff: heightDiff,
	}
}

func (srm *SyncRespMsg) String() string {
	if srm.IsUptodate() {
		return "already uptodate"
	}

	return fmt.Sprintf("Base %X End %X HeightDiff %d",
		srm.Base, srm.End, srm.HeightDiff)
}

func (srm *SyncRespMsg) IsUptodate() bool {
	return srm.HeightDiff == 0
}

func (srm *SyncRespMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Head
	binary.Write(buf, binary.BigEndian, srm.Head.Encode())
	// Base
	binary.Write(buf, binary.BigEndian, srm.Base)
	// End
	binary.Write(buf, binary.BigEndian, srm.End)
	// HeightDiff
	binary.Write(buf, binary.BigEndian, srm.HeightDiff)

	return buf.Bytes()
}

func (srm *SyncRespMsg) Decode(data io.Reader) error {
	// Head
	srm.Head = &Head{}
	if err := srm.Head.Decode(data); err != nil {
		return errors.Wrap(err, "SyncRespMsg_Decode")
	}
	// Base
	srm.Base = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, srm.Base); err != nil {
		return errors.Wrap(err, "SyncRespMsg_Decode: Base")
	}
	// End
	srm.End = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, srm.End); err != nil {
		return errors.Wrap(err, "SyncRespMsg_Decode: End")
	}
	// HeightDiff
	if err := binary.Read(data, binary.BigEndian, &srm.HeightDiff); err != nil {
		return errors.Wrap(err, "SyncRespMsg_Decode: HeightDiff")
	}

	return nil
}



func (srm *SyncRespMsg) Verify() error {
	if srm.Version != V1 {
		return fmt.Errorf("invalid version %d", srm.Version)
	}

	if srm.Type != MsgSyncResp {
		return fmt.Errorf("invalid type %d", srm.Type)
	}

	if len(srm.Base) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid base %X", srm.Base)
	}

	if len(srm.End) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid end %X", srm.End)
	}

	return nil
}


