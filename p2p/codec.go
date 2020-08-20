package p2p

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"

	"github.com/azd1997/ecoin/common/crypto"
)

// codec 用来加解密会话消息
type codec interface {
	encrypt(plainText []byte) ([]byte, error)
	decrypt(cipherText []byte) ([]byte, error)
}

// 实现 codec 接口
type aesgcmCodec struct {
	aead  cipher.AEAD	// AEAD是一种加密模式
	nonce []byte
}

func (aes *aesgcmCodec) encrypt(plainText []byte) ([]byte, error) {
	cipherText := aes.aead.Seal(nil, aes.nonce, plainText, nil)
	return cipherText, nil
}

func (aes *aesgcmCodec) decrypt(cipherText []byte) ([]byte, error) {
	plainText, err := aes.aead.Open(nil, aes.nonce, cipherText, nil)
	return plainText, err
}

// remotePubKey 对方的临时会话公钥； randPrivKey 自己生成的随机私钥
func newAESGCMCodec(remotePubKey *crypto.PublicKey, randPrivKey *crypto.PrivateKey) (*aesgcmCodec, error) {

	sharedKey := sha512.Sum512(crypto.GenerateSharedSecret(randPrivKey, remotePubKey))

	block, err := aes.NewCipher(sharedKey[:32])		// 取前32位作对称密钥
	if err != nil {
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &aesgcmCodec{
		aead:  aesgcm,
		nonce: sharedKey[32 : 32+12],	// 取后12位作随机数Nonce
	}, nil
}

// 对两个字节数组作异或
func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}
