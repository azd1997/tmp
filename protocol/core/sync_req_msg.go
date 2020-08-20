package core

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
)


// 同步请求:将自身最高区块哈希发给对方，对方若比自己高，就会返回SyncRespMsg，然后自己再根据Resp去向对方请求区块数据
type SyncReqMsg struct {
	*Head
	Base crypto.Hash		// Base填的是自身区块链最新区块的哈希，接收则将其后所有的区块哈希返回给请求者
}

func NewSyncReqMsg(base crypto.Hash) *SyncReqMsg {
	return &SyncReqMsg{
		Head: NewHeadV1(MsgSyncReq),
		Base: base,
	}
}

func (srm *SyncReqMsg) String() string {
	return fmt.Sprintf("Base %X", srm.Base)
}

func (srm *SyncReqMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Head
	binary.Write(buf, binary.BigEndian, srm.Head.Encode())
	// Base
	binary.Write(buf, binary.BigEndian, srm.Base)

	return buf.Bytes()
}

func (srm *SyncReqMsg) Decode(data io.Reader) error {
	// Head
	srm.Head = &Head{}
	if err := srm.Head.Decode(data); err != nil {
		return errors.Wrap(err, "SyncReqMsg_Decode")
	}
	// Base
	srm.Base = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, srm.Base); err != nil {
		return errors.Wrap(err, "SyncReqMsg_Decode: Base")
	}

	return nil
}

func (srm *SyncReqMsg) Verify() error {
	if srm.Version != V1 {
		return fmt.Errorf("invalid version %d", srm.Version)
	}

	if srm.Type != MsgSyncReq {
		return fmt.Errorf("invalid type %d", srm.Type)
	}

	if len(srm.Base) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid base %X", srm.Base)
	}

	return nil
}


