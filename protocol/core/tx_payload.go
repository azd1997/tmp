package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"time"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
)


type Payload interface {
	String() string
	Encode() []byte
	formatOK() bool
	// 有效性只能上层调用者去检查
}

/////////////////////////////////////////////////////////////////////////////////////

// 对于TxR2P, Payload需填TargetData(为TxUpload哈希)。至于这些哈希定位到的数据怎么整合不归这里管

type TargetData struct {
	Hashes []crypto.Hash	// TxUpload的哈希Id
}

// TODO： 实现所有的String，不能使用json，得手动输出格式
// 为了简便，Payload的String都打到一行上去，不然的话，缩进很难控制

func (td *TargetData) String() string {
	res := new(bytes.Buffer)
	res.WriteString("TargetData=>{Hashes=>{")
	n := len(td.Hashes)
	for i:=0; i<n; i++ {
		if i < n-1 {
			res.WriteString(fmt.Sprintf("%x, ", td.Hashes[i]))
		} else {
			res.WriteString(fmt.Sprintf("%x", td.Hashes[i]))
		}
	}
	res.WriteString("}}")
	return res.String()
}

func (td *TargetData) Encode() []byte {
	res, _ := encoding.GobEncode(td)
	return res
}

func (td *TargetData) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(td)
	if err != nil {return errors.Wrap(err, "TargetData_Decode")}
	return nil
}

// 检查targetData是否为哈希列表
func (td *TargetData) formatOK() bool {
	n := len(td.Hashes)
	for i:=0; i<n; i++ {
		if len(td.Hashes[i]) != crypto.HASH_LENGTH {
			return false
		}
	}
	return true
}

////////////////////////////////////////////////////////////////////

// 对于TxP2R, Payload需填TargetKey。

type TargetKey struct {
	Keys [][]byte	// 每段目标数据对应的解密密钥信息
}

func (tk *TargetKey) String() string {
	res := new(bytes.Buffer)
	res.WriteString("TargetKey=>{Keys=>{")
	n := len(tk.Keys)
	for i:=0; i<n; i++ {
		if i < n-1 {
			res.WriteString(fmt.Sprintf("%x, ", tk.Keys[i]))
		} else {
			res.WriteString(fmt.Sprintf("%x", tk.Keys[i]))
		}
	}
	res.WriteString("}}")
	return res.String()
}

func (tk *TargetKey) Encode() []byte {
	res, _ := encoding.GobEncode(tk)
	return res
}

func (tk *TargetKey) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(tk)
	if err != nil {return errors.Wrap(err, "TargetKey_Decode")}
	return nil
}

// 检查targetKey是否至少有一个
func (tk *TargetKey) formatOK() bool {
	return len(tk.Keys) >= 1
}

////////////////////////////////////////////////////////////////////

// 对于TxP2H/P2D, Payload需填TargetDataWithKey(为TxUpload哈希)。至于这些哈希定位到的数据怎么整合不归这里管


type TargetDataWithKey struct {
	Hashes []crypto.Hash	// TxUpload的哈希Id
	Keys [][]byte	// 对应的解密方法(解密密钥及可能的打乱重复方式)
	// keys的排列规则是 m = len(keys) <= len(hashes) = n
	// 从前向后依次对应
	// keys[m-1] 对应 hashes[m-1:n-1]
	// 因此，如果懒得换密码，可以直接只填一个key
}

func (tdk *TargetDataWithKey) String() string {
	res := new(bytes.Buffer)
	res.WriteString("TargetDataWithKey=>{Hashes=>{")
	n := len(tdk.Hashes)
	for i:=0; i<n; i++ {
		if i < n-1 {
			res.WriteString(fmt.Sprintf("%x, ", tdk.Hashes[i]))
		} else {
			res.WriteString(fmt.Sprintf("%x", tdk.Hashes[i]))
		}
	}
	res.WriteString("}, Keys=>{")
	n = len(tdk.Keys)
	for i:=0; i<n; i++ {
		if i < n-1 {
			res.WriteString(fmt.Sprintf("%x, ", tdk.Keys[i]))
		} else {
			res.WriteString(fmt.Sprintf("%x", tdk.Keys[i]))
		}
	}
	res.WriteString("}}")
	return res.String()
}

func (tdk *TargetDataWithKey) Encode() []byte {
	res, _ := encoding.GobEncode(tdk)
	return res
}

func (tdk *TargetDataWithKey) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(tdk)
	if err != nil {return errors.Wrap(err, "TargetDataWithKey_Decode")}
	return nil
}

func (tdk *TargetDataWithKey) formatOK() bool {
	n := len(tdk.Hashes)

	for i:=0; i<n; i++ {
		if len(tdk.Hashes[i]) != crypto.HASH_LENGTH {
			return false
		}
	}

	// 检查keys格式
	if len(tdk.Keys) == 0 || len(tdk.Keys) > n {
		return false
	}

	return true
}


////////////////////////////////////////////////////////////////////////////

// 对于TxH2P/D2P，Payload存的是诊断结果与建议

type TargetDiagnosis struct {
	Diags [][]byte	// 每段目标数据对应的诊断结果，如果总体给出诊断，则 len(diags) == 1
}

