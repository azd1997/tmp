package core

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/azd1997/ecoin/common/utils"
)

func TestHead(t *testing.T) {
	head := NewHeadV1(MsgSyncReq)
	headBytes := head.Encode()

	rHead := &Head{}
	err := rHead.Decode(bytes.NewReader(headBytes))
	if err != nil {
		t.Fatalf("decode Head failed: %v\n", err)
	}

	if err := utils.TCheckUint8("version", V1, rHead.Version); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint8("type", MsgSyncReq, rHead.Type); err != nil {
		t.Fatal(err)
	}
}

func TestTx(t *testing.T) {
	tp := NewTxParams(TX_GENERAL)

	tx := GenTxFromParams(tp)
	txBytes := tx.Encode()

	rTx := &Tx{}
	err := rTx.Decode(bytes.NewReader(txBytes))
	if err != nil {
		t.Fatalf("decode tx failed: %v\n", err)
	}

	if err := CheckTx(rTx, tp); err != nil {
		t.Fatal(err)
	}
}

func TestTxVerify(t *testing.T) {
	failedTx := GenTxFromParams(NewTxParams(TX_GENERAL))

	var tx *Tx
	tx = GenTxFromParams(NewTxParams(TX_GENERAL))
	//fmt.Println(tx)
	time.Sleep(2*time.Second)	// 由于模拟测试执行太快，时间戳会和交易创建时间戳一样，这里故意睡两秒
	if err := tx.Verify(); err != nil {
		//fmt.Println(tx.From, len(tx.From))
		t.Fatalf("expect valid, but %v\n", err)
	}

	tx = GenTxFromParams(NewTxParams(TX_GENERAL))
	tx.Version = 0
	if err := tx.Verify(); err == nil {
		t.Fatal("expect version error")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	tx = GenTxFromParams(NewTxParams(TX_GENERAL))
	tx.From = failedTx.From
	if err := tx.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	tx = GenTxFromParams(NewTxParams(TX_GENERAL))
	tx.Sig = failedTx.Sig
	if err := tx.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	tx = GenTxFromParams(NewTxParams(TX_GENERAL))
	tx.Description = failedTx.Description
	if err := tx.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}

	tx = GenTxFromParams(NewTxParams(TX_GENERAL))
	tx.Id = failedTx.Id
	if err := tx.Verify(); err == nil {
		t.Fatal("expect verify failed")
	} else {
		t.Logf("expect err:%v\n", err)
	}
}

func TestBlockHeader(t *testing.T) {
	bp := NewBlockHeaderParams()

	blockHeader := GenBlockHeaderFromParams(bp)
	blockHeaderBytes := blockHeader.Encode()

	rBlockHeader := &BlockHeader{}
	err := rBlockHeader.Decode(bytes.NewReader(blockHeaderBytes))
	if err != nil {
		t.Fatalf("decode block header failed: %v\n", err)
	}

	if err := utils.TCheckInt64("time", blockHeader.Time, rBlockHeader.Time); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlockHeader(rBlockHeader, bp); err != nil {
		t.Fatal(err)
	}

	if err := rBlockHeader.Verify(); err != nil {
		t.Fatalf("expect valid:%v\n", err)
	}
}

func TestEmptyBlock(t *testing.T) {
	bp := NewBlockParams(true)
	block := GenBlockFromParams(bp)
	blockBytes := block.Encode()
	t.Logf("empty block size:%d\n", len(blockBytes))
	//fmt.Println(block.MerkleRoot)

	rBlock := &Block{}
	err := rBlock.Decode(bytes.NewReader(blockBytes))
	if err != nil {
		t.Fatalf("decode block failed: %v\n", err)
	}

	if err := utils.TCheckInt64("time", block.Time, rBlock.Time); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlock(rBlock, bp); err != nil {
		t.Fatal(err)
	}

	if err := rBlock.Verify(); err != nil {
		t.Fatalf("expect valid:%v\n", err)
	}
}

func TestBlock(t *testing.T) {
	bp := NewBlockParams(false)
	block := GenBlockFromParams(bp)
	time.Sleep(1*time.Second)	// tx检查会检查时间戳，这里故意等1s
	blockBytes := block.Encode()

	rBlock := &Block{}
	err := rBlock.Decode(bytes.NewReader(blockBytes))
	if err != nil {
		t.Fatalf("decode block failed: %v\n", err)
	}

	if err := utils.TCheckInt64("time", block.Time, rBlock.Time); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlock(rBlock, bp); err != nil {
		t.Fatal(err)
	}

	if err := rBlock.Verify(); err != nil {
		t.Fatalf("expect valid:%v\n", err)
	}
}

