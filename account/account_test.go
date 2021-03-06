package account

import (
	"fmt"
	"github.com/azd1997/ecoin/account/role"
	"testing"

	"github.com/azd1997/ego/ecrypto"
)

func TestNewAccount(t *testing.T) {
	acc, err := NewAccount(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%s\n", acc)
}

func TestAccount_UserID(t *testing.T) {
	acc, err := NewAccount(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%s\n", acc)

	fmt.Printf("%s\n", acc.UserId())

	if !role.IsRole(acc.RoleNo) {
		t.Error(err)
	}
	fmt.Println("account is valid")
}

func TestAccount_Sign(t *testing.T) {
	acc, err := NewAccount(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%s\n", acc)

	targetForSign := ecrypto.RandomHash()
	fmt.Printf("Hash: %s\n", targetForSign)

	sig, err := acc.Sign(targetForSign[:])
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("Sig: %s\n", sig)

	yes := acc.VerifySign(targetForSign[:], sig, acc.PrivateKey.PubKey())
	if !yes {
		t.Error("verify sig failed")
	}
	fmt.Println("verify sig success")
}

func TestAccountSaveAndLoadWithGob(t *testing.T) {
	acc, err := NewAccount(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%s\n", acc)

	err = acc.SaveFileWithGobEncode("./test-account.gob")
	if err != nil {t.Error(err)}

	acc1 := &Account{}
	err = acc1.LoadFileWithGobDecode("./test-account.gob")
	if err != nil {t.Error(err)}

	// 比较acc和acc1
	fmt.Printf("%s\n", acc1)
}

func TestAccountSaveAndLoadWithJson(t *testing.T) {
	acc, err := NewAccount(1)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("%s\n", acc)

	err = acc.SaveFileWithJsonEncode("./test-account.json")
	if err != nil {t.Error(err)}

	acc1 := &Account{}
	err = acc1.LoadFileWithJsonDecode("./test-account.json")
	if err != nil {t.Error(err)}

	// 比较acc和acc1
	fmt.Printf("%s\n", acc1)
}

//func TestAccount_NewTX_Coinbase(t *testing.T) {
//	acc, err := NewAccount(1)
//	if err != nil {
//		t.Error(err)
//	}
//	fmt.Printf("%s\n", acc)
//
//	userid, err := acc.UserId()
//	if err != nil {
//		t.Error(err)
//	}
//	fmt.Printf("%s\n", userid.String())
//
//	// 这里直接构造交易，由于TXcoinbase参数检查时虽然传入gsm但未使用，所以并不会报错。
//	tx, err := acc.NewTX(TX_COINBASE, &TxCoinbaseArgs{
//		To:          userid,
//		Amount:      60,
//		Description: "",
//	})
//	if err != nil {
//		t.Error(err)
//	}
//	fmt.Printf("tx: %s\n", tx)
//}

//func TestAccounts_AddAccount(t *testing.T) {
//	sa := Accounts{Map: map[string]*Account{}}
//	fmt.Printf("%s\n", sa)
//
//	userid, err := sa.AddAccount(1)
//	if err != nil {
//		t.Error(err)
//	}
//	fmt.Printf("userid: %s\n", userid.String())
//
//	fmt.Printf("%s\n", sa)
//}
//
//func TestAccounts_SaveFileWithGobEncode(t *testing.T) {
//	sa := Accounts{Map: map[string]*Account{}}
//	fmt.Printf("%v\n", sa)
//
//	userid, err := sa.AddAccount(1)
//	if err != nil {
//		t.Error(err)
//	}
//	fmt.Printf("userid: %s\n", userid.String())
//
//	fmt.Printf("%v\n", sa)
//
//	// 保存
//	err = sa.SaveFileWithGobEncode("./test.accounts")
//	if err != nil {
//		t.Error(err)
//	}
//	// 查看是否生成了相应文件。确实是生成了
//}
//
//func TestAccounts_LoadFileWithGobDecode(t *testing.T) {
//
//	sa := Accounts{Map: map[string]*Account{}}
//	fmt.Printf("%s\n", sa)
//
//	userid, err := sa.AddAccount(1)
//	if err != nil {
//		t.Error(err)
//	}
//	fmt.Printf("userid: %s\n", userid.String())
//
//	fmt.Printf("%s\n", sa)
//
//	// 保存
//	err = sa.SaveFileWithGobEncode("./test.accounts")
//	if err != nil {
//		t.Error(err)
//	}
//
//	sa1 := Accounts{map[string]*Account{}}
//	err = sa1.LoadFileWithGobDecode("./test.accounts")
//	if err != nil {
//		t.Error(err)
//	}
//
//	fmt.Printf("%v\n", sa)
//	// 看能否恢复出来
//
//}

