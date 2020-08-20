package account

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"github.com/azd1997/ego/utils"
	"github.com/pkg/errors"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/azd1997/ecoin/common/crypto"
)

// Account 账户，包含私钥和公钥，标志唯一身份。UserID是外部可见的标志
type Account struct {
	RoleNo  uint8              `json:"roleNo"`
	PrivateKey *crypto.PrivateKey `json:"privKey"`
}

// NewAccount 新建账户
// TODO: 注意：新建账户时需添加入本地gsm.accounts并向外广播.
func NewAccount(roleNo uint8) (*Account, error) {
	privateKey, err := crypto.NewPrivateKeyS256()
	if err != nil {
		return nil, errors.Wrap(err, "NewAccount")
	}
	return &Account{
		PrivateKey: privateKey,
		RoleNo:  roleNo,
	}, nil
}

// LoadOrCreateAccount 从指定路径加载账户，加载不到就新建
func LoadOrCreateAccount(accountFile string) (acc *Account, err error) {
	acc = &Account{}
	// 1.1. 加载selfAccount文件，取账户文件
	// 1.1.1 检查是否存在账户文件
	exists, err := utils.FileExists(accountFile)
	if err != nil {
		return nil, err
	}

	jsonSuffix := strings.HasSuffix(accountFile, ".json")

	// 若不存在，则需要创建一个账户并保存到这个文件
	if !exists {
		log.Printf("默认路径下找不到指定账户文件: %s", accountFile)
		log.Println("准备创建新账户......")
		acc, err = NewAccount(0)
		//fmt.Println("78910")
		if err != nil {
			return nil, err
		}

		if jsonSuffix {
			err = acc.SaveFileWithJsonEncode(accountFile)
		} else {
			err = acc.SaveFileWithGobEncode(accountFile)
		}

		//fmt.Println("123456")
		if err != nil {
			return nil, err
		}
		log.Printf("新账户创建成功并保存至默认路径， 账户ID: %s", acc.UserId())
	} else {
		// 若存在，则从这个文件读取account
		log.Println("指定路径下发现账户文件， 准备加载......")

		jsonSuffix := strings.HasSuffix(accountFile, ".json")
		if jsonSuffix {
			err = acc.LoadFileWithJsonDecode(accountFile)
		} else {
			err = acc.LoadFileWithGobDecode(accountFile)	// 只要不是json后缀，都以gob编码
		}

		if err != nil {
			return nil, err
		}
		log.Printf("账户加载成功， 账户ID: %s", acc.UserId())
	}

	return acc, nil
}

// TODO: 待解决的问题：多个账户文件在同一个目录下怎么去选取。目前的做法是只读取指定文件名的账户文件。但如果要考虑多个账户呢？


///////////////////////////////////////////////////////////////////////////////////


// String 打印字符串
func (a *Account) String() string {
	return utils.JsonMarshalIndentToString(a)
}

// DEPRECATED ID GENERATE WAY: UserId publicKeyHashRipemd160 + checksum + version -> base58 -> userID
// NOW: privateKey + roleNo -> publicKey + roleNo -> compressedPubKey(33B) + roleNo -> roleNo(1B)|base32edPub(32B)
func (a *Account) UserId() ID {
	return crypto.PrivateKey2ID(a.PrivateKey, a.RoleNo)
}

// Sign 使用该账号对目标数据作签名。目标数据只能是基础类型、结构体、切片、表等，必须提前转为[]byte
func (a *Account) Sign(target []byte) (sig *crypto.Signature, err error) {
	return a.PrivateKey.Sign(target)
}

// VerifySign 验证签名; 这个pubKey不一定是本账户的PubKey
func (a *Account) VerifySign(target []byte, sig *crypto.Signature, pubKey *crypto.PublicKey) bool {
	return sig.Verify(target, pubKey)
}

// NewTX 该账户作为主体，构造新交易
// NewTX交给其他地方做
//func (a *Account) NewTX(typ uint, args ArgsOfNewTX) (tx TX, err error) {
//	// TODO: 根据账户类型不同来处理
//	return newTransaction(typ, args)
//	// TODO： 这层只是简单调用，参数检查交给tx自己去做。
//}


///////////////////////////////////////////////////////////////////////////////////


// SaveFileWithGobEncode 保存到文件
func (a *Account) SaveFileWithGobEncode(file string) (err error) {
	// 用于编码
	acc := a.toaccount()
	if acc == nil {
		return errors.Wrap(errors.New("toaccount failed"), "Account_SaveFile")
	}
	// gob编码
	if err = utils.SaveFileWithGobEncode(file, acc); err != nil {
		return errors.Wrap(err, "Account_SaveFile")
	}
	return nil
}

// LoadFileWithGobDecode 从本地文件中读取自己账户表（用于加载）
func (a *Account) LoadFileWithGobDecode(file string) (err error) {
	if _, err = os.Stat(file); os.IsNotExist(err) {
		return errors.Wrap(err, "Account_LoadFile")
	}

	a1 := &account{}
	var a2 *Account

	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "Account_LoadFile")
	}

	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	if err = decoder.Decode(a1); err != nil {
		return errors.Wrap(err, "Account_LoadFile")
	}

	a2 = a1.toAccount()
	if a2 == nil {
		return errors.Wrap(errors.New("toAccount failed"), "Account_LoadFile")
	}
	a.PrivateKey = a2.PrivateKey
	a.RoleNo = a2.RoleNo

	return nil
}

// SaveFileWithJsonEncode 保存到文件
func (a *Account) SaveFileWithJsonEncode(file string) (err error) {
	// 用于编码
	acc := a.toaccount()
	if acc == nil {
		return errors.Wrap(errors.New("toaccount failed"), "Account_SaveFile")
	}
	// json编码
	if err = utils.SaveFileWithJsonMarshal(file, acc); err != nil {
		return errors.Wrap(err, "Account_SaveFile")
	}
	return nil
}

// LoadFileWithJsonDecode 从本地文件中读取自己账户表（用于加载）
func (a *Account) LoadFileWithJsonDecode(file string) (err error) {
	if _, err = os.Stat(file); os.IsNotExist(err) {
		return errors.Wrap(err, "Account_LoadFile")
	}

	a1 := &account{}
	var a2 *Account

	fileContent, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.Wrap(err, "Account_LoadFile")
	}

	err = json.Unmarshal(fileContent, a1)
	if err != nil {
		return errors.Wrap(err, "Account_LoadFile")
	}

	a2 = a1.toAccount()
	if a2 == nil {
		return errors.Wrap(errors.New("toAccount failed"), "Account_LoadFile")
	}
	a.PrivateKey = a2.PrivateKey
	a.RoleNo = a2.RoleNo

	return nil
}

///////////////////////////////////////////////////////////////////////////////////

// 角色相关



///////////////////////////////////////////////////////////////////////////////////

// 编码保存相关

type account struct {
	RoleNo uint8 `json:"roleNo"`
	PrivKeyB []byte  `json:"privKeyB"`
}

func (a *account) toAccount() *Account {
	priv, _ := crypto.PrivKeyFromBytes(crypto.S256, a.PrivKeyB)
	if priv == nil {
		return nil
	}
	return &Account{
		RoleNo:  a.RoleNo,
		PrivateKey: priv,
	}
}

func (a *Account) toaccount() *account {
	return &account{
		RoleNo:   a.RoleNo,
		PrivKeyB: a.PrivateKey.Serialize(),
	}
}