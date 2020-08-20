package p2p

import (
	"bytes"
	"fmt"
	"time"

	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/params"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/p2p/peer"
	"github.com/azd1997/ecoin/protocol/handshake"
)

// negotiator负责握手

/*
sender:
	1. generate random session used temporary key
	2. use self long-term key to sign message
	3. send
receiver:
	1. generate random session used temporary key
	3. use self long-term key to sign message
	5. reply
final:
	1. get shared secret P from two temporary key
	2. sha512(P), use first 32 bytes as secret key and rest 12 bytes as nonce
	3. use AES-GCM-256 to encrypt/decrypt following message
*/

const (
	handshakeProtocolID = 0
	nonceSize           = 12
)

type negotiator interface {
	handshakeTo(conn TCPConn, peer *peer.Peer) (codec, error)
	recvHandshake(conn TCPConn, accept bool) (*peer.Peer, codec, error)
}

type negotiatorImp struct {
	account                 *account.Account
	chainID                 uint8
	codeVersion             params.CodeVersion
	minimizeVersionRequired params.CodeVersion
	genSessionKeyFunc       func() (*crypto.PrivateKey, error) // for test stub
}

func newNegotiator(account *account.Account, chainID uint8) negotiator {
	result := &negotiatorImp{
		account:                 account,
		chainID:                 chainID,
		codeVersion:             params.CurrentCodeVersion,
		minimizeVersionRequired: params.MinimizeVersionRequired,
		genSessionKeyFunc:       genSessionKeyFunc,
	}
	return result
}

// handshakeTo 通过与peer建立的连接向对方发起握手，协商消息加密的对称秘钥
func (n *negotiatorImp) handshakeTo(conn TCPConn, peer *peer.Peer) (codec, error) {
	// 生成会话用的临时私钥。 该私钥将用来生成会话加密用的对称密钥以及其Nonce值
	sessionPrivKey, err := n.genSessionKeyFunc()
	if err != nil {
		return nil, err
	}

	// 发送握手请求
	requestBytes := n.genRequest(sessionPrivKey)
	conn.Send(requestBytes)

	// 等待对方的回应
	response, err := n.waitResponse(conn, sessionPrivKey)
	if err != nil {
		return nil, err
	}

	// 根据对方回应决定是否拒绝该回应
	if err := n.whetherRejectResp(response, peer.ID); err != nil {
		return nil, err
	}

	// 接受回应之后，根据回应解析出对方生成的临时会话私钥中的公钥
	peerSessionKey, err := crypto.ParsePubKeyS256(response.SessionKey)
	if err != nil {
		return nil, err
	}

	// 根据对方的临时公钥和自己的临时私钥生成会话加解密模块
	return newAESGCMCodec(peerSessionKey, sessionPrivKey)
}

