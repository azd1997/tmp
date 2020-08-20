/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/2 15:56
* @Description: The file is for
***********************************************************************/

package enode

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/azd1997/ecoin/account/role"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/enode/bc"
	"github.com/azd1997/ecoin/p2p"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ego/epattern"
)

const (
	coreProtocolID           = 100
	coreProtocol             = "CoreProtocol"
	maxBlocksNumInResponse   = 16
	initializingSyncInterval = 1 * time.Second
	syncInterval             = 5 * time.Second
)

// waitingBlocks 是指同步区块时向其他节点请求的这些区块。接下来就需要等待这些区块
type waitingBlocks struct {
	peerID           crypto.ID
	lastResponseTime time.Time
	remainNums       uint32
	response         []*core.BlockRespMsg
}

// net 运行核心协议，与网络中其他节点达成共识
type net struct {
	InitFinishC chan bool
	// inited net模块是否已初始化好
	inited bool

	// workerNode 工人节点 账户角色为A类账户，则有义务承担worker的职责，负责出块
	workerNode bool

	// protocolRunner 协议运行器。 net实现了core协议，但它得添加到p2p.Node中，生成一个protocolRunner
	// 这其实就相当于http WEB编程中的handlerMux。
	protocolRunner p2p.ProtocolRunner

	// sendQ 消息发送通道
	sendQ chan *p2p.PeerData

	// chain 本地区块链
	chain *bc.Chain

	// txPool 交易池
	txPool *txPool

	// syncTicker 定时通知节点向网络中其他节点同步数据
	syncTicker *time.Ticker
	// syncHashResp 同步响应表，当向其他节点发起同步请求后，需要等待响应，并填充该表
	syncHashResp map[crypto.ID]*core.SyncRespMsg
	// waitingHash 当前时间，我在等待别人发过来的区块哈希
	waitingHash bool
	// waitingBlocks 当前时间，我等待别人发过来的 区块列表
	waitingBlocks []*waitingBlocks

	// txsToBroadcast 待广播的交易列表
	txsToBroadcast chan []*core.Tx
	// broadcastFilter 广播过滤器
	broadcastFilter map[string]time.Time

	// proofPool 证明池
	proofPool *proofPool

	// potProof POT证明。本机的PotProof产生的通道
	potProof chan *core.PoTProof
	potProofCollect chan *core.PoTProof	// 网络模块收集到的proof从这个chan传给pot竞争器

	// potWinnerBlock 本机节点作为出块节点
	potWinnerBlock chan *core.Block

	// lm 循环模式
	lm *epattern.LoopMode
}

func newNet(node p2p.Node, chain *bc.Chain, pool *txPool, nodeRole uint8) *net {
	result := &net{
		InitFinishC:     make(chan bool, 1),
		inited:          false,
		workerNode:      role.IsARole(nodeRole), // A类节点才具有出块权利和义务
		sendQ:           make(chan *p2p.PeerData, 512),
		chain:           chain,
		txPool:          pool,
		syncTicker:      time.NewTicker(initializingSyncInterval),
		syncHashResp:    make(map[crypto.ID]*core.SyncRespMsg),
		waitingHash:     false,
		txsToBroadcast:  make(chan []*core.Tx, txsCacheSize),
		broadcastFilter: make(map[string]time.Time),

		potProof:make(chan *core.PoTProof, 1),
		potProofCollect:make(chan *core.PoTProof, 128),
		potWinnerBlock:make(chan *core.Block),

		lm:              epattern.NewLoop(2),
	}

	result.protocolRunner = node.AddProtocol(result)
	return result
}

////////////////////////////////////////////////

// net 实现 p2p.Protocol接口

func (n *net) ID() uint8 {
	return coreProtocolID
}

func (n *net) Name() string {
	return coreProtocol
}

/////////////////////////////////////////////////

func (n *net) start() {
	go n.loop()   // 启动主工作循环
	go n.doSend() // 启动发送循环
	n.lm.StartWorking()
}

func (n *net) stop() {
	n.lm.Stop()
}

func (n *net) loop() {
	n.lm.Add()
	defer n.lm.Done()

	cleanupTicker := time.NewTicker(30 * time.Second)
	recvPktChan := n.protocolRunner.GetRecvChan()

	for {
		select {
		case <-n.lm.D:
			return
		case pkt := <-recvPktChan: // 处理接收到的P2P数据包
			n.handleRecvPacket(pkt)
		case <-n.syncTicker.C: // 发起同步请求
			n.sync()
		case pot := <-n.potProof:
			n.broadcastProof(pot)
		case newBlock := <-n.potWinnerBlock:
			n.broadcastBlock(newBlock)
		case txs := <-n.txsToBroadcast: // 广播交易
			n.broadcastTx(txs)
		case <-cleanupTicker.C: // 清理那些被过滤的节点，时间到了该解封 TODO
			now := time.Now()

			for k, v := range n.broadcastFilter {
				if now.Sub(v) > 1*time.Hour {
					delete(n.broadcastFilter, k)
				}
			}

		}
	}
}

