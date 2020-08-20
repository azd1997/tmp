package handshake

import (
	"encoding/gob"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/pkg/errors"
	"io"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/params"
)

const (
	ACCEPT = uint8(1)
	REJECT = uint8(2)
)

type Response struct {
	Version     uint8
	Accept      uint8
	CodeVersion params.CodeVersion
	NodeRole    uint8		// 节点类型
	SessionKey  []byte		// 临时会话密钥
	Sig         []byte
}

func NewAcceptResponseV1(codeVersion params.CodeVersion, nodeRole uint8,
	sessionKey []byte) *Response {
	return &Response{
		Version:     V1,
		Accept:      ACCEPT,
		CodeVersion: codeVersion,
		NodeRole:    nodeRole,
		SessionKey:  sessionKey,
	}
}

func NewRejectResponseV1() *Response {
	return &Response{
		Version: V1,
		Accept:  REJECT,
	}
}

func (r *Response) Decode(data io.Reader) error {
	err := gob.NewDecoder(data).Decode(r)
	if err != nil {
		return errors.Wrap(err, "Response_Decode")
	}
	return nil
}

func (r *Response) Encode() []byte {
	res, _ := encoding.GobEncode(r)
	return res
}

// Sign generate the signature for HandshakeResponse and set to the field Sig
func (r *Response) Sign(privKey *crypto.PrivateKey) {
	sig, _ := privKey.Sign(r.getSignContentHash())
	r.Sig = sig.Serialize()
}

// Verify 检查响应是否有效
func (r *Response) Verify(peerId crypto.ID) bool {
	// 还原签名
	sig, err := crypto.ParseSignatureS256(r.Sig)
	if err != nil {
		return false
	}
	// 还原对方公钥
	pubKey := crypto.ID2PublicKey(peerId)
	return sig.Verify(r.getSignContentHash(), pubKey)
}

func (r *Response) IsAccept() bool {
	return r.Accept == ACCEPT
}

func (r *Response) getSignContentHash() []byte {
	rcopy := *r
	rcopy.Sig = nil
	data := rcopy.Encode()
	hash := crypto.HashD(data)
	return hash
}
