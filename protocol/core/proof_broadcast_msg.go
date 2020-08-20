package core

import (
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
)

// POT 证明消息
// 为避免证明消息在传播过程被篡改
type ProofBroadcastMsg struct {
	*Head	// 消息头
	*PoTProof // 证明
}

func NewProofBroadcastMsg(proof *PoTProof) *ProofBroadcastMsg {
	return &ProofBroadcastMsg{
		Head:  NewHeadV1(MsgProofBroadcast),
		PoTProof: proof,
	}
}

func (pbm *ProofBroadcastMsg) Decode(data io.Reader) error {
	pbm.Head, pbm.PoTProof = &Head{}, &PoTProof{}

	// Head
	if err := pbm.Head.Decode(data); err != nil {
		return errors.Wrap(err, "ProofBroadcastMsg_Decode")
	}
	// Block
	if err := pbm.PoTProof.Decode(data); err != nil {
		return errors.Wrap(err, "ProofBroadcastMsg_Decode")
	}

	return nil
}

func (pbm *ProofBroadcastMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Head
	binary.Write(buf, binary.BigEndian, pbm.Head.Encode())
	// PoTProof
	binary.Write(buf, binary.BigEndian, pbm.PoTProof.Encode())

	return buf.Bytes()
}

func (pbm *ProofBroadcastMsg) Verify() error {
	if pbm.Version != V1 {
		return fmt.Errorf("invalid version %d", pbm.Version)
	}

	if pbm.Type != MsgProofBroadcast {
		return fmt.Errorf("invlaid type %d", pbm.Type)
	}

	if pbm.PoTProof == nil {
		return fmt.Errorf("nil proof")
	}

	if err := pbm.PoTProof.Verify(); err != nil {
		return fmt.Errorf("invalid proof: %v", err)
	}

	return nil
}

func (pbm *ProofBroadcastMsg) String() string {
	return fmt.Sprintf("Proof %s", pbm.PoTProof.String())
}

