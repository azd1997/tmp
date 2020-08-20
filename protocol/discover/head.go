package discover

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"time"

	"github.com/azd1997/ecoin/common/utils"
)

type Head struct {
	Version  uint8
	Type     DiscoverMsgType
	Time     int64
}

func NewHeadV1(t DiscoverMsgType) *Head {
	return &Head{
		Version:  DISCOVER_V1,
		Type:     t,
		Time:     time.Now().Unix(),
	}
}

func (h *Head) Decode(data io.Reader) error {

	if err := binary.Read(data, binary.BigEndian, &h.Version); err != nil {
		return errors.Wrap(err, "Head_Decode: read Version")
	}
	if err := binary.Read(data, binary.BigEndian, &h.Type); err != nil {
		return errors.Wrap(err, "Head_Decode: read Type")
	}
	if err := binary.Read(data, binary.BigEndian, &h.Time); err != nil {
		return errors.Wrap(err, "Head_Decode: read Time")
	}

	return nil
}

func (h *Head) Encode() []byte {
	res := utils.GetBuf()
	defer utils.ReturnBuf(res)

	binary.Write(res, binary.BigEndian, h.Version)
	binary.Write(res, binary.BigEndian, h.Type)
	binary.Write(res, binary.BigEndian, h.Time)

	return res.Bytes()
}

func (h *Head) String() string {
	return fmt.Sprintf("Version %d Type %d Time %s",
		h.Version, h.Type, utils.TimeToString(h.Time))
}
