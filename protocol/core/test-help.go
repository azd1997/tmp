package core

import (
	"bytes"
	"fmt"
	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/log"
	"math"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func errorf(prefix string, expect interface{}, result interface{}) error {
	return fmt.Errorf("%s verify failed, expect %v, result %v", prefix, expect, result)
}

type TxParams struct {
	txType uint8
	amount uint32
	payload        []byte
	description []byte
	privateKey     *crypto.PrivateKey
	from crypto.ID
	to crypto.ID
}

// 测试时，调用处需作相应的修改

func NewTxParams(txType uint8) *TxParams {
	switch txType {
	case TX_COINBASE:
		return newTxCoinbaseParams()
	case TX_GENERAL:
		return newTxGeneralParams()
	default:	// TODO: 以后再加其他类型的交易参数
		log.Error("unknown tx type")
		return nil
	}
}

func newTxCoinbaseParams() *TxParams {
	toPriv, _ := crypto.NewPrivateKeyS256()
	to := crypto.PrivateKey2ID(toPriv, role.HOSPITAL)

	return &TxParams{
		txType:TX_COINBASE,
		amount:randNum(),
		description: []byte("[DESCRIPTION] " + string(randBytes())),
		to: to,
	}
}

func newTxGeneralParams() *TxParams {
	fromPriv, _ := crypto.NewPrivateKeyS256()
	from := crypto.PrivateKey2ID(fromPriv, role.PATIENT)
	toPriv, _ := crypto.NewPrivateKeyS256()
	to := crypto.PrivateKey2ID(toPriv, role.HOSPITAL)

	return &TxParams{
		txType:TX_GENERAL,
		amount:randNum(),
		description: []byte("[DESCRIPTION] " + string(randBytes())),
		privateKey:fromPriv,
		from:from,
		to: to,
	}
}

func newTxParams() *TxParams {
	fromPriv, _ := crypto.NewPrivateKeyS256()
	from := crypto.PrivateKey2ID(fromPriv, role.PATIENT)
	toPriv, _ := crypto.NewPrivateKeyS256()
	to := crypto.PrivateKey2ID(toPriv, role.HOSPITAL)

	return &TxParams{
		txType:TX_GENERAL,
		amount:randNum(),
		payload:       []byte("[PAYLOAD] " + string(randBytes())),
		description: []byte("[DESCRIPTION] " + string(randBytes())),
		privateKey:fromPriv,
		from:from,
		to: to,
	}
}

// TODO: 更多交易参数构造方式

func GenTxFromParams(param *TxParams) *Tx {
	tx := NewTx(param.txType, param.from, param.to, param.amount, param.payload, nil, 0, param.description)
	if param.privateKey != nil {
		tx.Sign(param.privateKey)
	}
	return tx
}

func CheckTx(tx *Tx, tp *TxParams) error {
	if tx.Version != V1 {
		return errorf("protocol version", V1, tx.Version)
	}
	if !bytes.Equal(tx.Payload, tp.payload) {
		return errorf("tx payload", tp.payload, tx.Payload)
	}
	if !bytes.Equal(tx.Description, tp.description) {
		return errorf("tx description", tp.description, tx.Description)
	}
	if tx.From != tp.from {
		return errorf("tx from", tp.from, tx.From)
	}
	if tx.To != tp.to {
		return errorf("tx to", tp.to, tx.To)
	}

	return nil
}

type BlockHeaderParams struct {
	prevHash crypto.Hash
	createby    crypto.ID
	txMerkleRoot   crypto.Hash
}

func NewBlockHeaderParams() *BlockHeaderParams {
	creatorPrivKey, _ := crypto.NewPrivateKeyS256()
	creator := crypto.PrivateKey2ID(creatorPrivKey, role.HOSPITAL)

	return &BlockHeaderParams{
		prevHash:randHash(),
		txMerkleRoot:randHash(),
		createby:creator,
	}
}

func GenBlockHeaderFromParams(param *BlockHeaderParams) *BlockHeader {
	blockHeader := NewBlockHeaderV1(param.prevHash, param.createby, param.txMerkleRoot)
	return blockHeader
}

func CheckBlockHeader(b *BlockHeader, bp *BlockHeaderParams) error {
	if b.Version != V1 {
		return errorf("block version", V1, b.Version)
	}
	if !bytes.Equal(b.PrevHash, bp.prevHash) {
		return errorf("block prev hash", bp.prevHash, b.PrevHash)
	}
	if b.CreateBy != bp.createby {
		return errorf("block creator", bp.createby, b.CreateBy)
	}
	if !bytes.Equal(b.MerkleRoot, bp.txMerkleRoot) {
		return errorf("block tx merkle root", bp.txMerkleRoot, b.MerkleRoot)
	}

	return nil
}

type BlockParams struct {
	*BlockHeaderParams
	TxsParams []*TxParams
}

func NewBlockParams(empty bool) *BlockParams {
	headerParams := NewBlockHeaderParams()

	var txsParams []*TxParams
	if !empty {
		txNum := rand.Intn(10) + 1 // at least one tx
		for i := 0; i < txNum; i++ {
			txsParams = append(txsParams, NewTxParams(uint8(rand.Intn(2) + 1)))	// TODO：这里暂且只随机生成前两种交易(代号1/2)
		}
	} else {
		headerParams.txMerkleRoot = EmptyMerkleRoot
	}

	return &BlockParams{
		BlockHeaderParams: headerParams,
		TxsParams:        txsParams,
	}
}

func GenBlockFromParams(bp *BlockParams) *Block {
	blockHeader := GenBlockHeaderFromParams(bp.BlockHeaderParams)

	txs := []*Tx{}
	for _, param := range bp.TxsParams {
		txs = append(txs, GenTxFromParams(param))
	}

	return NewBlock(blockHeader, txs)
}

func CheckBlock(b *Block, bp *BlockParams) error {
	if err := CheckBlockHeader(b.BlockHeader, bp.BlockHeaderParams); err != nil {
		return err
	}

	if len(b.Txs) != len(bp.TxsParams) {
		return errorf("tx size", len(bp.TxsParams), len(b.Txs))
	}
	for i := 0; i < len(bp.TxsParams); i++ {
		if err := CheckTx(b.Txs[i], bp.TxsParams[i]); err != nil {
			return nil
		}
	}

	return nil
}

func randBytes() []byte {
	// copy from https://golang.org/pkg/math/rand/#Rand Example
	strs := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes definitely",
		"You may rely on it",
		"As I see it yes",
		"Most likely",
		"Outlook good",
		"Yes",
		"Signs point to yes",
		"Reply hazy try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
	}
	return []byte(fmt.Sprintf("%s -- %d",
		strs[rand.Intn(len(strs))],
		time.Now().UnixNano()))
}

func randNum() uint32 {
	return rand.Uint32()
}

func randHash() crypto.Hash {
	h := make([]byte, crypto.HASH_LENGTH)
	for i:=0; i<len(h); i++ {
		h[i] = uint8(rand.Intn(math.MaxUint8))
	}
	return h
}
