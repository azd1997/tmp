package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/azd1997/ecoin/account/role"
	"github.com/pkg/errors"
	"io"
	"time"
	"unicode/utf8"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
)

const TxBasicLen = 11           // version + type + uncompleted + timeunix
const TxMaxDescriptionLen = 200 // 交易描述最多200字(rune)

// 诊断类型
const (
	DIAG_AUTO   = 1
	DIAG_DOCTOR = 2
)

// 交易类型
const (
	TX_AUTO = iota
	TX_COINBASE		// coinbase交易，用于给出块者发放奖励
	TX_GENERAL		// 通用转账交易
	TX_R2P			// 研究机构->病人的数据购买请求交易
	TX_P2R			// 病人->研究机构的数据购买交易回复
	TX_P2H			// 病人->医院的诊断请求交易
	TX_H2P			// 医院->病人的诊断回复交易
	TX_P2D			// 病人->医生的诊断请求交易
	TX_D2P			// 医生->病人的诊断回复交易
	TX_ARBITRATE	// 仲裁交易纠纷用
	TX_UPLOAD		// 上传数据摘要用
	TX_REGREQ		// 注册账号用	register request
	TX_REGRESP		// 注册账号用 register response
	numTXTypes
)

var TxTypes = map[uint8]string{
	TX_COINBASE:  "Coinbase",
	TX_GENERAL:   "General",
	TX_R2P:       "R2P",
	TX_P2R:       "P2R",
	TX_P2H:       "P2H",
	TX_H2P:       "H2P",
	TX_P2D:       "P2D",
	TX_D2P:       "D2P",
	TX_ARBITRATE: "Arbitrate",
	TX_UPLOAD: "Upload",		// 上传交易不二段，如果用户瞎上传会被扣分直至封号
	TX_REGREQ: "RegisterRequest",	// 注册交易也是二段交易
	TX_REGRESP: "RegisterResponse",
}

type ErrID2PublicKeyFailed struct {
	errID crypto.ID
}

func (err ErrID2PublicKeyFailed) Error() string {
	return fmt.Sprintf("[%s] ID2PublicKey failed\n", err.errID)
}

// Tx 交易的基本序列化格式(成员均使用基本数据格式)
// 每一笔交易都是一个交易单链表，只有在初始交易才需要填交易双方ID
// 和购买目标数据索引(Target)
// TxMsg 在Tx基础上包装通信头Header，用于广播
// TODO： Tx还应包含支付款项相关的成员变量，暂时不添加进去
type Tx struct {
	Version uint8	// 协议版本
	Type uint8		// 交易类型
	Uncompleted uint8	// 交易活动未完成? 0表示false(完成)， 1表示true(未完成)
	TimeUnix int64	// Unix时间戳
	Id crypto.Hash 		// 交易哈希

	// 对于TxCoinbase，From为空
	// 对于TxUpload/TxRegister，From为病人自己
	// 对于TxArbitrate，From为仲裁者，也就是worker账户
	From crypto.ID		// 发起者账户ID

	// 对于TxCoinbase，To为出块者自己
	// 对于TxUpload/TxRegister，To为目标医院
	// 对于TxArbitrate，To为空
	To crypto.ID		// 接收者账户ID

	Amount uint32	// 转账数额
	//TODO: 要检查转账数目，由于检查转账树木需要进行查询，所以在协议层没法检查，丢给上层去做

	Sig []byte		// 发起者的签名. TxCoinbase无需签名

	// Target 用来指定目标数据的索引
	// 目标数据的索引是通过TxUpload实现的
	// 所有数据上传时均填写好所有有关的索引信息，并且被所有worker节点缓存到数据库
	// 每个病人账户下边(包括worker节点存的病人)都有一张表存 <TxUploadHash, TxUpload>
	// 因此Target为TxUploadHash就可以满足查询需求
	//
	// 对于TxUpload本身而言，Target由上层进行序列化，而不在core层处理
	// 相信上层会对真正的Target进行处理，不到必须检查时不去检查
	//
	// 对于R2P、P2H、P2D，Target为要的TxUpload的哈希
	// 对于TxUpload而言，Target为上传的数据的索引信息的序列化结果
	//
	//
	// Reply 回复项
	// 对于TxUpload，为空
	// 对于TxR2P,Reply为空
	// 对于TxP2R,Reply为返回的解密信息的序列化，同样，这里也不检查
	// 对于TxP2H/TxP2D,Reply为发给医院/医生的解密信息序列化
	// 对于TxH2P/TxD2P,Reply为诊断结果
	// 对于TxArbitrate，Reply为仲裁结果
	// 对于TxRegister两段而言，Reply也表示注册信息和注册通过信息
	// 其解释交给上层去做
	//
	// 原先定义的Target []byte和Reply []byte，被合并为 Payload []byte
	Payload []byte

	// 关于来源交易历史，有两种做法
	// 一种是每次都包含过往的所有交易，并且为了减少空间消耗，尽量减少重复信息
	// 另一种是每次包含前次交易的Id，然后由上层(enode)去检查所有的来源交易
	// 另外一种做法是只填写来源交易的ID，由检查者自行查询
	// 前一种做法实现容易，每次都可以直接core.Tx自检查。但是额外消耗的网络资源不可忽视
	// 后一种做法比较精简，但是没办法在协议层去检查Tx的来源交易
	// 这里选择后一种
	// NOTICE: 考虑节省空间，只要是不是origin交易（初始交易），后续的交易可以将部分值域缺省
	// 不填就是默认原来的，填则可以进行修改
	PrevTxId crypto.Hash		// 最近的 来源交易 的哈希Id

	// Description 用来添加描述
	Description []byte	// 描述
}

