package discover

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
	"net"
)

type Address struct {
	IP   net.IP
	Port int32
}

func NewAddress(ipstr string, port int32) *Address {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil
	}
	return &Address{
		IP:   ip,
		Port: port,
	}
}

func (a *Address) Decode(data io.Reader) error {
	var ipLen uint8
	if err := binary.Read(data, binary.BigEndian, &ipLen); err != nil {
		return errors.Wrap(err, "Address_Decode: read ipLen")
	}
	//fmt.Println("ipLen2=", ipLen)

	ipBuf := make([]byte, ipLen)
	if err := binary.Read(data, binary.BigEndian, ipBuf); err != nil {
		return errors.Wrap(err, "Address_Decode: read ipBuf")
	}
	if err := a.IP.UnmarshalText(ipBuf); err != nil {
		return errors.Wrap(err, "Address_Decode")
	}
	if err := binary.Read(data, binary.BigEndian, &a.Port); err != nil {
		return errors.Wrap(err, "Address_Decode: read port")
	}
	return nil
}

func (a *Address) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	ipBytes, _ := a.IP.MarshalText()
	// fmt.Println(a.IP, ipBytes)
	ipLen := utils.Uint8Len(ipBytes)
	//fmt.Println("ipLen1=", ipLen)
	binary.Write(buf, binary.BigEndian, ipLen)
	binary.Write(buf, binary.BigEndian, ipBytes)

	binary.Write(buf, binary.BigEndian, a.Port)

	return buf.Bytes()
}

func (a *Address) String() string {
	return fmt.Sprintf("%v:%d", a.IP, a.Port)
}
