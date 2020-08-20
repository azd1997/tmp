package p2p

import (
	"bytes"
	"net"
	"testing"

	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/common/params"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/p2p/peer"
	"github.com/azd1997/ecoin/protocol/handshake"
)

var negotiatorTestVar = &struct {
	sendPrivKey        *crypto.PrivateKey
	sendID crypto.ID
	sendSessionPrivKey *crypto.PrivateKey
	sendSessionPubKey  *crypto.PublicKey

	recvPrivKey        *crypto.PrivateKey
	recvID crypto.ID
	recvSessionPrivKey *crypto.PrivateKey
	recvSessionPubKey  *crypto.PublicKey

	expectCodec codec

	remoteIP   net.IP
	remotePort int

	chainID    uint8
	errChainID uint8
}{}

func init() {
	tv := negotiatorTestVar

	tv.sendPrivKey, _ = crypto.NewPrivateKeyS256()
	tv.sendID = crypto.PrivateKey2ID(tv.sendPrivKey, role.HOSPITAL)

	sendSessionPrivKeyHex := "BF40216703E409988F9E07EFFF851AE8AD53B2EC9193D1CE7CB28AB066274466"
	keyBytes, _ := encoding.FromHex(sendSessionPrivKeyHex)
	tv.sendSessionPrivKey, tv.sendSessionPubKey = crypto.PrivKeyFromBytesS256(keyBytes)

	tv.recvPrivKey, _ = crypto.NewPrivateKeyS256()
	tv.recvID = crypto.PrivateKey2ID(tv.recvPrivKey, role.HOSPITAL)

	recvSessionPrivKeyHex := "B9F7952D389470A15E996642DDCD099C9C557F8444D730BA79A3E56BCDF671CA"
	keyBytes, _ = encoding.FromHex(recvSessionPrivKeyHex)
	tv.recvSessionPrivKey, tv.recvSessionPubKey = crypto.PrivKeyFromBytesS256(keyBytes)

	tv.expectCodec, _ = newAESGCMCodec(tv.recvSessionPubKey, tv.sendSessionPrivKey)

	tv.remoteIP = net.ParseIP("192.168.1.2")
	tv.remotePort = 10000

	tv.chainID = 1
	tv.errChainID = 2
}

func TestHandshakeTo(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	receiver := newReceiver(role.HOSPITAL)
	conn := newTCPConnMock()

	// mock accept response
	resp := receiver.genAcceptResponse(tv.recvSessionPrivKey)
	conn.setRecvPkt(resp)

	// a full node handshake to another full node
	peer2 := peer.NewPeer(tv.remoteIP, tv.remotePort, tv.recvID)
	codec, err := sender.handshakeTo(conn, peer2)
	if err != nil {
		t.Fatalf("handshakeTo err:%v\n", err)
	}

	// check handshake request
	_, reqPkt, _ := verifyTCPPacket(conn.getSendPkt())
	req := &handshake.Request{}
	err = req.Decode(bytes.NewBuffer(reqPkt))
	if err != nil {
		t.Fatalf("decode request failed: %v\n", err)
	}
	checkRequest(t, req)

	// check codec
	checkCodec(t, codec, tv.expectCodec)
}

func TestRecvHandshake(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	receiver := newReceiver(role.HOSPITAL)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey)
	conn.setRecvPkt(req)

	// a full node wait another full node handshake
	peer2, codec, err := receiver.recvHandshake(conn, true)
	if err != nil {
		t.Fatalf("recvHandshake err:%v\n", err)
	}

	// check handshake response
	_, respPkt, _ := verifyTCPPacket(conn.getSendPkt())
	resp := &handshake.Response{}
	err = resp.Decode(bytes.NewReader(respPkt))
	if err != nil {
		t.Fatalf("decode response failed: %v\n", err)
	}
	checkResponse(t, resp)

	// check peer2
	checkPeer(t, peer2)

	// check codec
	checkCodec(t, codec, tv.expectCodec)
}

