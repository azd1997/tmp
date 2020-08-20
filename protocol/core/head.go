package core

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
)

type Head struct {
	Version  uint8
	Type     MsgType
}

func NewHeadV1(typ MsgType) *Head {
	return &Head{
		Version:  V1,
		Type:     typ,
	}
}

func (h *Head) Decode(data io.Reader) error {
	if err := binary.Read(data, binary.BigEndian, &h.Version); err != nil {
		return errors.Wrap(err, "Head_Decode: Version")
	}
	if err := binary.Read(data, binary.BigEndian, &h.Type); err != nil {
		return errors.Wrap(err, "Head_Decode: Type")
	}

	return nil
}

func (h *Head) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, h.Version)
	binary.Write(buf, binary.BigEndian, h.Type)

	return buf.Bytes()
}

func (h *Head) String() string {
	return fmt.Sprintf("Version %d Type %d", h.Version, h.Type)
}
