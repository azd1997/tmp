package utils

import (
	"bytes"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (


	TIME_FORMAT = "2006/01/02 15:04:05"
)


var bufPool = &sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// GetBuf gets a *bytes.Buffer from pool
func GetBuf() *bytes.Buffer {
	result := bufPool.Get().(*bytes.Buffer)
	result.Reset()
	return result
}

// ReturnBuf returns a *bytes.Buffer to Pool once you don't need it
func ReturnBuf(buf *bytes.Buffer) {
	bufPool.Put(buf)
}

// AccessCheck checks whether the file or directory exists
func AccessCheck(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("not found %s or permision denied", err)
	}
	return nil
}

// ParseIPPort parse IP:Port format sring
func ParseIPPort(ipPort string) (net.IP, int) {
	s := strings.Split(ipPort, ":")
	if len(s) != 2 {
		return nil, 0
	}

	ip := net.ParseIP(s[0])
	if ip == nil || ip.To4() == nil {
		return nil, 0
	}

	port, err := strconv.Atoi(s[1])
	if err != nil || port <= 0 || port > 65535 {
		return nil, 0
	}

	return ip, port
}

// ReadableBigInt returns more readable format for big.Int
func ReadableBigInt(num *big.Int) string {
	hexStr := fmt.Sprintf("%X", num)
	length := len(hexStr)

	var result string
	format := "0x%s..(%d)"
	cut := 6
	if length > cut {
		result = fmt.Sprintf(format, hexStr[0:cut], length)
	} else {
		result = fmt.Sprintf(format, hexStr, length)
	}
	return result
}

// Uint8Len returns bytes length in uint8 type
func Uint8Len(data []byte) uint8 {
	return uint8(len(data))
}

// Uint16Len returns bytes length in uint16 type
func Uint16Len(data []byte) uint16 {
	return uint16(len(data))
}

// Uint32Len returns bytes length in uint32 type
func Uint32Len(data []byte) uint32 {
	return uint32(len(data))
}

// TimeToString returns a textual representation of the time;
// it only accepts int64 or time.Time type
func TimeToString(t interface{}) string {

	if int64T, ok := t.(int64); ok {
		return time.Unix(int64T, 0).Format(TIME_FORMAT)
	}

	if timeT, ok := t.(time.Time); ok {
		return timeT.Format(TIME_FORMAT)
	}

	log.Fatalf("invalid call to TimeToString (%v)\n", t)
	return ""
}