func NewTx(typ uint8, from, to crypto.ID, amount uint32, payload []byte,
	prevTxId crypto.Hash, uncompleted uint8, description []byte) *Tx {
	tx := &Tx{
		// 定长
		Version:     V1,
		Type:        typ,
		Uncompleted: uncompleted,
		TimeUnix:    time.Now().Unix(),
		Amount:      amount,

		// 不定长
		Id:          nil,
		From:        from,
		To:          to,
		Sig:		 nil,
		Payload:payload,
		PrevTxId:        prevTxId,
		Description:  description,
	}

	// 设置Id
	id := tx.Hash()
	if id == nil {return nil}
	tx.Id = id

	return tx	// 待签名
}

//// Size 计算binary序列化的tx总占用长度，在获取Tx的时候能判断出其序列化后有多长
//// 但这里考虑方便性使用gob编码，因此Size方法暂时弃置，而且需要注意Size()中
//// TX_BASIC_LEN 还未加上变长成员变量的长度占位大小
//func (tx *Tx) Size() int {
//	if tx == nil {return 0}
//
//	l := TxBasicLen
//	l += len(tx.Id) + len(tx.From) + len(tx.To) + len(tx.Sig) +
//		len(tx.Description) + len(tx.Payload)
//
//	return l + tx.PrevTxId.Size()
//}

// String 按Json有缩进换行格式转为字符串，主要用于控制台打印和测试
// TODO: 真正有意义的、可读的String输出只能交给上层去做
// 真正实现时，需要switch case，对不同的交易类型作不同的输出
func (tx *Tx) String() string {
	return encoding.JsonMarshalIndentToString(tx)
}

// Hash 取哈希 取哈希时不算签名和Id在内，Hash将会作为Id
func (tx *Tx) Hash() crypto.Hash {
	txCopy := *tx
	//fmt.Println(txCopy)
	txCopy.Id, txCopy.Sig = nil, nil
	enced := txCopy.Encode()
	if enced == nil {return nil}
	//fmt.Println("TxId: ", crypto.HashD(enced))
	return crypto.HashD(enced)
}

// Sign 签名，对tx.Id签名，必须是在tx.SetId()之后
func (tx *Tx) Sign(priv *crypto.PrivateKey) error {
	sig, err := crypto.Sign(priv, tx.Id)
	if err != nil {
		return errors.Wrap(err, "Tx_Sign")
	}
	tx.Sig = sig.Serialize()
	return nil
}

