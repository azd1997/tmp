package discover

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
)

type PongMsg struct {
	*Head
	PingHash crypto.Hash
	From crypto.ID
}

func NewPongMsg(pingHash crypto.Hash, from crypto.ID) *PongMsg {
	return &PongMsg{
		Head:     NewHeadV1(MSG_PONG),
		PingHash: pingHash,
		From:   from,
	}
}

func (p *PongMsg) Decode(data io.Reader) error {
	var err error

	p.Head = &Head{}
	if err = p.Head.Decode(data); err != nil {
		return errors.Wrap(err, "PongMsg_Decode")
	}

	p.PingHash = make([]byte, crypto.HASH_LENGTH)
	if err = binary.Read(data, binary.BigEndian, p.PingHash); err != nil {
		return errors.Wrap(err, "PongMsg_Decode: read PingHash")
	}

	fromBytes := make([]byte, crypto.ID_LEN_WITH_ROLE)
	if err = binary.Read(data, binary.BigEndian, fromBytes); err != nil {
		return errors.Wrap(err, "PongMsg_Decode: read fromBytes")
	}
	p.From = crypto.ID(fromBytes)

	return nil
}

func (p *PongMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, p.Head.Encode())
	binary.Write(buf, binary.BigEndian, p.PingHash)	// 这里不需要长度，因为有crypto.HASH_LEN. 这是“约定的”长度，没必要再在数据包里浪费值域
	binary.Write(buf, binary.BigEndian, []byte(p.From))		// 这里不需要长度，因为有crypto.ID_LEN_WITH_ROLE. 这是“约定的”长度，没必要再在数据包里浪费值域

	return buf.Bytes()
}

func (p *PongMsg) String() string {
	return fmt.Sprintf("Head %v PingHash %X From %s", p.Head, p.PingHash, p.From)
}
