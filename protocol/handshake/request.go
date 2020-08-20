package handshake

import (
	"encoding/gob"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/params"
)

type Request struct {
	Version     uint8
	ChainID     uint8
	CodeVersion params.CodeVersion
	NodeRole    uint8
	From crypto.ID
	SessionKey  []byte
	Sig         []byte
}

// pubkey为对方的公钥，sessionkey为临时会话公钥
func NewRequestV1(chainID uint8, codeVersion params.CodeVersion, nodeRole uint8,
	from crypto.ID, sessionKey []byte) *Request {
	return &Request{
		Version:     V1,
		ChainID:     chainID,
		CodeVersion: codeVersion,
		NodeRole:    nodeRole,
		From:        from,
		SessionKey:  sessionKey,
	}
}

func (r *Request) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(r)
	if err != nil {
		return errors.Wrap(err, "Request_Decode")
	}
	return nil
}

func (r *Request) Encode() []byte {
	res, _ := encoding.GobEncode(r)
	return res
}

// Sign generate the signature set to the field Sig
// 传入的私钥是签名者的私钥
func (r *Request) Sign(privKey *crypto.PrivateKey) {
	sig, _ := privKey.Sign(r.getSignContentHash())
	r.Sig = sig.Serialize()
}

// Verify checks the response is valid or not
// 传入的账户是接收者的账户，这个账户只是为了调用验证签名方法
// 接收者使用发送者request中的pubkey尝试验证签名
func (r *Request) Verify() bool {
	fromPublicKey := crypto.ID2PublicKey(r.From)
	if fromPublicKey == nil {
		return false
	}
	sig, err := crypto.ParseSignatureS256(r.Sig)
	if err != nil {
		return false
	}
	return sig.Verify(r.getSignContentHash(), fromPublicKey)
}

func (r *Request) getSignContentHash() []byte {
	rcopy := *r
	rcopy.Sig = nil
	data := rcopy.Encode()
	hash := crypto.HashD(data)
	return hash
}