// IsNextOf 判断当前交易是否是prevTx的下一步交易
func (tx *Tx) IsNextOf(prevTx *Tx) bool {
	// 1. 检查tx.PrevTxId
	if tx.PrevTxId == nil {
		return false
	}

	// 2. 检查tx.PrevTxId ?= prevTx.Id
	if !bytes.Equal(tx.PrevTxId, prevTx.Id) {
		return false
	}

	// 3. TODO ：检查其他各项内容是否匹配


	return true
}


///////////////////////////////////////////////////////////////////////////////////////////


// Verify 验证tx的有效性
// TODO: 注意：转账者余额、信誉分、target、reply有效性均没有校验，留给上层去做
// 这里主要对格式进行校验。
// 如果要求不能使用公钥传递在消息内，那么
func (tx *Tx) Verify() error {
	// 协议版本
	if tx.Version != V1 {
		return fmt.Errorf("invalid tx version %d", tx.Version)
	}
	// 哈希长度
	if len(tx.Id) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid hash length %d", len(tx.Id))
	}
	// 交易时间
	if tx.TimeUnix >= time.Now().Unix() {
		return fmt.Errorf("invalid tx timeunix %d", tx.TimeUnix)
	}
	// 描述不能过长
	if err := tx.verifyDescription(); err != nil {
		return err
	}

	// 交易类型
	switch tx.Type {
	case TX_COINBASE:
		if err := tx.verifyTxCoinBase(); err != nil {
			return err
		}
	case TX_GENERAL:
		if err := tx.verifyTxGeneral(); err != nil {
			return err
		}
	case TX_R2P:
		if err := tx.verifyTxGeneral(); err != nil {
			return err
		}
	case TX_P2R:
		if err := tx.verifyTxP2R(); err != nil {
			return err
		}
	case TX_P2H:
		if err := tx.verifyTxP2H(); err != nil {
			return err
		}
	case TX_H2P:
		if err := tx.verifyTxH2P(); err != nil {
			return err
		}
	case TX_P2D:
		if err := tx.verifyTxP2D(); err != nil {
			return err
		}
	case TX_D2P:
		if err := tx.verifyTxD2P(); err != nil {
			return err
		}
	case TX_ARBITRATE:
		if err := tx.verifyTxArbitrate(); err != nil {
			return err
		}
	case TX_UPLOAD:
		if err := tx.verifyTxUpload(); err != nil {
			return err
		}
	case TX_REGREQ:
		if err := tx.verifyTxRegreq(); err != nil {
			return err
		}
	case TX_REGRESP:
		if err := tx.verifyTxRegresp(); err != nil {
			return err
		}
	default:
		return errors.New("unknown tx type")
	}

	return nil
}

// tx.Type = Txcoinbase 检查
func (tx *Tx) verifyTxCoinBase() error {
	// 检查应空项
	if tx.From != crypto.ZeroID || tx.Sig != nil || tx.Payload != nil {
		return errors.New("TxCoinbase: From/Sig/Payload must be empty")
	}
	// 检查uncompleted:应为0(false)
	if tx.Uncompleted != 0 {
		return errors.New("TxCoinbase: Uncompleted should be 0(false)")
	}
	// 接收者是否有效: 长度/角色/解析
	if len(tx.To) != crypto.ID_LEN_WITH_ROLE {
		return errors.New("TxCoinbase: len(To) != 33(crypto.ID_LEN_WITH_ROLE)")
	}
	roleno := tx.To[0]
	if !role.IsARole(roleno) {
		return errors.New("TxCoinbase: From is not ARole")
	}
	toPublicKey := crypto.ID2PublicKey(tx.To)
	if toPublicKey == nil {
		return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.To}, "TxCoinbase")
	}
	// txId是否吻合
	if !bytes.Equal(tx.Hash(), tx.Id) {
		return errors.New("TxCoinbase: unmatched Id")
	}
	return nil
}

