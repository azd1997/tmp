package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/encoding"
)

// TxBroadcastMsg 用于网络传播，加上了一个Head
type TxBroadcastMsg struct {
	*Head
	Txs []*Tx	// 可以一次广播一个或多个交易
}

func NewTxBroadcastMsg(txs []*Tx) *TxBroadcastMsg {
	return &TxBroadcastMsg{
		Head: NewHeadV1(MsgTxBroadcast),
		Txs:  txs,
	}
}

func (tbm *TxBroadcastMsg) String() string {
	res := new(bytes.Buffer)
	res.WriteString("{")
	n := len(tbm.Txs)
	for i:=0; i<n; i++ {
		if i!=n-1 {
			res.WriteString(encoding.ToHex(tbm.Txs[i].Id[:2]) + ", ")
		} else {
			res.WriteString(encoding.ToHex(tbm.Txs[i].Id[:2]))
		}
	}
	res.WriteString("}")

	return fmt.Sprintf("%d tx: %s", n, res.String())
}

////////////////////////////////////////////////////////////////////

func (tbm *TxBroadcastMsg) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Head
	binary.Write(buf, binary.BigEndian, tbm.Head.Encode())
	// Txs
	txsL := uint16(len(tbm.Txs))
	binary.Write(buf, binary.BigEndian, txsL)
	for i:=uint16(0); i<txsL; i++ {
		txB := tbm.Txs[i].Encode()
		binary.Write(buf, binary.BigEndian, uint16(len(txB)))
		binary.Write(buf, binary.BigEndian, txB)
	}

	return buf.Bytes()
}

func (tbm *TxBroadcastMsg) Decode(data io.Reader) error {
	// Head
	tbm.Head = &Head{}
	if err := binary.Read(data, binary.BigEndian, tbm.Head); err != nil {
		return errors.Wrap(err, "TxBroadcastMsg_Decode")
	}
	// Txs
	txsL := uint16(0)
	if err := binary.Read(data, binary.BigEndian, &txsL); err != nil {
		return errors.Wrap(err, "TxBroadcastMsg_Decode: txsL")
	}
	tbm.Txs = make([]*Tx, txsL)
	for i:=uint16(0); i<txsL; i++ {
		txBL := uint16(0)
		if err := binary.Read(data, binary.BigEndian, &txBL); err != nil {
			return errors.Wrapf(err, "TxBroadcastMsg_Decode: Txs[%d] txBL", i)
		}
		txB := make([]byte, txBL)
		if err := binary.Read(data, binary.BigEndian, txB); err != nil {
			return errors.Wrapf(err, "TxBroadcastMsg_Decode: Txs[%d] txB", i)
		}
		tbm.Txs[i] = &Tx{}
		if err := tbm.Txs[i].Decode(bytes.NewReader(txB)); err != nil {
			return errors.Wrapf(err, "TxBroadcastMsg_Decode: Txs[%d]", i)
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////

func (tbm *TxBroadcastMsg) Verify() error {
	if tbm.Version != V1 {
		return fmt.Errorf("invalid version %d", tbm.Version)
	}

	if tbm.Type != MsgTxBroadcast {
		return fmt.Errorf("invalid type %d", tbm.Type)
	}

	if tbm.Txs == nil {
		return fmt.Errorf("nil Txs")
	}

	for _, tx := range tbm.Txs {
		if err := tx.Verify(); err != nil {
			return fmt.Errorf("invalid tx:%v", err)
		}
	}

	return nil
}