// recvHandshake 接受握手消息
func (n *negotiatorImp) recvHandshake(conn TCPConn, accept bool) (*peer.Peer, codec, error) {
	// 等待请求
	request, err := n.waitRequest(conn)
	if err != nil {
		return nil, nil, err
	}

	// 验证请求的合法性
	if !request.Verify() {
		return nil, nil, ErrNegotiateInvalidSig
	}

	// 从请求中解析出对方的临时会话私钥中的公钥
	peerSessionKey, err := crypto.ParsePubKeyS256(request.SessionKey)
	if err != nil {
		return nil, nil, ErrNegotiateBrokenData{
			info: fmt.Sprintf("parse handshake session public key failed:%v", err),
		}
	}

	//  一开始就设定不接受握手请求
	if !accept {
		rejectRsp := n.genRejectResponse()
		conn.Send(rejectRsp)
		return nil, nil, nil
	}

	// 检查对方握手请求，确定是否拒绝，err不为空，则拒绝
	if err := n.whetherRejectReq(request); err != nil {
		return nil, nil, err
	}

	// 接受请求
	// 生成会话用的临时私钥。 该私钥将用来生成会话加密用的对称密钥以及其Nonce值
	sessionPrivKey, err := n.genSessionKeyFunc()
	if err != nil {
		return nil, nil, err
	}

	// 生成接受请求的回应
	acceptRsp := n.genAcceptResponse(sessionPrivKey)
	conn.Send(acceptRsp)

	// 根据对方的临时公钥和自己的临时私钥 生成加解密模块
	ec, err := newAESGCMCodec(peerSessionKey, sessionPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// 获取请求方peer
	peer1, err := n.getPeerFromRequest(conn, request)
	if err != nil {
		return nil, nil, err
	}

	return peer1, ec, nil
}

// 等待握手的回应消息
func (n *negotiatorImp) waitResponse(conn TCPConn, sessionPrivKey *crypto.PrivateKey) (*handshake.Response, error) {
	// 从连接读取数据包packet中的payload数据
	plainText, err := n.readPacket(conn)
	if err != nil {
		return nil, err
	}

	// 从payload中解码出握手响应（handshake.Response）消息
	resp := &handshake.Response{}
	err = resp.Decode(bytes.NewReader(plainText))
	if err != nil {
		return nil, ErrNegotiateBrokenData{
			info: fmt.Sprintf("decode handshake response failed: %v", err),
		}
	}
	return resp, nil
}

// 等待握手请求
func (n *negotiatorImp) waitRequest(conn TCPConn) (*handshake.Request, error) {
	// 从连接读取数据包packet中的payload数据
	plainText, err := n.readPacket(conn)
	if err != nil {
		return nil, err
	}

	// 从payload中解码出握手响应（handshake.Response）消息
	req := &handshake.Request{}
	err = req.Decode(bytes.NewReader(plainText))
	if err != nil {
		return nil, ErrNegotiateBrokenData{
			info: fmt.Sprintf("decode handshake request failed: %v", err),
		}
	}

	return req, nil
}

// 根据自己的临时会话私钥 生成握手请求
func (n *negotiatorImp) genRequest(sessionPrivKey *crypto.PrivateKey) []byte {

	// 从自己的临时会话私钥中得到临时会话公钥的压缩编码
	sessionPubKey := sessionPrivKey.PubKey()
	sessionPubKeyBytes := sessionPubKey.SerializeCompressed()

	// 构造握手请求并签名
	req := handshake.NewRequestV1(n.chainID, n.codeVersion, n.account.RoleNo,
		crypto.PrivateKey2ID(n.account.PrivateKey, n.account.RoleNo), sessionPubKeyBytes)
	req.Sign(n.account.PrivateKey)

	// 构造TCP packet
	return buildTCPPacket(req.Encode(), handshakeProtocolID)
}

// 生成拒绝握手请求的响应
func (n *negotiatorImp) genRejectResponse() []byte {
	resp := handshake.NewRejectResponseV1()
	resp.Sign(n.account.PrivateKey)

	return buildTCPPacket(resp.Encode(), handshakeProtocolID)
}

// 生成接受握手的响应
func (n *negotiatorImp) genAcceptResponse(sessionPrivKey *crypto.PrivateKey) []byte {
	resp := handshake.NewAcceptResponseV1(n.codeVersion, n.account.RoleNo,
		sessionPrivKey.PubKey().SerializeCompressed())
	resp.Sign(n.account.PrivateKey)

	return buildTCPPacket(resp.Encode(), handshakeProtocolID)
}

// 尝试从连接读取数据包；如果超过5s接收不到packet，则返回nil packet
func (n *negotiatorImp) readPacket(conn TCPConn) ([]byte, error) {
	timeoutTicker := time.NewTicker(5 * time.Second)
	recvC := conn.GetRecvChannel()
	var payload []byte
	var protocolID uint8
	var ok bool

	select {
	case <-timeoutTicker.C:
		return nil, ErrNegotiateTimeout
	case packet := <-recvC:
		if ok, payload, protocolID = verifyTCPPacket(packet); !ok {
			return nil, ErrNegotiateBrokenData{
				info: fmt.Sprintf("veirfy handshake packet checksum failed"),
			}
		}
	}

	if protocolID != handshakeProtocolID {
		return nil, ErrNegotiateBrokenData{
			info: fmt.Sprintf("invalid protocol ID for handshake %d", protocolID),
		}
	}

	return payload, nil
}

// 检查握手请求是否有效，据此决定是否拒绝
func (n *negotiatorImp) whetherRejectReq(request *handshake.Request) error {
	// TODO： 检查链ID
	if request.ChainID != n.chainID {
		return ErrNegotiateChainIDMismatch
	}

	// 检查软件版本
	if request.CodeVersion < n.minimizeVersionRequired {
		return ErrNegotiateCodeVersionMismatch{n.minimizeVersionRequired, request.CodeVersion}
	}

	// TODO: 检查请求的来源节点类型(节点/账户的角色)
	// 暂时假设只有同角色类型的结点才可以通信
	if request.NodeRole != n.account.RoleNo {
		return ErrNegotiateNodeRoleMismatch
	}

	return nil
}

// 是否拒绝握手回应
func (n *negotiatorImp) whetherRejectResp(response *handshake.Response, remoteID crypto.ID) error {
	// 验证响应的签名
	if !response.Verify(remoteID) {
		return ErrNegotiateInvalidSig
	}

	// 检查响应是否可以接收
	if !response.IsAccept() {
		return ErrNegotiateConnectionRefused
	}


	// 检查软件版本
	if int(response.CodeVersion) < int(n.minimizeVersionRequired) {
		return ErrNegotiateCodeVersionMismatch{n.minimizeVersionRequired, response.CodeVersion}
	}

	// TODO： 检查response来源的节点角色
	// 暂时假设只有同角色类型的结点才可以通信
	if response.NodeRole != n.account.RoleNo {
		return ErrNegotiateNodeRoleMismatch
	}

	return nil
}

// 从握手请求中获取对方结点信息
func (n *negotiatorImp) getPeerFromRequest(conn TCPConn, request *handshake.Request) (*peer.Peer, error) {
	addr := conn.RemoteAddr()
	ip, port := utils.ParseIPPort(addr.String())
	peerFromReq := peer.NewPeer(ip, port, request.From)
	return peerFromReq, nil
}

// 生成临时会话私钥
func genSessionKeyFunc() (*crypto.PrivateKey, error) {
	sessionPrivKey, err := crypto.NewPrivateKeyS256()
	if err != nil {
		return nil, err
	}

	return sessionPrivKey, nil
}