func (tx *Tx) verifyTxGeneral() error {
	// 检查应空项
	if tx.Payload != nil {
		return errors.New("TxGeneral: Payload must be empty")
	}
	// 检查uncompleted:应为0(false)
	if tx.Uncompleted != 0 {
		return errors.New("TxGeneral: Uncompleted should be 0(false)")
	}
	// 发起者是否有效：长度/解析
	if len(tx.From) != crypto.ID_LEN_WITH_ROLE {
		return errors.New("TxGeneral: len(From) != 33(crypto.ID_LEN_WITH_ROLE)")
	}
	fromPublicKey := crypto.ID2PublicKey(tx.From)
	if fromPublicKey == nil {
		return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.From}, "TxGeneral")
	}
	// 接收者是否有效: 长度/解析
	if len(tx.To) != crypto.ID_LEN_WITH_ROLE {
		return errors.New("TxGeneral: len(To) != 33(crypto.ID_LEN_WITH_ROLE)")
	}
	toPublicKey := crypto.ID2PublicKey(tx.To)
	if toPublicKey == nil {
		return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.To}, "TxGeneral")
	}
	// 检查签名
	sig, err := crypto.ParseSignatureS256(tx.Sig)
	if err != nil {
		return errors.Wrap(err, "TxGeneral")
	}
	if !sig.Verify(tx.Id, fromPublicKey) {
		return errors.New("TxGeneral: verify Sig failed")
	}
	// txId是否吻合
	if !bytes.Equal(tx.Hash(), tx.Id) {
		return errors.New("TxGeneral: unmatched Id")
	}
	return nil
}


// 检查TxR2P
func (tx *Tx) verifyTxR2P() error {
	// r2p要考虑两种情况:prev==nil 和prev != nil
	if tx.PrevTxId == nil {
		return tx.verifyTxR2P_0()
	} else {
		return tx.verifyTxR2P_1()
	}
}

// 对初始的TxR2P的检查
func (tx *Tx) verifyTxR2P_0() error {
	// 1. 检查uncompleted:应为0(false)
	if tx.Uncompleted != 0 {
		return errors.New("TxR2P(0): Uncompleted should be 0(false)")
	}
	// 2. 发起者是否有效：角色/长度/解析
	if len(tx.From) != crypto.ID_LEN_WITH_ROLE {
		return errors.New("TxR2P(0): len(From) != 33(crypto.ID_LEN_WITH_ROLE)")
	}
	if !role.IsResearcher(tx.From[0]) {
		return errors.New("TxR2P(0): From is not Researcher")
	}
	fromPublicKey := crypto.ID2PublicKey(tx.From)
	if fromPublicKey == nil {
		return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.From}, "TxR2P(0)")
	}
	// 3. 接收者是否有效: 角色/长度/解析
	if len(tx.To) != crypto.ID_LEN_WITH_ROLE {
		return errors.New("TxGeneral: len(To) != 33(crypto.ID_LEN_WITH_ROLE)")
	}
	if !role.IsPatient(tx.To[0]) {
		return errors.New("TxR2P(0): To is not Patient")
	}
	toPublicKey := crypto.ID2PublicKey(tx.To)
	if toPublicKey == nil {
		return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.To}, "TxR2P(0)")
	}
	// 4. 检查签名
	sig, err := crypto.ParseSignatureS256(tx.Sig)
	if err != nil {
		return errors.Wrap(err, "TxR2P(0)")
	}
	if !sig.Verify(tx.Id, fromPublicKey) {
		return errors.New("TxR2P(0): verify Sig failed")
	}
	// 5. 检查Target:必须能反序列化为TargetData，并且保证内部是哈希列表，至于TargetData是否有效，不管，给上层去验证
	td := &TargetData{}
	err = td.Decode(bytes.NewReader(tx.Payload))
	if err != nil {
		return errors.Wrap(err, "TxR2P(0)")
	}
	if !td.formatOK() {
		return errors.New("TxR2P(0): TargetData is not a Hash list")
	}
	// 6. txId是否吻合
	if !bytes.Equal(tx.Hash(), tx.Id) {
		return errors.New("TxR2P(0): unmatched Id")
	}

	return nil
}

