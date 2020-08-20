/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/7 12:36
* @Description: The file is for
***********************************************************************/

package view

import (
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/protocol/raw"
)


////////////////////////////// BlockJSON //////////////////////////////

// 这里是作为展示区块信息时的简略的交易信息
type TxInBlockJSON struct {
	Id   string `json:"id"`
	Type        uint8  `json:"type"`
	From string `json:"from"`
	To     string `json:"to"`   // To.ToHex()
	Amount uint32 `json:"amount"`

	// TODO
}

type BlockJSON struct {
	Version    uint8            `json:"version"`
	Time       int64            `json:"time"`
	Hash       string           `json:"hash"`
	PrevHash   string           `json:"prev_hash"`
	MerkleRoot string           `json:"merkle_root"`
	CreateBy   string           `json:"create_by"`
	Height     uint64           `json:"height"`
	Txs        []*TxInBlockJSON `json:"txs"`
}

func (b *BlockJSON) FromBlockInfo(info *BlockInfo) {
	b.Version = info.Version
	b.Time = info.Time
	b.Hash = encoding.ToHex(info.Hash)
	b.PrevHash = encoding.ToHex(info.PrevHash)
	b.MerkleRoot = encoding.ToHex(info.MerkleRoot)
	b.CreateBy = info.CreateBy.ToHex()
	b.Height = info.Height

	for _, tx := range info.Block.Txs {
		txJSON := &TxInBlockJSON{}
		txJSON.Id = encoding.ToHex(tx.Id)
		txJSON.From = tx.From.ToHex()
		txJSON.To = tx.To.ToHex()
		txJSON.Amount = tx.Amount
		txJSON.Type = tx.Type
		b.Txs = append(b.Txs, txJSON)
	}
}

/////////////////////////////// TxJSON ///////////////////////////////////

type TxJSON struct {
	Version     uint8  `json:"version"`
	Type        uint8  `json:"type"`
	Uncompleted uint8  `json:"uncompleted"`
	Time        int64  `json:"time"`
	Id          string `json:"id"` // hex

	From   string `json:"from"` // From.ToHex()
	To     string `json:"to"`   // To.ToHex()
	Amount uint32 `json:"amount"`
	Sig    string `json:"signature"` // hex(Sig)

	Payload  string `json:"payload"`    // string(Payload)
	PrevTxId string `json:"prev_tx_id"` // hex

	Description string `json:"description"`

	// 这是交易所在的区块的信息
	Height    uint64 `json:"height"`
	BlockHash string `json:"block_hash"` // hex
}

// 从JSON转为CoreTx
func (t *TxJSON) ToCoreTx() *core.Tx {
	if t.Version != core.V1 {
		return nil
	}
	var prevId []byte
	var from, to []byte
	var sig []byte
	var err error

	if prevId, err = encoding.FromHex(t.PrevTxId); err != nil {
		return nil
	}
	if from, err = encoding.FromHex(t.From); err != nil {
		return nil
	}
	if to, err = encoding.FromHex(t.To); err != nil {
		return nil
	}
	if sig, err = encoding.FromHex(t.Sig); err != nil {
		return nil
	}

	result := core.NewTx(t.Type, crypto.ID(from), crypto.ID(to), t.Amount, []byte(t.Payload), prevId, t.Uncompleted, []byte(t.Description))
	result.Sig = sig

	return result
}

// 从TxInfo转为JSON
func (t *TxJSON) FromTxInfo(info *TxInfo) {
	t.Version = info.Version
	t.Type = info.Type
	t.Uncompleted = info.Uncompleted
	t.Time = info.TimeUnix
	t.Id = encoding.ToHex(info.Id)

	t.From = info.From.ToHex()
	t.To = info.To.ToHex()
	t.Amount = info.Amount
	t.Sig = encoding.ToHex(info.Sig)

	// TODO
	t.Payload = string(info.Payload)
	t.PrevTxId = encoding.ToHex(info.PrevTxId)
	t.Description = string(info.Description)

	t.Height = info.Height
	t.BlockHash = encoding.ToHex(info.BlockHash)

}

////////////////////////////// RawTxJSON //////////////////////////////////////

type RawTxJSON struct {
	Type        uint8  `json:"type"`
	Uncompleted uint8  `json:"uncompleted"`

	To     string `json:"to"`   // To.ToHex()
	Amount uint32 `json:"amount"`

	Payload  string `json:"payload"`    // string(Payload)
	PrevTxId string `json:"prev_tx_id"` // hex

	Description string `json:"description"`
}

func (r *RawTxJSON) ToRawTx() (*raw.Tx, error) {
	result := &raw.Tx{}
	var err error

	// TODO
	err = err

	return result, nil
}