func TestReject(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	receiver := newReceiver(role.PATIENT)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey)
	conn.setRecvPkt(req)

	// reject request
	_, _, err := receiver.recvHandshake(conn, false)
	if err != nil {
		t.Fatalf("decrypt response failed:%v\n", err)
	}

	// check handshake response
	_, respPkt, _ := verifyTCPPacket(conn.getSendPkt())
	resp := &handshake.Response{}
	err = resp.Decode(bytes.NewReader(respPkt))
	if err != nil {
		t.Fatalf("decode response failed: %v\n", err)
	}

	err = sender.whetherRejectResp(resp, tv.recvID)
	if err != ErrNegotiateConnectionRefused {
		t.Fatal("expect connection refused error")
	}
}

func TestChainIDMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	sender.chainID = tv.errChainID
	receiver := newReceiver(role.PATIENT)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey)
	conn.setRecvPkt(req)

	_, _, err := receiver.recvHandshake(conn, true)
	if err != ErrNegotiateChainIDMismatch {
		t.Fatalf("expect chain ID mismatch error, %v\n", err)
	}
}

func TestSenderNodeRoleMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	receiver := newReceiver(role.PATIENT)
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey)
	conn.setRecvPkt(req)

	_, _, err := receiver.recvHandshake(conn, true)
	if err != ErrNegotiateNodeRoleMismatch {
		t.Fatalf("expect node role mismatch error, %v\n", err)
	}
}

func TestReceiverNodeTypeMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	receiver := newReceiver(role.PATIENT)
	conn := newTCPConnMock()

	// mock accept response
	resp := receiver.genAcceptResponse(tv.recvSessionPrivKey)
	conn.setRecvPkt(resp)

	peer2 := peer.NewPeer(tv.remoteIP, tv.remotePort, tv.recvID)
	_, err := sender.handshakeTo(conn, peer2)
	if err != ErrNegotiateNodeRoleMismatch {
		t.Fatalf("expect node role mismatch error, %v\n", err)
	}
}

func TestSenderCodeVersionMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	receiver := newReceiver(role.PATIENT)
	receiver.minimizeVersionRequired++
	conn := newTCPConnMock()

	// mock request
	req := sender.genRequest(tv.sendSessionPrivKey)
	conn.setRecvPkt(req)

	_, _, err := receiver.recvHandshake(conn, true)
	if _, ok := err.(ErrNegotiateCodeVersionMismatch); !ok {
		t.Fatalf("expect code version mismatch error, %v\n", err)
	}
}

func TestReceiverCoderVersionMismatch(t *testing.T) {
	tv := negotiatorTestVar
	sender := newSender(role.HOSPITAL)
	sender.minimizeVersionRequired++
	receiver := newReceiver(role.PATIENT)
	conn := newTCPConnMock()

	// mock accept response
	resp := receiver.genAcceptResponse(tv.recvSessionPrivKey)
	conn.setRecvPkt(resp)

	peer2 := peer.NewPeer(tv.remoteIP, tv.remotePort, tv.recvID)
	_, err := sender.handshakeTo(conn, peer2)
	if _, ok := err.(ErrNegotiateCodeVersionMismatch); !ok {
		t.Fatalf("expect code version mismatch error, %v\n", err)
	}
}

func newSender(rol role.No) *negotiatorImp {
	tv := negotiatorTestVar
	ng := newNegotiator(&account.Account{RoleNo:rol, PrivateKey:tv.sendPrivKey}, tv.chainID)
	result := ng.(*negotiatorImp)
	result.genSessionKeyFunc = senderGenSessionKeyFunc
	return result
}

func newReceiver(rol role.No) *negotiatorImp {
	tv := negotiatorTestVar
	ng := newNegotiator(&account.Account{RoleNo:rol, PrivateKey:tv.recvPrivKey}, tv.chainID)
	result := ng.(*negotiatorImp)
	result.genSessionKeyFunc = receiverGenSessionKeyFunc
	return result
}