// 向对端节点发送数据
func (n *net) send(data []byte, peerID crypto.ID) {
	n.sendQ <- &p2p.PeerData{
		Data: data,
		Peer: peerID,
	}
}

// 广播数据
func (n *net) broadcast(data []byte) {
	// 当前待发送数据取哈希，作为数据标识。并加入广播过滤表
	h := crypto.HashD(data)
	encoded := base64.StdEncoding.EncodeToString(h)
	n.broadcastFilter[encoded] = time.Now()

	// 发送队列未满则发送；满则广播失败（并且一段时间内不能广播）
	select {
	case n.sendQ <- &p2p.PeerData{Data: data}:
	default:
		logger.Warn("net send queue full, drop packet")
	}
}

// 发送循环
// 不断从发送队列接收数据，并将之通过protocolRunner发送出去
func (n *net) doSend() {
	n.lm.Add()
	defer n.lm.Done()

	for {
		select {
		case <-n.lm.D:
			return
		case sendData := <-n.sendQ:
			if err := n.protocolRunner.Send(sendData); err != nil {
				logger.Warn("send failed: %v\n", err)
			}
		}
	}
}

///////////////////////////////////////////////////////////////////

// 主动操作

// sync是同步的主动作
// 其职责是向邻居节点（也就是peer.table中记录的节点）请求（同步）区块数据
// sync第一步是请求区块哈希，第二步是根据区块哈希范围请求具体的区块数据
// 理论上需要两个同步间隔syncInterval
func (n *net) sync() {
	// 如果没有区块要等待，那么syncRequest请求区块哈希或者请求区块
	if len(n.waitingBlocks) == 0 {
		n.syncRequest()
		return
	}

	// 否则清理掉超时的等待区块，更新等待列表
	now := time.Now()
	var waiting []*waitingBlocks
	for _, exp := range n.waitingBlocks {
		// expect transferring a block per 5 seconds
		// TODO
		if now.Sub(exp.lastResponseTime) <= time.Duration(5*maxBlocksNumInResponse)*time.Second {
			waiting = append(waiting, exp)
			continue
		}

		logger.Info("peer %s response blockInfos timeout(now %s, last active %s), remain %d\n",
			exp.peerID, utils.TimeToString(now),
			utils.TimeToString(exp.lastResponseTime), exp.remainNums)

	}
	n.waitingBlocks = waiting
}

// syncRequest 发送SyncReqMsg请求对方节点的区块哈希范围（如果对方比自己高的话）
func (n *net) syncRequest() {
	// 如果syncHashResp表是空的，那么需要请求区块哈希。 （这里的hash指的是区块哈希）
	if len(n.syncHashResp) == 0 {
		latestHash := n.chain.GetSyncBlockHash()
		for _, h := range latestHash {
			request := core.NewSyncReqMsg(h).Encode()
			n.broadcast(request)
		}
		n.waitingHash = true // 当前正在等待区块哈希
		return
	}

	// 否则请求具体的区块数据。
	// 查看n.syncHashResp，向所有比自己新(高)的节点请求区块，并且过滤掉那些相同的响应
	queryFilter := make(map[string]bool)
	alreadyUptodate := true
	for peerID, resp := range n.syncHashResp {
		// 自己是否比对方更高？是则跳过
		if resp.IsUptodate() {
			continue
		}
		alreadyUptodate = false

		// 过滤掉相同的响应
		queryFlag := fmt.Sprintf("%X-%d", resp.End, resp.HeightDiff)
		if _, find := queryFilter[queryFlag]; find {
			continue
		}
		queryFilter[queryFlag] = true

		// 构造区块请求消息
		request := core.NewBlockReqMsg(resp.Base, resp.End, n.workerNode).Encode()
		n.send(request, peerID)

		// 添加到等待列表
		exp := &waitingBlocks{
			peerID:           peerID,
			lastResponseTime: time.Now(), // 以当前时间就行
			remainNums:       resp.HeightDiff,
		}
		n.waitingBlocks = append(n.waitingBlocks, exp)
		logger.Debug("add block response expection, peer:%s remainNums:%d, from %X to %X\n",
			exp.peerID, exp.remainNums, resp.Base, resp.End)
	}

	// 清理掉等待
	n.syncHashResp = make(map[crypto.ID]*core.SyncRespMsg)
	n.waitingHash = false

	// 结束初始化流程（当前已经是最新状态）
	if !n.inited && alreadyUptodate {
		n.InitFinishC <- true // 通知外部 net模块初始化结束
		n.inited = true       // 内部标识初始化结束
		// 初始化结束之后就要减少同步间隔，以保证自己时刻跟进网络最新状态
		n.syncTicker.Stop()
		n.syncTicker = time.NewTicker(syncInterval)
		logger.Debug("network for CoreProtocol init finished")
	}
}