// 对非初始的TxR2P的检查
// 需要注意：上层调用时还需要检查prev链路上的交易是否存在
//func (tx *Tx) verifyTxR2P_1() error {
//	// 1. 检查应空项 (设计为不允许更改TargetData)
//	if tx.Payload != nil || tx.From != crypto.ZeroID || tx.To != crypto.ZeroID {
//		return errors.New("TxR2P(1): From/To/Payload must be empty")
//	}
//	// 2. 迭代到初始R2P，并统计出现多少uncompleted，uncompleted不能超过3次
//	r2p0 := tx.PrevTxId
//	isr2p := 1		// 1表示是
//	sign := -1		// 用来改变isr2p
//	uncompletedByR := 0
//	for r2p0.PrevTxId != nil {
//		if isr2p == 1 {
//			if r2p0.Uncompleted == 0 {	// 0表示完成，这是不正常的
//				return errors.New("TxR2P(1): wrong tx history")
//			}
//			uncompletedByR++
//		}
//		r2p0 = r2p0.PrevTxId
//		isr2p = isr2p * sign
//	}
//	if tx.Uncompleted == 1 {uncompletedByR++}	// 如果tx这次是未完成的，那么需要加一
//	// 现在r2p0为初始交易
//	// TODO: NOTICE!! 初始交易的正确性不检查
//	// 现在要检查r2p0是不是r2p(看isr2p就行)以及uncompletedByR是否超过3
//	if isr2p == 0 || uncompletedByR > 3 {
//		return errors.New("TxR2P(1): wrong tx history")
//	}
//	// 2. 发起者是否有效：角色/长度/解析
//	if len(r2p0.From) != crypto.ID_LEN_WITH_ROLE {
//		return errors.New("TxR2P(1): len(r2p0.From) != 33(crypto.ID_LEN_WITH_ROLE)")
//	}
//	if !role.IsResearcher(r2p0.From[0]) {
//		return errors.New("TxR2P(1): r2p0.From is not Researcher")
//	}
//	fromPublicKey := crypto.ID2PublicKey(tx.From)
//	if fromPublicKey == nil {
//		return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.From}, "TxR2P(1)")
//	}
//	// 3. 接收者是否有效: 角色/长度/解析
//	if len(r2p0.To) != crypto.ID_LEN_WITH_ROLE {
//		return errors.New("TxR2P(1): len(r2p0.To) != 333(crypto.ID_LEN_WITH_ROLE)4")
//	}
//	if !role.IsPatient(r2p0.To[0]) {
//		return errors.New("TxR2P(1): r2p0.To is not Patient")
//	}
//	toPublicKey := crypto.ID2PublicKey(tx.To)
//	if toPublicKey == nil {
//		return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.To}, "TxR2P(1)")
//	}
//	// 4. 检查签名
//	sig, err := crypto.ParseSignatureS256(tx.Sig)
//	if err != nil {
//		return errors.Wrap(err, "TxR2P(1)")
//	}
//	if !sig.Verify(tx.Id, fromPublicKey) {
//		return errors.New("TxR2P(1): verify Sig failed")
//	}
//	// 6. txId是否吻合
//	if !bytes.Equal(tx.Hash(), tx.Id) {
//		return errors.New("TxR2P(1): unmatched Id")
//	}
//
//	return nil
//}