func checkRequest(t *testing.T, req *handshake.Request) {
	tv := negotiatorTestVar

	if !req.Verify() {
		t.Fatal("verify request failed")
	}
	if err := utils.TCheckUint8("version", handshake.V1, req.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("chain id", tv.chainID, req.ChainID); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint16("code version", uint16(params.CurrentCodeVersion),
		uint16(req.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node role", uint8(role.HOSPITAL), uint8(req.NodeRole)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("send ID", []byte(tv.sendID), []byte(req.From)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", tv.sendSessionPubKey.SerializeCompressed(), req.SessionKey); err != nil {
		t.Fatal(err)
	}
}

func checkResponse(t *testing.T, resp *handshake.Response) {
	tv := negotiatorTestVar

	if !resp.Verify(tv.recvID) {
		t.Fatal("verify response failed")
	}
	if err := utils.TCheckUint8("version", handshake.V1, resp.Version); err != nil {
		t.Fatal(err)
	}
	if !resp.IsAccept() {
		t.Fatal("expect accept")
	}
	if err := utils.TCheckUint16("code version", uint16(params.CurrentCodeVersion),
		uint16(resp.CodeVersion)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("node role", uint8(role.HOSPITAL), uint8(resp.NodeRole)); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("session key", tv.recvSessionPubKey.SerializeCompressed(), resp.SessionKey); err != nil {
		t.Fatal(err)
	}
}

func checkPeer(t *testing.T, p *peer.Peer) {
	tv := negotiatorTestVar

	if err := utils.TCheckIP("peer IP", tv.remoteIP, p.IP); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("peer port", tv.remotePort, p.Port); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("peer ID", []byte(tv.sendID), []byte(p.ID)); err != nil {
		t.Fatal(err)
	}
}

func checkCodec(t *testing.T, result codec, expect codec) {
	originText := []byte("negotiator test codec check")

	// result encrypt, expect decrypt
	cipherText, err := result.encrypt(originText)
	if err != nil {
		t.Fatalf("result codec encrypt failed:%v\n", err)
	}
	plainText, err := expect.decrypt(cipherText)
	if err != nil {
		t.Fatalf("expect codec decrypt failed:%v\n", err)
	}

	if err := utils.TCheckBytes("plain text", originText, plainText); err != nil {
		t.Fatal(err)
	}

	// expect encrypt, result decrypt
	cipherText, _ = expect.encrypt(originText)
	if plainText, err = result.decrypt(cipherText); err != nil {
		t.Fatalf("result codec decrypt failed:%v\n", err)
	}
	if err := utils.TCheckBytes("plain text", originText, plainText); err != nil {
		t.Fatal(err)
	}
}

///////////////////////////////////////genSessionKeyFuncStub

func senderGenSessionKeyFunc() (*crypto.PrivateKey, error) {
	tv := negotiatorTestVar
	return tv.sendSessionPrivKey, nil
}

func receiverGenSessionKeyFunc() (*crypto.PrivateKey, error) {
	tv := negotiatorTestVar
	return tv.recvSessionPrivKey, nil
}

///////////////////////////////////////tcpConnMock

type tcpConnMock struct {
	sendPkt []byte
	recvQ   chan []byte
}

func newTCPConnMock() *tcpConnMock {
	return &tcpConnMock{
		recvQ: make(chan []byte, 128),
	}
}

func (t *tcpConnMock) Send(data []byte) {
	t.sendPkt = data
}
func (t *tcpConnMock) GetRecvChannel() <-chan []byte {
	return t.recvQ
}
func (t *tcpConnMock) SetSplitFunc(func(received *bytes.Buffer) ([][]byte, error)) {}
func (t *tcpConnMock) SetDisconnectCb(func(addr net.Addr))                         {}
func (t *tcpConnMock) RemoteAddr() net.Addr {
	tv := negotiatorTestVar
	return &net.TCPAddr{
		IP:   tv.remoteIP,
		Port: tv.remotePort,
	}
}
func (t *tcpConnMock) Disconnect() {}
func (t *tcpConnMock) getSendPkt() []byte {
	return t.sendPkt
}
func (t *tcpConnMock) setRecvPkt(data []byte) {
	t.recvQ <- data
}