func (td *TargetDiagnosis) String() string {
	res := new(bytes.Buffer)
	res.WriteString("TargetDiagnosis=>{Diags=>{")
	n := len(td.Diags)
	for i:=0; i<n; i++ {
		if i < n-1 {
			res.WriteString(fmt.Sprintf("%x, ", td.Diags[i]))
		} else {
			res.WriteString(fmt.Sprintf("%x", td.Diags[i]))
		}
	}
	res.WriteString("}}")
	return res.String()
}

func (td *TargetDiagnosis) Encode() []byte {
	res, _ := encoding.GobEncode(td)
	return res
}

func (td *TargetDiagnosis) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(td)
	if err != nil {return errors.Wrap(err, "TargetDiagnosis_Decode")}
	return nil
}

// 检查targetDiagnosis是否至少有一个
func (td *TargetDiagnosis) formatOK() bool {
	return len(td.Diags) >= 1
}


///////////////////////////////////////////////////////////////////////////


// 对于TxArbitrate而言，Payload存仲裁结果，与targetdata保持等长，仲裁结果为每个数据是否都能解开

type TargetArbitrate struct {
	Arbs []bool	// R-P交易三次僵持之后仲裁每段target数据是否能被解开， len(Arbs) = len(Hashes)
	Bad bool	// Bad标志是交易双方哪一方作恶?默认false为R，true则表示P作恶。本地根据此再作记录与处理
	// R作恶则R的支付款项(冻结在中转池中)一半给P，一半给仲裁者，此外还要处罚R
	// P作恶则处罚P，扣信用积分
}

func (ta *TargetArbitrate) String() string {
	res := new(bytes.Buffer)
	res.WriteString("TargetArbitrate=>{Bad=>{")
	if ta.Bad {
		res.WriteString("P")
	} else {
		res.WriteString("R")
	}
	res.WriteString("}, Arbs=>{")

	n := len(ta.Arbs)
	for i:=0; i<n; i++ {
		if i < n-1 {
			res.WriteString(fmt.Sprintf("%v, ", ta.Arbs[i]))
		} else {
			res.WriteString(fmt.Sprintf("%v", ta.Arbs[i]))
		}
	}
	res.WriteString("}}")
	return res.String()
}

func (ta *TargetArbitrate) Encode() []byte {
	res, _ := encoding.GobEncode(ta)
	return res
}

func (ta *TargetArbitrate) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(ta)
	if err != nil {return errors.Wrap(err, "TargetArbitrate_Decode")}
	return nil
}

func (ta *TargetArbitrate) formatOK() bool {
	return true
}


////////////////////////////////////////////////////////////////////////


// 对于TxUpload，Payload存该条数据索引信息及其他信息

type TargetInfo struct {
	StoreAt string	// 存储到的那个数据仓库的地址
	Type uint8		// 所上传的数据的类型: 心电/温度/...
	Ill	uint8			// 疾病类型，0为健康，1或者更高代表各种类型的疾病，暂时没有定义
	TimeStart int64	// 该段数据采集的开始时间	Unix时间戳
	TimeEnd  int64	// 该段数据采集的结束时间
	Num int64			// 这段数据中总共的数据点数，例如1个小时区间的数据被打包在一起，总共有60条数据在里边
}

func (ti *TargetInfo) String() string {
	return fmt.Sprintf("TargetInfo=>{StoreAt=>{\"%s\"}, Type=>{%d}, Ill=>{%d}, TimeStart=>{%d}, TimeEnd=>{%d}, Num=>{%d}}",
		ti.StoreAt, ti.Type, ti.Ill, ti.TimeStart, ti.TimeEnd, ti.Num)
}

func (ti *TargetInfo) Encode() []byte {
	res, _ := encoding.GobEncode(ti)
	return res
}

func (ti *TargetInfo) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(ti)
	if err != nil {return errors.Wrap(err, "TargetInfo_Decode")}
	return nil
}

func (ti *TargetInfo) formatOK() bool {
	return ti.StoreAt != "" && ti.Num > 0 && ti.TimeStart < ti.TimeEnd && ti.TimeEnd < time.Now().Unix()
}


//////////////////////////////////////////////////////////////////////////////

// TODO:

// 对于TxRegreq，Payload存注册信息。而且对于注册信息，病人可以使用注册的那家医院的公钥加密，以保证隐私
// 而对于其他三类对外提供服务的角色，必须明文注册其资质，也就是说区块链可以查到它们的身份信息

type RegisterInfo struct {

}

// TODO
func (ri *RegisterInfo) String() string {
	return ""
}

func (ri *RegisterInfo) Encode() []byte {
	res, _ := encoding.GobEncode(ri)
	return res
}

func (ri *RegisterInfo) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(ri)
	if err != nil {return errors.Wrap(err, "RegisterInfo_Decode")}
	return nil
}

func (ri *RegisterInfo) formatOK() bool {
	return true
}

/////////////////////////////////////////////////////////////////////

// 对于TxRegresp，Payload需要存通过注册还是不通过

type RegisterResp struct {

}

// TODO
func (rr *RegisterResp) String() string {
	return ""
}

func (rr *RegisterResp) Encode() []byte {
	res, _ := encoding.GobEncode(rr)
	return res
}

func (rr *RegisterResp) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(rr)
	if err != nil {return errors.Wrap(err, "RegisterResp_Decode")}
	return nil
}

func (rr *RegisterResp) formatOK() bool {
	return true
}


////////////////////////////////////////////////////////////////////////////