// 广播区块
func (n *net) broadcastBlock(b *core.Block) {
	content := core.NewBlockBroadcastMsg(b).Encode()
	n.broadcast(content)
}

// 广播交易
func (n *net) broadcastTx(txs []*core.Tx) {
	content := core.NewTxBroadcastMsg(txs).Encode()
	n.broadcast(content)
}

// 广播证明
func (n *net) broadcastProof(proof *core.PoTProof) {
	content := core.NewProofBroadcastMsg(proof).Encode()
	n.broadcast(content)
}

//////////////////////////////////////// handler /////////////////////////////////////////////

// 接收到P2P数据包时的handler方法
func (n *net) handleRecvPacket(pd *p2p.PeerData) {
	var err error
	var msg *core.Head = &core.Head{}

	// 解码出消息头
	if err = msg.Decode(bytes.NewReader(pd.Data)); err != nil {
		return
	}

	errorLog := func() {
		logger.Warn("receive err type(%d) msg from %s\n", msg.Type, pd.Peer)
		return
	}

	// net模块还没初始化完成，则忽略区块/交易/证明广播消息
	if !n.inited && (msg.Type == core.MsgBlockBroadcast || msg.Type == core.MsgTxBroadcast || msg.Type == core.MsgProofBroadcast) {
		return
	}

	// 读出消息的内容，根据消息头包含的消息类型来处理
	data := bytes.NewReader(pd.Data)
	switch msg.Type {
	case core.MsgSyncReq:
		syncRequest := &core.SyncReqMsg{}
		if err = syncRequest.Decode(data); err != nil {
			errorLog()
			return
		}
		n.handleSyncReqMsg(syncRequest, pd.Peer)

	case core.MsgSyncResp:
		syncHashResp := &core.SyncRespMsg{}
		if err = syncHashResp.Decode(data); err != nil {
			errorLog()
			return
		}
		n.handleSyncRespMsg(syncHashResp, pd.Peer)

	case core.MsgBlockReq:
		blockRequest := &core.BlockReqMsg{}
		if err = blockRequest.Decode(data); err != nil {
			errorLog()
			return
		}
		n.handleBlocksReqMsg(blockRequest, pd.Peer)

	case core.MsgBlockResp:
		blockResponse := &core.BlockRespMsg{}
		if err = blockResponse.Decode(data); err != nil {
			errorLog()
			return
		}
		n.handleBlocksRespMsg(blockResponse, pd.Peer)

	case core.MsgBlockBroadcast:
		block := &core.BlockBroadcastMsg{}
		if err = block.Decode(data); err != nil {
			errorLog()
			return
		}
		n.handleBlockBroadcastMsg(pd.Data, block, pd.Peer)

	case core.MsgTxBroadcast:
		txs := &core.TxBroadcastMsg{}
		if err = txs.Decode(data); err != nil {
			errorLog()
			return
		}
		n.handleTxBroadcastMsg(pd.Data, txs, pd.Peer)
	case core.MsgProofBroadcast:
		proofMsg := &core.ProofBroadcastMsg{}
		if err = proofMsg.Decode(data); err != nil {
			errorLog()
			return
		}
		n.handleProofBroadcastMsg(pd.Data, proofMsg, pd.Peer)

	default:
		errorLog()
	}
}

func (n *net) handleSyncReqMsg(r *core.SyncReqMsg, peerID crypto.ID) {
	logger.Debug("receive SyncRequest from %s, %v\n", peerID, r)

	// 对方最高区块哈希作为base，到自己的本地区块链查询，返回syncEnd和heightDiff。
	// 注意如果base过旧，那么自己只会返回syncMaxBlocks的差距（只是为了避免请求压力全部打在一个节点上）
	syncEnd, heightDiff, err := n.chain.GetSyncHash(r.Base)

	var response []byte

	if err != nil {
		if _, ok := err.(bc.ErrAlreadyUpToDate); ok {
			response = core.NewSyncRespMsg(nil, nil, 0).Encode()
			logger.Debug("reply sync request already uptodate\n")
		} else {
			logger.Debug("%v\n", err)
			return
		}
	} else {
		response = core.NewSyncRespMsg(r.Base, syncEnd, heightDiff).Encode()
		logger.Debug("replay %d sync request with block hash\n", heightDiff)
	}

	if response == nil {
		logger.Warn("generate SyncRequest response failed\n")
		return
	}

	n.send(response, peerID)
}

