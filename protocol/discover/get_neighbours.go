package discover

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
)

type GetNeighboursMsg struct {
	*Head
	From crypto.ID
}

func NewGetNeighboursMsg(from crypto.ID) *GetNeighboursMsg {
	return &GetNeighboursMsg{
		Head:   NewHeadV1(MSG_GET_NEIGHBERS),
		From:from,
	}
}

func (g *GetNeighboursMsg) Decode(data io.Reader) error {
	var err error

	g.Head = &Head{}
	if err = g.Head.Decode(data); err != nil {
		return errors.Wrap(err, "GetNeighboursMsg_Decode")
	}

	fromBytes := make([]byte, crypto.ID_LEN_WITH_ROLE)
	if err = binary.Read(data, binary.BigEndian, fromBytes); err != nil {
		return errors.Wrap(err, "GetNeighboursMsg_Decode: read fromBytes")
	}
	g.From = crypto.ID(fromBytes)

	return nil
}

func (g *GetNeighboursMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, g.Head.Encode())
	binary.Write(buf, binary.BigEndian, []byte(g.From))		// 这里不需要长度，因为有crypto.ID_LEN_WITH_ROLE. 这是“约定的”长度，没必要再在数据包里浪费值域

	return buf.Bytes()
}

func (g *GetNeighboursMsg) String() string {
	return fmt.Sprintf("Head %v From %s\n", g.Head, g.From)
}
