package crypto

import (
	"encoding/base32"
	"fmt"
	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/encoding"
	"math"
	"math/rand"
)

// NOTICE! ID串首部有一字节表示角色编号（或者说结点类型），后边则为公钥压缩编码再base32编码的字符串(53B)
// 并且角色编号直接就是uint8或者说byte，也就是明文编码
// 约定结构体类型的公钥私钥用PublicKey/PrivateKey表示； 压缩编码的字节序列的公约私钥则用PubKey/PrivKey表示
// 为避免可能产生的循环引用，这里不使用role.No，而是使用其原型uint8
//
// 使用ID时一定要记住ID长度为54B，第一个字节表示角色编号
//// 要注意id尽管是54B长度的字符串，但是其打印UTF-8编码下的字符串字符数不一定是54个，是不定的。 此处参考go语言rune相关
// 为了保证ID打印的效果一致，这里为ID实现String方法
//

const ID_LEN_WITH_ROLE = 54

var (
	base32Codec = base32.StdEncoding.WithPadding(base32.NoPadding)
)

var ZeroID ID = ID(make([]byte, ID_LEN_WITH_ROLE))		// 这也是基于不会出现全0 ID的假设。 TODO：如何解决这个问题：其实是可以解决的，在编解码时加入长度字段。暂时不这么做

type ID string

func (id *ID) String() string {
	return fmt.Sprintf("<%d>|[%X]", (*id)[0], (*id)[1:])
}

func (id *ID) ToHex() string {
	return encoding.ToHex([]byte(*id))
}

func (id *ID) RoleNo() uint8 {
	return (*id)[0]
}

func (id *ID) IsZeroID() bool {
	return *id == ZeroID
}

func (id *ID) IsValid() bool {
	idB := []byte(*id)
	if len(idB) != ID_LEN_WITH_ROLE {
		return false
	}
	if !role.IsRole(id.RoleNo()) {
		return false
	}
	return true
}

func PrivateKey2ID(privateKey *PrivateKey, roleNo uint8) ID {
	return PublicKey2ID(privateKey.PubKey(), roleNo)
}

func PublicKey2ID(publicKey *PublicKey, roleNo uint8) ID {
	pubCompressed := publicKey.SerializeCompressed()
	// fmt.Println(pubCompressed)
	return PubKey2ID(pubCompressed, roleNo)
}

// ID2PubKey
func ID2PublicKey(id ID) *PublicKey {
	pubKeyB := ID2PubKey(id)
	if pubKeyB == nil {
		return nil
	}

	var publicKey *PublicKey
	publicKey, err := ParsePubKeyS256(pubKeyB)
	if err != nil {
		return nil
	}

	return publicKey
}

// ID2PubKey 返回序列化的压缩的公钥字节切片
func ID2PubKey(id ID) []byte {
	pubKeyB, _ := base32Codec.DecodeString(string(id[1:]))
	return pubKeyB
}

// PubKey2ID 将序列化的压缩过的公钥字节切片转为ID
func PubKey2ID(compressedKey []byte, roleNo uint8) ID {
	// fmt.Println(roleNo, compressedKey)
	base32Key := base32Codec.EncodeToString(compressedKey)
	// fmt.Println(base32Key)

	idB := append([]byte{roleNo}, []byte(base32Key)...)
	id := ID(idB)
	// fmt.Println(id.String(), len(id))
	return id
}

// RandID 生成一个随机ID
func RandID() ID {
	key := make([]byte, 53)
	for i:=0; i<len(key); i++ {
		key[i] = uint8(rand.Intn(math.MaxUint8))
	}
	rolenos := [4]uint8{1,2,10,11}
	roleno := rolenos[rand.Intn(4)]

	slice := append([]byte{roleno}, key...)

	return ID(slice)
}