func (n *net) handleSyncRespMsg(r *core.SyncRespMsg, peerID crypto.ID) {
	// 如果自己不在等待区块哈希，那么不就处理该消息
	if !n.waitingHash {
		return
	}

	logger.Debug("receive SyncResponse from %s, %v\n", peerID, r)
	// 记录该消息，等待处理
	n.syncHashResp[peerID] = r
}

func (n *net) handleBlocksReqMsg(r *core.BlockReqMsg, peerID crypto.ID) {
	logger.Debug("receive BlockRequest from %s, base %X\n", peerID, r.Base)

	// 从本地区块链获取这些区块
	blocks, err := n.chain.GetSyncBlocks(r.Base, r.End, r.IsOnlyHeader())
	if err != nil {
		logger.Warn("%v\n", err)
		return
	}

	logger.Debug("reply BlockRequest with %d blockInfos\n", len(blocks))
	// 如果超过了单次响应最大携带区块数16，那么就缩减为16
	for len(blocks) > 0 {
		sendNum := maxBlocksNumInResponse
		if len(blocks) < maxBlocksNumInResponse {
			sendNum = len(blocks)
		}

		response := core.NewBlockRespMsg(blocks[:sendNum]).Encode()
		if response == nil {
			logger.Warn("generate BlockResponse failed\n")
			return
		}
		n.send(response, peerID)

		blocks = blocks[sendNum:]
	}
}

func (n *net) handleBlocksRespMsg(r *core.BlockRespMsg, peerID crypto.ID) {
	logger.Debug("receive BlockResponse from %s, %d blockInfos\n", peerID, len(r.Blocks))
	for i, exp := range n.waitingBlocks {
		// 从等待列表中找到对端节点的这一个kv对，刷新对端节点的上次响应时间
		if exp.peerID == peerID {
			exp.lastResponseTime = time.Now()
			exp.remainNums -= uint32(len(r.Blocks))
			exp.response = append(exp.response, r)

			// 如果从exp.peerID那没有区块要等了，那么将之移除
			// 并且将所有响应的区块添加到本地区块链
			remove := false
			if exp.remainNums == 0 {
				logger.Info("finish blockInfos sync from %s\n", peerID)

				var toAddBlocks []*core.Block
				for _, resp := range exp.response {
					toAddBlocks = append(toAddBlocks, resp.Blocks...)
				}
				n.chain.AddBlocks(toAddBlocks, false)
				remove = true
			}
			if exp.remainNums < 0 {
				logger.Warn("receive err block response from %s\n", peerID)
				remove = true
			}

			if remove {
				n.waitingBlocks = append(n.waitingBlocks[:i], n.waitingBlocks[i+1:]...)
			}

			return
		}
	}

}

func (n *net) handleBlockBroadcastMsg(originData []byte, b *core.BlockBroadcastMsg, peerID crypto.ID) {
	if n.relayBroadcast(originData) { // 该广播数据第一次收到
		hash := b.Block.Hash
		logger.Debug("first time receive block broadcast from %s, hash %X\n", peerID, hash)
		n.chain.AddBlocks([]*core.Block{b.Block}, false)
	}
}

func (n *net) handleTxBroadcastMsg(originData []byte, b *core.TxBroadcastMsg, peerID crypto.ID) {
	if n.relayBroadcast(originData) {
		logger.Debug("first time receive evidence broadcast from %s, %v\n", peerID, b)
		n.txPool.addTx(b.Txs, true)
	}
}

func (n *net) handleProofBroadcastMsg(originData []byte, b *core.ProofBroadcastMsg, peerID crypto.ID) {
	if n.relayBroadcast(originData) {
		logger.Debug("first time receive tx broadcast from %s, %v\n", peerID, b)
		//n.txPool.addTx(b.Txs, true)
		n.potProofCollect <- b.PoTProof
	}
}

// 检查该区块或者交易的广播数据originData是否收到过，收到过返回false
// 如果以前没收到过，那么帮忙接替广播relayBroadcast
func (n *net) relayBroadcast(originData []byte) bool {
	h := crypto.HashD(originData)
	encoded := base64.StdEncoding.EncodeToString(h)
	if _, ok := n.broadcastFilter[encoded]; ok {
		return false
	}

	n.broadcastFilter[encoded] = time.Now()
	n.broadcast(originData) // 帮忙广播
	return true
}
