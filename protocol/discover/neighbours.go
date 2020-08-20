package discover

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
)

type NeighboursMsg struct {
	*Head
	From crypto.ID	// 源结点ID
	Nodes []*Node
}

func NewNeighboursMsg(selfId crypto.ID, nodes []*Node) *NeighboursMsg {
	return &NeighboursMsg{
		Head:  NewHeadV1(MSG_NEIGHBERS),
		From: selfId,
		Nodes: nodes,
	}
}

func (n *NeighboursMsg) Decode(data io.Reader) error {
	var nodesNum uint16
	var nodes []*Node
	var err error

	n.Head = &Head{}
	if err = n.Head.Decode(data); err != nil {
		return errors.Wrap(err, "NeighboursMsg_Decode")
	}

	fromBytes := make([]byte, crypto.ID_LEN_WITH_ROLE)
	if err = binary.Read(data, binary.BigEndian, fromBytes); err != nil {
		return errors.Wrap(err, "NeighboursMsg_Decode: read fromBytes")
	}
	n.From = crypto.ID(fromBytes)
	//fmt.Println(string(n.From))

	if err = binary.Read(data, binary.BigEndian, &nodesNum); err != nil {
		return errors.Wrap(err, "NeighboursMsg_Decode: read nodesNum")
	}
	//fmt.Println("nodesNum2=", nodesNum)
	for i := uint16(0); i < nodesNum; i++ {
		node := &Node{}
		if err = node.Decode(data); err != nil {
			return errors.Wrap(err, fmt.Sprintf("NeighboursMsg_Decode: Node[%d]", i))
		}
		nodes = append(nodes, node)
	}
	n.Nodes = nodes

	return nil
}

func (n *NeighboursMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, n.Head.Encode())
	binary.Write(buf, binary.BigEndian, []byte(n.From))
	//fmt.Println(string(n.From))
	nodesNum := uint16(len(n.Nodes))
	//fmt.Println("nodesNum1=", nodesNum)
	binary.Write(buf, binary.BigEndian, nodesNum)
	for i := uint16(0); i < nodesNum; i++ {
		binary.Write(buf, binary.BigEndian, n.Nodes[i].Encode())
	}

	//fmt.Println(encoding.ToHex(crypto.HashD(buf.Bytes())))
	return buf.Bytes()
}

func (n *NeighboursMsg) String() string {
	result := fmt.Sprintf("Head %v From %s", n.Head, n.From)
	for i, node := range n.Nodes {
		result += fmt.Sprintf("[%d] %s", i, node)
	}
	return result
}
