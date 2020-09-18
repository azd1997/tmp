/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/9/13 16:00
* @Description: The file is for
***********************************************************************/

package potpb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"

	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/utils"
)

type Proof struct {
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
	// 这个证明是为了index这个区块准备的。index从1开始
	Index uint64
	// 签名，对除Sig以外的字段作签名
	Sig []byte
}

func NewProof(from crypto.ID, txsNum uint32, txsMerkle crypto.Hash, base crypto.Hash,
	privateKey *crypto.PrivateKey) (*Proof, error) {

	rawProof := &Proof{
		From:from,
		TxsNum:txsNum,
		TxsMerkle:txsMerkle,
		Base:base,
	}
	b := rawProof.Encode()
	sig, err := crypto.Sign(privateKey, b)
	if err != nil {
		return nil, err
	}
	sigBytes := sig.Serialize()
	rawProof.Sig = sigBytes
	return rawProof, nil
}

func (p *Proof) Encode() []byte {
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
	// Index
	binary.Write(buf, binary.BigEndian, p.Index)

	// 签名
	sigLen := uint16(len(p.Sig))
	binary.Write(buf, binary.BigEndian, sigLen)
	if sigLen != 0 {
		binary.Write(buf, binary.BigEndian, p.Sig)
	}

	return buf.Bytes()
}

func (p *Proof) Decode(data io.Reader) error {
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
	// Index
	if err := binary.Read(data, binary.BigEndian, &p.Index); err != nil {
		return errors.Wrap(err, "PoTProof_Decode: Index")
	}
	// 签名
	sigLen := uint16(0)
	if err := binary.Read(data, binary.BigEndian, &sigLen); err != nil {
		return errors.Wrap(err, "PoTProof_Decode: sigLen")
	}
	if sigLen > 0 {
		p.Sig = make([]byte, sigLen)
		if err := binary.Read(data, binary.BigEndian, p.Sig); err != nil {
			return errors.Wrap(err, "PoTProof_Decode: Sig")
		}
	}

	return nil
}

// Verify只作格式检查
func (p *Proof) Verify() error {
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

func (p *Proof) String() string {
	return fmt.Sprintf("From %s TxsNum %d TxsMerkle %X Base %X Index %d Sig %X",
		p.From.String(), p.TxsNum, p.TxsMerkle, p.Base, p.Index, p.Sig)
}

func (p *Proof) GreaterThan(ap *Proof) bool {
	if p.Index != ap.Index || !bytes.Equal(p.Base, ap.Base) {
		log.Fatalln("cannot compare!")
	}

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

