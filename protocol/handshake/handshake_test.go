package handshake

import (
	"bytes"
	"testing"

	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/params"
	"github.com/azd1997/ecoin/common/utils"
)

func TestRequest(t *testing.T) {
	fromPrivateKey, _ := crypto.NewPrivateKeyS256()
	fromID := crypto.PrivateKey2ID(fromPrivateKey, role.HOSPITAL)
	sessionPrivKey, _ := crypto.NewPrivateKeyS256()
	chainID := uint8(1)
	codeVersion := params.CodeVersion(1)
	var nodeRole uint8 = role.HOSPITAL
	sessionKey := sessionPrivKey.PubKey().SerializeCompressed()

	request := NewRequestV1(chainID, codeVersion, nodeRole, fromID, sessionKey)
	request.Sign(fromPrivateKey)
	requestBytes := request.Encode()

	rRequest := &Request{}
	err := rRequest.Decode(bytes.NewReader(requestBytes))
	if err != nil {
		t.Fatalf("decode Request failed: %v\n", err)
	}

	if err := utils.TCheckUint8("version", V1, rRequest.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("chain ID", chainID, rRequest.ChainID); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint16("code version", uint16(codeVersion), uint16(rRequest.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node type", nodeRole, rRequest.NodeRole); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckString("from id", string(fromID), string(rRequest.From)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", sessionKey, rRequest.SessionKey); err != nil {
		t.Fatal(err)
	}
	if !rRequest.Verify() {
		t.Fatal("verify failed\n")
	}
}

func TestAcceptResponse(t *testing.T) {
	fromPrivateKey, _ := crypto.NewPrivateKeyS256()
	fromPublicKey := fromPrivateKey.PubKey()

	sessionPrivKey, _ := crypto.NewPrivateKeyS256()
	codeVersion := params.NodeVersionV1
	var nodeRole uint8 = role.PATIENT
	sessionKey := sessionPrivKey.PubKey().SerializeCompressed()

	acceptResponse := NewAcceptResponseV1(codeVersion, nodeRole, sessionKey)
	acceptResponse.Sign(fromPrivateKey)
	acceptResponseBytes := acceptResponse.Encode()

	rAcceptResponse := &Response{}
	err := rAcceptResponse.Decode(bytes.NewReader(acceptResponseBytes))
	if err != nil {
		t.Fatalf("decode Response failed: %v\n", err)
	}

	if err := utils.TCheckUint8("version", V1, rAcceptResponse.Version); err != nil {
		t.Fatal(err)
	}
	if !rAcceptResponse.IsAccept() {
		t.Fatal("expect accept\n")
	}
	if err := utils.TCheckUint16("code version", uint16(codeVersion), uint16(rAcceptResponse.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node type", nodeRole, rAcceptResponse.NodeRole); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", sessionKey, rAcceptResponse.SessionKey); err != nil {
		t.Fatal(err)
	}
	if !rAcceptResponse.Verify(crypto.PublicKey2ID(fromPublicKey, nodeRole)) {
		t.Fatal("verify failed\n")
	}
}

func TestRejectResponse(t *testing.T) {
	fromPrivateKey, _ := crypto.NewPrivateKeyS256()
	fromPublicKey := fromPrivateKey.PubKey()

	rejectResponse := NewRejectResponseV1()
	rejectResponse.Sign(fromPrivateKey)
	rejectResponseBytes := rejectResponse.Encode()
	var nodeRole uint8 = role.PATIENT

	rRejectResponse := &Response{}
	err := rRejectResponse.Decode(bytes.NewReader(rejectResponseBytes))
	if err != nil {
		t.Fatalf("decode Response failed: %v\n", err)
	}
	if rRejectResponse.IsAccept() {
		t.Fatal("expect reject")
	}
	if !rRejectResponse.Verify(crypto.PublicKey2ID(fromPublicKey, nodeRole)) {
		t.Fatal("verify failed\n")
	}
}
