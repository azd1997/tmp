package crypto

import (
	"crypto/elliptic"

	"github.com/btcsuite/btcd/btcec"
)

// 使用btcec库包装一下，避免其他模块大量包引用导致后期修改不便
// 暂时放弃使用自制的加密库


const (
	PUBKEY_LEN_UNCOMPRESSED = btcec.PubKeyBytesLenUncompressed
	PUBKEY_LEN_COMPRESSED = btcec.PubKeyBytesLenCompressed
)


var S256 *KoblitzCurve = btcec.S256()


type PublicKey = btcec.PublicKey
type PrivateKey = btcec.PrivateKey
type Signature = btcec.Signature
type KoblitzCurve = btcec.KoblitzCurve


//////////////////////////////////////////////////////////////////////////


// 新建PrivateKey
func NewPrivateKey(curve elliptic.Curve) (*PrivateKey, error) {
	return btcec.NewPrivateKey(curve)
}

// 新建PrivateKey
func NewPrivateKeyS256() (*PrivateKey, error) {
	return btcec.NewPrivateKey(S256)
}




//////////////////////////////////////////////////////////////////////////


// 公钥加密
func Encrypt(pubkey *PublicKey, raw []byte) ([]byte, error) {
	return btcec.Encrypt(pubkey, raw)
}

// 私钥解密
func Decrypt(privkey *PrivateKey, encrypted []byte) ([]byte, error) {
	return btcec.Decrypt(privkey, encrypted)
}

// 私钥签名
func Sign(privkey *PrivateKey, target []byte) (*Signature, error) {
	return privkey.Sign(target)
}

// 公钥验证签名
func VerifySign(sig *Signature, target []byte, pubkey *PublicKey) bool {
	return sig.Verify(target, pubkey)
}


//////////////////////////////////////////////////////////////////////////


func PrivKeyFromBytes(curve elliptic.Curve, privKeyBytes []byte) (*PrivateKey, *PublicKey) {
	return btcec.PrivKeyFromBytes(curve, privKeyBytes)
}

func PrivKeyFromBytesS256(privKeyBytes []byte) (*PrivateKey, *PublicKey) {
	return btcec.PrivKeyFromBytes(S256, privKeyBytes)
}

func ParsePubKey(pubkeyStr []byte, curve *KoblitzCurve) (*PublicKey, error) {
	return btcec.ParsePubKey(pubkeyStr, curve)
}

func ParsePubKeyS256(pubkeyStr []byte) (*PublicKey, error) {
	return btcec.ParsePubKey(pubkeyStr, S256)
}

func ParseSignature(sig []byte, curve elliptic.Curve) (*Signature, error) {
	return btcec.ParseSignature(sig, curve)
}

func ParseSignatureS256(sig []byte) (*Signature, error) {
	return btcec.ParseSignature(sig, S256)
}



//////////////////////////////////////////////////////////////////////////


// 生成共享的对称密钥(用于密钥交换阶段)
func GenerateSharedSecret(privKey *PrivateKey, pubKey *PublicKey) []byte {
	return btcec.GenerateSharedSecret(privKey, pubKey)
}
