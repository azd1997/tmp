package core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/pkg/errors"
	"io"
)

// 为了支持多种共识协议，定义Proof接口
type Proof interface {

}


//////////////////////////////////// POT ///////////////////////////////////


// PoT为winner's PotMsg，主要包含交易数与交易总哈希，其实可以通过区块交易列表取长度和merkleroot得到
// TODO：暂时不考虑PotMsg多次传播过程中被篡改的问题，有待解决
type PoTProof struct {
	From crypto.ID
	TxsNum uint32	// 有效交易数
	// 有效交易Merkle树根哈希。由上层去计算根哈希，这里不管
	// 只要是根据交易列表得到的能够证明交易列表唯一性的标识即可
	// 所以这里不限定方法。（目前采用上层取默克尔根的做法）
	TxsMerkle crypto.Hash
	// 指该Proof是基于本地的最高区块（得告诉别人你是不是基于这个区块），
	// 如果胜出了，所构建的区块的PrevHash必须是Base
	// 另外一种可选的做法是提供区块高度信息。但那样的话区块头结构也需要包含高度信息
	// 否则无法自解释自证明
	Base crypto.Hash
}

func NewPoTProof(from crypto.ID, txsNum uint32, txsMerkle crypto.Hash, base crypto.Hash) *PoTProof {
	return &PoTProof{
		From:from,
		TxsNum:txsNum,
		TxsMerkle:txsMerkle,
		Base:base,
	}
}

func (p *PoTProof) Encode() []byte {
	buf := utils.GetBuf()
	defer utils.ReturnBuf(buf)

	// From
	binary.Write(buf, binary.BigEndian, []byte(p.From))
	// TxsNum
	binary.Write(buf, binary.BigEndian, p.TxsNum)
	// TxsMerkle
	binary.Write(buf, binary.BigEndian, p.TxsMerkle)
	// Base
	binary.Write(buf, binary.BigEndian, p.Base)

	return buf.Bytes()
}

func (p *PoTProof) Decode(data io.Reader) error {
	// From
	fromBytes := make([]byte, crypto.ID_LEN_WITH_ROLE)
	if err := binary.Read(data, binary.BigEndian, fromBytes); err != nil {
		return errors.Wrap(err, "PoTProof_Decode: fromBytes")
	}
	p.From = crypto.ID(fromBytes)
	// TxsNum
	if err := binary.Read(data, binary.BigEndian, &p.TxsNum); err != nil {
		return errors.Wrap(err, "PoTProof_Decode: TxsNum")
	}
	// TxsMerkle
	p.TxsMerkle = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, p.TxsMerkle); err != nil {
		return errors.Wrap(err, "PoTProof_Decode: TxsMerkle")
	}
	// Base
	p.Base = make([]byte, crypto.HASH_LENGTH)
	if err := binary.Read(data, binary.BigEndian, p.Base); err != nil {
		return errors.Wrap(err, "PoTProof_Decode: Base")
	}

	return nil
}

// Verify只作格式检查
func (p *PoTProof) Verify() error {
	// 1. 检查From的角色
	if !role.IsARole(p.From.RoleNo()) {
		return errors.New("PoTProof_Verify: not a A role")
	}
	// 2. 检查TxsMerkle
	if len(p.TxsMerkle) != crypto.ID_LEN_WITH_ROLE {
		return errors.New("PoTProof_Verify: invalid TxsMerkle length")
	}
	// 3. 检查Base
	if len(p.Base) != crypto.ID_LEN_WITH_ROLE {
		return errors.New("PoTProof_Verify: invalid Base length")
	}

	return nil
}

func (p *PoTProof) String() string {
	return fmt.Sprintf("From %s TxsNum %d TxsMerkle %X Base %X",
		p.From.String(), p.TxsNum, p.TxsMerkle, p.Base)
}

func (p *PoTProof) GreaterThan(ap *PoTProof) bool {
	if p.TxsNum == ap.TxsNum {	// 交易数相同
		// Merkle根相同（这是有可能的，但概率极低。
		// 这是因为哪怕收集了完全一样的有效交易，但插入的顺序也很难一致，
		// 计算出的Merkle根就不同）
		if !bytes.Equal(p.TxsMerkle, ap.TxsMerkle) {
			for i:=0; i<crypto.HASH_LENGTH; i++ {
				if p.TxsMerkle[i] > ap.TxsMerkle[i] {
					return true
				}
			}
			return false
		}
		// 实在是连Merkle根都相同，那就直接比较ID以及Base
		// 为方便起见，直接取Base第一个字节，查看其奇偶性
		// 这样也保证了一定的公平性
		if p.Base[0] % 2 == 0 {
			return string(p.From) > string(ap.From)
		} else {
			return string(p.From) < string(ap.From)
		}

	}
	return p.TxsNum > ap.TxsNum
}

//////////////////////////////////// POT ///////////////////////////////////

type PoWProof struct {

}












