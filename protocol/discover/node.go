package discover

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
)

type Node struct {
	Addr   *Address
	ID crypto.ID	// 注意这里采取的设计是crypto.ID既作账户ID，也作节点ID
}

func NewNode(addr *Address, id crypto.ID) *Node {
	return &Node{
		Addr:   addr,
		ID:id,
	}
}

func (n *Node) Decode(data io.Reader) error {
	var err error

	n.Addr = &Address{}
	if err = n.Addr.Decode(data); err != nil {
		return errors.Wrap(err, "Node_Decode")
	}
	idBytes := make([]byte, crypto.ID_LEN_WITH_ROLE)
	if err = binary.Read(data, binary.BigEndian, idBytes); err != nil {
		return errors.Wrap(err, "Node_Decode: read idBytes")
	}
	n.ID = crypto.ID(idBytes)

	return nil
}

func (n *Node) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	binary.Write(buf, binary.BigEndian, n.Addr.Encode())
	binary.Write(buf, binary.BigEndian, []byte(n.ID))

	return buf.Bytes()
}

func (n *Node) String() string {
	return fmt.Sprintf("Addr %v ID %s", n.Addr, n.ID.String())
}