// 对非初始的TxR2P的检查
// 需要注意：上层调用时还需要检查prev链路上的交易是否存在
// TODO： 对于非初始的交易，允许缺省参数
func (tx *Tx) verifyTxR2P_1() error {
	//// 1. 检查应空项 (设计为不允许更改TargetData)
	//if tx.Payload != nil || tx.From != crypto.ZeroID || tx.To != crypto.ZeroID {
	//	return errors.New("TxR2P(1): From/To/Payload must be empty")
	//}
	//// 2. 迭代到初始R2P，并统计出现多少uncompleted，uncompleted不能超过3次
	//r2p0 := tx.PrevTxId
	//isr2p := 1		// 1表示是
	//sign := -1		// 用来改变isr2p
	//uncompletedByR := 0
	//for r2p0.PrevTxId != nil {
	//	if isr2p == 1 {
	//		if r2p0.Uncompleted == 0 {	// 0表示完成，这是不正常的
	//			return errors.New("TxR2P(1): wrong tx history")
	//		}
	//		uncompletedByR++
	//	}
	//	r2p0 = r2p0.PrevTxId
	//	isr2p = isr2p * sign
	//}
	//if tx.Uncompleted == 1 {uncompletedByR++}	// 如果tx这次是未完成的，那么需要加一
	//// 现在r2p0为初始交易
	//// TODO: NOTICE!! 初始交易的正确性不检查
	//// 现在要检查r2p0是不是r2p(看isr2p就行)以及uncompletedByR是否超过3
	//if isr2p == 0 || uncompletedByR > 3 {
	//	return errors.New("TxR2P(1): wrong tx history")
	//}
	//// 2. 发起者是否有效：角色/长度/解析
	//if len(r2p0.From) != crypto.ID_LEN_WITH_ROLE {
	//	return errors.New("TxR2P(1): len(r2p0.From) != 33(crypto.ID_LEN_WITH_ROLE)")
	//}
	//if !role.IsResearcher(r2p0.From[0]) {
	//	return errors.New("TxR2P(1): r2p0.From is not Researcher")
	//}
	//fromPublicKey := crypto.ID2PublicKey(tx.From)
	//if fromPublicKey == nil {
	//	return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.From}, "TxR2P(1)")
	//}
	//// 3. 接收者是否有效: 角色/长度/解析
	//if len(r2p0.To) != crypto.ID_LEN_WITH_ROLE {
	//	return errors.New("TxR2P(1): len(r2p0.To) != 333(crypto.ID_LEN_WITH_ROLE)4")
	//}
	//if !role.IsPatient(r2p0.To[0]) {
	//	return errors.New("TxR2P(1): r2p0.To is not Patient")
	//}
	//toPublicKey := crypto.ID2PublicKey(tx.To)
	//if toPublicKey == nil {
	//	return errors.Wrap(ErrID2PublicKeyFailed{errID:tx.To}, "TxR2P(1)")
	//}
	//// 4. 检查签名
	//sig, err := crypto.ParseSignatureS256(tx.Sig)
	//if err != nil {
	//	return errors.Wrap(err, "TxR2P(1)")
	//}
	//if !sig.Verify(tx.Id, fromPublicKey) {
	//	return errors.New("TxR2P(1): verify Sig failed")
	//}
	//// 6. txId是否吻合
	//if !bytes.Equal(tx.Hash(), tx.Id) {
	//	return errors.New("TxR2P(1): unmatched Id")
	//}

	return nil
}

// TODO: 剩余待实现

func (tx *Tx) verifyTxP2R() error {
	return nil
}

func (tx *Tx) verifyTxP2H() error {
	return nil
}

func (tx *Tx) verifyTxH2P() error {
	return nil
}

func (tx *Tx) verifyTxP2D() error {
	return nil
}

func (tx *Tx) verifyTxD2P() error {
	return nil
}

func (tx *Tx) verifyTxArbitrate() error {
	return nil
}

func (tx *Tx) verifyTxUpload() error {
	return nil
}

func (tx *Tx) verifyTxRegreq() error {
	return nil
}

func (tx *Tx) verifyTxRegresp() error {
	return nil
}



func (tx *Tx) verifyDescription() error {
	count := utf8.RuneCount(tx.Description)
	if count == -1 {
		return fmt.Errorf("invalid description, not utf-8 encoding")
	}
	if count > TxMaxDescriptionLen {
		return fmt.Errorf("invalid description length %d", count)
	}

	return nil
}

/////////////////////////////////////////////////////

// 由于Tx的结构目前没有完全固定，且成员较多，使用gob方便一些

// Encode 序列化编码
func (tx *Tx) Encode() []byte {
	res, _ := encoding.GobEncode(tx)
	return res
}

// Decode 解码
func (tx *Tx) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(tx)
	if err != nil {
		return errors.Wrap(err, "Tx_Decode")
	}
	return nil
}

/////////////////////////////////////////////////////

