package p2p

import (
	"bytes"
	"encoding/binary"
	"github.com/azd1997/ecoin/common/utils"
	"hash/crc32"
)

/*
+-------------+-----------+--------------+
|   Length    |    CRC    |    Protocol  |
+-------------+-----------+--------------+
|                Payload                 |
+----------------------------------------+

(bytes)
Length		4
CRC			4
Protocol	1
*/

const tcpHeaderSize = 9

// 构建TCP packet
// 和provider(UDP Server)处不同，这里由于真正构建Packet时进行了粘包拆包处理，因此协议层可以直接使用gob编码
// 而provider处，由于上层需要先检查Head，Head检查通过才能解码Packet，所以使用gob编码是不方便的，因此使用了binary。
// 这里的包格式：
// Len(4B) | CRC(4B) | ProtocolID(1B) | payload(-) 也就是TCP包的头部占9B
func buildTCPPacket(payload []byte, protocolID uint8) []byte {
	length := utils.Uint32Len(payload)
	crc := crc32.ChecksumIEEE(payload)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, length)
	binary.Write(buf, binary.BigEndian, crc)
	binary.Write(buf, binary.BigEndian, protocolID)
	buf.Write(payload)

	return buf.Bytes()
}

// 从TCP连接读取的数据存入缓冲bytes.Buffer，而本函数则读取bytes.Buffer将数据分割成一个个的packet
func splitTCPStream(received *bytes.Buffer) ([][]byte, error) {
	var length uint32
	var packets [][]byte

	for received.Len() > tcpHeaderSize {
		// peeker' reading has no effect on received
		peeker := bytes.NewReader(received.Bytes())
		binary.Read(peeker, binary.BigEndian, &length)

		packetLen := tcpHeaderSize + length
		if received.Len() < int(packetLen) {
			break
		}

		packet := make([]byte, packetLen)
		if _, err := received.Read(packet); err != nil {
			return nil, err
		}

		packets = append(packets, packet)
	}

	return packets, nil
}

// 检查从bytes.Buffer中分割出来的packet是否有效，有效则将payload及协议ID返回
func verifyTCPPacket(packet []byte) (bool, []byte, uint8) {
	var length uint32
	var crc uint32
	var protocolID uint8

	packetReader := bytes.NewReader(packet)
	binary.Read(packetReader, binary.BigEndian, &length)
	binary.Read(packetReader, binary.BigEndian, &crc)
	binary.Read(packetReader, binary.BigEndian, &protocolID)

	payload := make([]byte, length)
	packetReader.Read(payload)

	checkCrc := crc32.ChecksumIEEE(payload)
	if crc != checkCrc {
		return false, nil, 0
	}

	return true, payload, protocolID
}
