package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
	"time"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
)

// BlockHeader 区块头
// 和其他区块链的区块头稍有不同的是：
// 这里的MerkleRoot不含Coinbase交易。
// 区块的证明信息由CreateBy, len(block.Txs)-1, MerkleRoot三部分构成，而不额外设置变量
type BlockHeader struct {
	Version      uint8	// V1版本使用POT版本，之后如果扩展其他共识，切换版本
	Time         int64	// ns级		// 因为定时器需要根据区块构造时间来工作，所以这里取ns级
	Hash crypto.Hash	// 当前区块哈希
	PrevHash     crypto.Hash
	MerkleRoot crypto.Hash		// 交易Merkle树的根哈希值
	CreateBy        crypto.ID		// 创建者ID
}

func NewBlockHeaderV1(prevHash crypto.Hash, createBy crypto.ID, merkleRoot crypto.Hash) *BlockHeader {
	bh := &BlockHeader{
		Version:    V1,
		Time:       time.Now().UnixNano(),		// TODO： utils.TimeToString()需要做修改
		PrevHash:   prevHash,
		MerkleRoot: merkleRoot,
		CreateBy:   createBy,
	}
	bhBytes, err := encoding.GobEncode(bh)
	if err != nil {return nil}
	bh.Hash = crypto.HashD(bhBytes)
	return bh
}

func (b *BlockHeader) Decode(data io.Reader) error {
	// Version
	if err := binary.Read(data, binary.BigEndian, &b.Version); err != nil {
		return errors.Wrap(err, "BlockHeader_Decode: Version")
	}
	// Time
	if err := binary.Read(data, binary.BigEndian, &b.Time); err != nil {
		return errors.Wrap(err, "BlockHeader_Decode: Time")
	}
	// Hash
	b.Hash = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, b.Hash); err != nil {
		return errors.Wrap(err, "BlockHeader_Decode: Hash")
	}
	// PrevHash
	b.PrevHash = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, b.PrevHash); err != nil {
		return errors.Wrap(err, "BlockHeader_Decode: PrevHash")
	}
	// MerkleRoot
	b.MerkleRoot = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, b.MerkleRoot); err != nil {
		return errors.Wrap(err, "BlockHeader_Decode: MerkleRoot")
	}
	//fmt.Println(b.MerkleRoot)
	// CreateBy
	createByBytes := make([]byte, crypto.ID_LEN_WITH_ROLE)
	if err := binary.Read(data, binary.BigEndian, createByBytes); err != nil {
		return errors.Wrap(err, "BlockHeader_Decode: createByBytes")
	}
	b.CreateBy = crypto.ID(createByBytes)

	return nil
}

func (b *BlockHeader) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// Version
	binary.Write(buf, binary.BigEndian, b.Version)
	// Time
	binary.Write(buf, binary.BigEndian, b.Time)
	// Hash
	binary.Write(buf, binary.BigEndian, b.Hash)
	// PrevHash
	binary.Write(buf, binary.BigEndian, b.PrevHash)
	// MerkleRoot
	binary.Write(buf, binary.BigEndian, b.MerkleRoot)
	// CreateBy
	binary.Write(buf, binary.BigEndian, []byte(b.CreateBy))

	return buf.Bytes()
}

// 浅拷贝，只拷贝内部成员的引用
func (b *BlockHeader) ShallowCopy() *BlockHeader {
	return &BlockHeader{
		Version:      b.Version,
		Time:         b.Time,
		PrevHash:     b.PrevHash,
		Hash:b.Hash,
		MerkleRoot:b.MerkleRoot,
		CreateBy:b.CreateBy,
	}
}


func (b *BlockHeader) Verify() error {
	if b.Version != V1 {
		return fmt.Errorf("invalid header version")
	}

	if len(b.PrevHash) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid LastHash %X", b.PrevHash)
	}

	if len(b.CreateBy) != crypto.ID_LEN_WITH_ROLE {
		return fmt.Errorf("invalid Worker %s", b.CreateBy)
	}

	if b.MerkleRoot == nil {
		return fmt.Errorf("nil MerkleRoot")
	}

	if !bytes.Equal(b.MerkleRoot, EmptyMerkleRoot) && len(b.MerkleRoot) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid MerkleRoot %X", b.MerkleRoot)
	}

	return nil
}

func (b *BlockHeader) CalcHash() []byte {
	bCopy := *b
	bCopy.Hash = nil
	res, _ := encoding.GobEncode(&bCopy)
	h := crypto.HashD(res)
	return h
}

// 是否为空默克尔根
func (b *BlockHeader) IsEmptyMerkleRoot() bool {
	return bytes.Equal(b.MerkleRoot, EmptyMerkleRoot)
}

// 设置空默克尔跟
func (b *BlockHeader) SetEmptyMerkleRoot() {
	b.MerkleRoot = EmptyMerkleRoot
}

func (b *BlockHeader) String() string {
	return fmt.Sprintf("Version %d Time %s Hash %X LastHash %X MerkleRoot %X CreateBy %s",
		b.Version, time.Unix(0, b.Time), b.Hash, b.PrevHash, b.MerkleRoot, b.CreateBy)
}
