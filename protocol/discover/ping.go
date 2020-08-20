package discover

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
)

// Head(12) | From(-) | To(-)
type PingMsg struct {
	*Head
	From crypto.ID	// 源ID
}

func NewPingMsg(from crypto.ID) *PingMsg {
	return &PingMsg{
		Head:   NewHeadV1(MSG_PING),
		From:from,
	}
}

// -HeadLen- | HeadEncoded	(这里两个长度都略去了)
// -DataLen- | DataEncoded
func (p *PingMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, p.Head.Encode())
	binary.Write(buf, binary.BigEndian, []byte(p.From))	// 这里不需要长度，因为有crypto.ID_LEN_WITH_ROLE. 这是“约定的”长度，没必要再在数据包里浪费值域
	// 记得转为[]byte存储，如果直接是p.From，其实际长度比[]byte长，因为还有额外的信息

	return buf.Bytes()
}

func (p *PingMsg) String() string {
	return fmt.Sprintf("Head %v From %s", p.Head, p.From)
}

// NOTICE： res := &PingMsg{}
func (p *PingMsg) Decode(data io.Reader) error {
	var err error

	p.Head = &Head{}
	if err = p.Head.Decode(data); err != nil {
		//fmt.Println("111", err)
		return errors.Wrap(err, "PingMsg_Decode")
	}

	fromBytes := make([]byte, crypto.ID_LEN_WITH_ROLE)
	if err = binary.Read(data, binary.BigEndian, fromBytes); err != nil {
		//fmt.Println("222", err)
		return errors.Wrap(err, "PingMsg_Decode: read fromBytes")
	}
	p.From = crypto.ID(fromBytes)

	return nil
}