func TestSyncRequest(t *testing.T) {
	base := randHash()

	req := NewSyncReqMsg(base)
	reqBytes := req.Encode()

	rReq := &SyncReqMsg{}
	err := rReq.Decode(bytes.NewReader(reqBytes))
	if err != nil {
		t.Fatalf("decode SyncReqMsg failed: %v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgSyncReq, rReq.Type); err != nil {
		t.Fatal(err)
	}

	if err := utils.TCheckBytes("base", base, rReq.Base); err != nil {
		t.Fatal(err)
	}
}

func TestSyncResponse(t *testing.T) {
	base := randHash()
	end := randHash()
	heightDiff := uint32(0)

	resp := NewSyncRespMsg(base, end, heightDiff)
	respBytes := resp.Encode()

	rResp := &SyncRespMsg{}
	err := rResp.Decode(bytes.NewReader(respBytes))
	if err != nil {
		t.Fatalf("decode SyncRespMsg failed: %v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgSyncResp, rResp.Type); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("base", base, rResp.Base); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("end", end, rResp.End); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckUint32("height diff", heightDiff, resp.HeightDiff); err != nil {
		t.Fatal(err)
	}
	if !rResp.IsUptodate() {
		t.Fatalf("expect uptodate\n")
	}
}

func TestBlockRequest(t *testing.T) {
	base := randHash()
	end := randHash()

	req := NewBlockReqMsg(base, end, true)
	reqBytes := req.Encode()

	rReq := &BlockReqMsg{}
	err := rReq.Decode(bytes.NewReader(reqBytes))
	if err != nil {
		t.Fatalf("decode BlockReqMsg failed: %v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgBlockReq, rReq.Type); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("base", base, rReq.Base); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("end", end, rReq.End); err != nil {
		t.Fatal(err)
	}
	if !rReq.IsOnlyHeader() {
		t.Fatalf("expect only header\n")
	}
}

func TestBlockResponse(t *testing.T) {
	blockAParams := NewBlockParams(true)
	blockA := GenBlockFromParams(blockAParams)

	blockBParams := NewBlockParams(false)
	blockB := GenBlockFromParams(blockBParams)

	// response
	resp := NewBlockRespMsg([]*Block{blockA, blockB})
	respBytes := resp.Encode()

	rResp := &BlockRespMsg{}
	err := rResp.Decode(bytes.NewReader(respBytes))
	if err != nil {
		t.Fatalf("decode BlockRespMsg failed: %v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgBlockResp, rResp.Type); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("block num", 2, len(rResp.Blocks)); err != nil {
		t.Fatal(err)
	}

	if err := CheckBlock(rResp.Blocks[0], blockAParams); err != nil {
		t.Fatal(err)
	}
	if err := CheckBlock(rResp.Blocks[1], blockBParams); err != nil {
		t.Fatal(err)
	}
}

func TestBlockBroadcast(t *testing.T) {
	blockParams := NewBlockParams(false)
	block := GenBlockFromParams(blockParams)

	broadcast := NewBlockBroadcastMsg(block)
	broadcastBytes := broadcast.Encode()

	rBroadcast := &BlockBroadcastMsg{}
	err := rBroadcast.Decode(bytes.NewReader(broadcastBytes))
	if err != nil {
		t.Fatalf("decode BlockBroadcastMsg failed: %v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgBlockBroadcast, rBroadcast.Type); err != nil {
		t.Fatal(err)
	}
	if err := CheckBlock(rBroadcast.Block, blockParams); err != nil {
		t.Fatal(err)
	}
}

func TestTxBroadcastMsg(t *testing.T) {
	txsNum := rand.Intn(10) + 1 // at least one evidence

	var txsParams []*TxParams
	var txs []*Tx
	for i := 0; i < txsNum; i++ {
		params := NewTxParams(uint8(rand.Intn(2) + 1))
		txsParams = append(txsParams, params)
		txs = append(txs, GenTxFromParams(params))
	}

	broadcast := NewTxBroadcastMsg(txs)
	broadcastBytes := broadcast.Encode()

	rBroadcast := &TxBroadcastMsg{}
	err := rBroadcast.Decode(bytes.NewReader(broadcastBytes))
	if err != nil {
		t.Fatalf("decode TxBroadcastMsg failed: %v\n", err)
	}

	if err := utils.TCheckUint8("type", MsgTxBroadcast, rBroadcast.Type); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckInt("evidence number", txsNum, len(rBroadcast.Txs)); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < txsNum; i++ {
		if err := CheckTx(rBroadcast.Txs[i], txsParams[i]); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProofBroadcastMsg(t *testing.T) {

}
