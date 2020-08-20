package db

import (
	"bytes"
	"fmt"
	"github.com/azd1997/ecoin/common/crypto"
	"os"
	"testing"

	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/protocol/core"
)

// NOTICE: 测试时会因为交易是随便构造的，会出现报错：余额不足

var dbTestVar = &struct {
	dbPath string

	genesis         *core.Block
	secondBlock     *core.Block
	thirdEmptyBlock *core.Block
	genesisHeight   uint64
	secondHeight    uint64
	thirdHeight     uint64

	txA       *core.Tx
	txB       *core.Tx
	txAHeight uint64
	txBHeight uint64

	// its creator is the 2th block creator
	// and one of its tx(txC) owner is the txA initiator
	fourthBlock                 *core.Block
	txC                   *core.Tx
	fourthHeight                uint64
	secondBlockMinerExpectScore uint64
}{
	genesisHeight:               1,
	secondHeight:                2,
	thirdHeight:                 3,
	fourthHeight:                4,
	secondBlockMinerExpectScore: 2,
}

func init() {
	tv := dbTestVar

	tv.genesis = core.GenBlockFromParams(core.NewBlockParams(false))
	tv.secondBlock = core.GenBlockFromParams(core.NewBlockParams(false))
	tv.thirdEmptyBlock = core.GenBlockFromParams(core.NewBlockParams(true))

	tv.txA = tv.genesis.Txs[0]
	tv.txAHeight = tv.genesisHeight
	tv.txB = tv.secondBlock.Txs[0]
	tv.txBHeight = tv.secondHeight

	tv.fourthBlock = core.GenBlockFromParams(core.NewBlockParams(false))
	tv.txC = tv.fourthBlock.Txs[0]
	tv.fourthBlock.CreateBy = tv.secondBlock.CreateBy
	tv.txC.From = tv.txA.From
}

func setup() {
	tv := dbTestVar

	runningDir, err := os.Getwd()
	if err != nil {
		logger.Fatalln(err)
	}

	tv.dbPath = runningDir + "/db_test_tmp"
	if err := os.MkdirAll(tv.dbPath, 0700); err != nil {
		logger.Fatal("create tmp directory failed:%v\n", err)
	}

	if err := Init(tv.dbPath); err != nil {
		logger.Fatal("initialize db failed:%v\n", err)
	}
}

func cleanup() {
	tv := dbTestVar

	Close()
	if err := os.RemoveAll(tv.dbPath); err != nil {
		logger.Fatal("remove tmp directory failed:%v\n", err)
	}
}

func insertGenesis(t *testing.T) {
	tv := dbTestVar

	if err := PutGenesis(tv.genesis); err != nil {
		t.Fatalf("insert genesis failed:%v\n", err)
	}
}

func insertTestData(t *testing.T) {
	tv := dbTestVar

	insertGenesis(t)
	if err := PutBlock(tv.secondBlock, tv.secondHeight); err != nil {
		t.Fatalf("insert the second block failed:%v\n", err)
	}
	if err := PutBlock(tv.thirdEmptyBlock, tv.thirdHeight); err != nil {
		t.Fatalf("insert the third block failed:%v\n", err)
	}
	if err := PutBlock(tv.fourthBlock, tv.fourthHeight); err != nil {
		t.Fatalf("insert the fourth block failed:%v\n", err)
	}
}

func TestGenesis(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()

	// exists test
	if HasGenesis() {
		t.Fatalf("expect no genesis block exists\n")
	}

	insertGenesis(t)
	if !HasGenesis() {
		t.Fatalf("expect genesis block exists\n")
	}

	// recover test
	dbBlock, hash, err := GetBlockViaHeight(tv.genesisHeight)
	if err != nil {
		t.Fatal(err)
	}

	h := tv.genesis.CalcHash()
	if err := utils.TCheckBytes("block hash", h, hash); err != nil {
		t.Fatal(err)
	}

	checkBlock(t, "", tv.genesis, dbBlock)
}

func TestGetHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		height     uint64
		expectHash []byte
	}{
		{tv.genesisHeight, tv.genesis.CalcHash()},
		{tv.secondHeight, tv.secondBlock.CalcHash()},
		{tv.thirdHeight, tv.thirdEmptyBlock.CalcHash()},
	}

	for i, cs := range cases {
		result, err := GetHash(cs.height)
		if err != nil {
			t.Fatal(err)
		}

		if err := utils.TCheckBytes(fmt.Sprintf("[%d] test case", i),
			cs.expectHash, result); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetHeaderViaHeight(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		height       uint64
		expectHeader *core.BlockHeader
		expectHash   []byte
	}{
		{tv.genesisHeight, tv.genesis.BlockHeader, tv.genesis.CalcHash()},
		{tv.secondHeight, tv.secondBlock.BlockHeader, tv.secondBlock.CalcHash()},
		{tv.thirdHeight, tv.thirdEmptyBlock.BlockHeader, tv.thirdEmptyBlock.CalcHash()},
	}

	for i, cs := range cases {
		header, hash, err := GetHeaderViaHeight(cs.height)
		if err != nil {
			t.Fatal(err)
		}

		checkHeader(t, fmt.Sprintf("[%d] ", i), cs.expectHeader, header)

		if err := utils.TCheckBytes(fmt.Sprintf("[%d] hash", i), cs.expectHash, hash); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetHeaderViaHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		hash         []byte
		expectHeader *core.BlockHeader
		expectHeight uint64
	}{
		{tv.genesis.CalcHash(), tv.genesis.BlockHeader, tv.genesisHeight},
		{tv.secondBlock.CalcHash(), tv.secondBlock.BlockHeader, tv.secondHeight},
		{tv.thirdEmptyBlock.CalcHash(), tv.thirdEmptyBlock.BlockHeader, tv.thirdHeight},
	}

	for i, cs := range cases {
		header, height, err := GetHeaderViaHash(cs.hash)
		if err != nil {
			t.Fatal(err)
		}

		checkHeader(t, fmt.Sprintf("[%d] ", i), cs.expectHeader, header)

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] height ", i), cs.expectHeight, height); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetBlockViaHeight(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		height      uint64
		expectBlock *core.Block
		expectHash  []byte
	}{
		{tv.genesisHeight, tv.genesis, tv.genesis.CalcHash()},
		{tv.secondHeight, tv.secondBlock, tv.secondBlock.CalcHash()},
		{tv.thirdHeight, tv.thirdEmptyBlock, tv.thirdEmptyBlock.CalcHash()},
	}

	for i, cs := range cases {
		block, hash, err := GetBlockViaHeight(cs.height)
		if err != nil {
			t.Fatal(err)
		}

		checkBlock(t, fmt.Sprintf("[%d] block ", i), cs.expectBlock, block)

		if err := utils.TCheckBytes(fmt.Sprintf("[%d] hash", i), cs.expectHash, hash); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetBlockViaHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		hash         []byte
		expectBlock  *core.Block
		expectHeight uint64
	}{
		{tv.genesis.CalcHash(), tv.genesis, tv.genesisHeight},
		{tv.secondBlock.CalcHash(), tv.secondBlock, tv.secondHeight},
		{tv.thirdEmptyBlock.CalcHash(), tv.thirdEmptyBlock, tv.thirdHeight},
	}

	for i, cs := range cases {
		block, height, err := GetBlockViaHash(cs.hash)
		if err != nil {
			t.Fatal(err)
		}

		checkBlock(t, fmt.Sprintf("[%d] block ", i), cs.expectBlock, block)

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] height ", i), cs.expectHeight, height); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetTxViaHash(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		hash           []byte
		expectTx *core.Tx
		expectHeight   uint64
	}{
		{tv.txA.Hash(), tv.txA, tv.txAHeight},
		{tv.txB.Hash(), tv.txB, tv.txBHeight},
	}

	for i, cs := range cases {
		tx, height, err := GetTxViaHash(cs.hash)
		if err != nil {
			t.Fatal(err)
		}

		checkTx(t, fmt.Sprintf("[%d] tx ", i), cs.expectTx, tx)

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] height ", i), cs.expectHeight, height); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetTxHashesViaKey(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		userid           crypto.ID
		expectedTxs  [][]byte
		expectHeights []uint64
	}{
		{tv.txA.From, [][]byte{tv.txA.Hash(), tv.txC.Hash()}, []uint64{tv.txAHeight}},
		{tv.txB.From, [][]byte{tv.txB.Hash()}, []uint64{tv.txBHeight}},
	}

	for i, cs := range cases {
		txs, heights, err := GetTxFromHashesViaID(cs.userid)
		if err != nil {
			t.Fatal(err)
		}

		if err := utils.TCheckInt(fmt.Sprintf("[%d] tx size", i), len(cs.expectedTxs), len(txs)); err != nil {
			t.Fatal(err)
		}

		for j := len(txs); j < len(txs); j++ {
			if err := utils.TCheckBytes(fmt.Sprintf("[%d-%d] tx hash ", i, j), cs.expectedTxs[j], txs[j]); err != nil {
				t.Fatal(err)
			}
			if err := utils.TCheckUint64(fmt.Sprintf("[%d-%d] height", i, j), cs.expectHeights[j], heights[j]); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestHasTx(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	if HasTx([]byte("not_exist_tx")) {
		t.Fatal("expect not found")
	}

	if !HasTx(tv.txA.Hash()) {
		t.Fatal("expect txA exist")
	}
}

func TestGetBalanceViaKey(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()
	insertTestData(t)

	cases := []struct {
		miner       crypto.ID
		expectBalance uint64
	}{
		{tv.thirdEmptyBlock.CreateBy, 1},
		{tv.secondBlock.CreateBy, tv.secondBlockMinerExpectScore},
		{crypto.RandID(), 0},
	}

	for i, cs := range cases {
		score, err := GetBalanceViaID(cs.miner)
		if err != nil {
			t.Fatal(err)
		}

		if err := utils.TCheckUint64(fmt.Sprintf("[%d] score", i), cs.expectBalance, score); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGetLatestHeight(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()

	_, err := GetLatestHeight()
	if err == nil {
		t.Fatalf("expect error\n")
	}

	insertGenesis(t)
	height, _ := GetLatestHeight()
	if err := utils.TCheckUint64("height", tv.genesisHeight, height); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.secondBlock, tv.secondHeight)
	height, _ = GetLatestHeight()
	if err := utils.TCheckUint64("height", tv.secondHeight, height); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.thirdEmptyBlock, tv.thirdHeight)
	height, _ = GetLatestHeight()
	if err := utils.TCheckUint64("height", tv.thirdHeight, height); err != nil {
		t.Fatal(err)
	}
}

func TestGetLatestHeader(t *testing.T) {
	tv := dbTestVar
	setup()
	defer cleanup()

	_, _, _, err := GetLatestHeader()
	if err == nil {
		t.Fatalf("expect error\n")
	}

	insertGenesis(t)
	header, height, hash, _ := GetLatestHeader()
	checkHeader(t, "latest header", tv.genesis.BlockHeader, header)
	if err := utils.TCheckUint64("1st latest header height", tv.genesisHeight, height); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("1st latest header hash", tv.genesis.CalcHash(), hash); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.secondBlock, tv.secondHeight)
	header, height, hash, _ = GetLatestHeader()
	checkHeader(t, "latest header", tv.secondBlock.BlockHeader, header)
	if err := utils.TCheckUint64("2th latest header height", tv.secondHeight, height); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("2th latest header hash", tv.secondBlock.CalcHash(), hash); err != nil {
		t.Fatal(err)
	}

	PutBlock(tv.thirdEmptyBlock, tv.thirdHeight)
	header, height, hash, _ = GetLatestHeader()
	checkHeader(t, "latest header", tv.thirdEmptyBlock.BlockHeader, header)
	if err := utils.TCheckUint64("3th latest header height", tv.thirdHeight, height); err != nil {
		t.Fatal(err)
	}
	if err := utils.TCheckBytes("3th latest header hash", tv.thirdEmptyBlock.CalcHash(), hash); err != nil {
		t.Fatal(err)
	}
}

func checkTx(t *testing.T, prefix string, expect *core.Tx, result *core.Tx) {
	expectBytes := expect.Encode()
	resultBytes := result.Encode()

	if !bytes.Equal(expectBytes, resultBytes) {
		t.Fatalf("%s tx mismatch, expect %v, result %v\n", prefix, expect, result)
	}
}

func checkHeader(t *testing.T, prefix string, expect *core.BlockHeader, result *core.BlockHeader) {
	expectBytes := expect.Encode()
	resultBytes := result.Encode()

	if !bytes.Equal(expectBytes, resultBytes) {
		t.Fatalf("%s header mismatch, expect %v, result %v\n", prefix, expect, result)
	}
}

func checkBlock(t *testing.T, prefix string, expect *core.Block, result *core.Block) {
	expectBytes := expect.Encode()
	resultBytes := result.Encode()

	if !bytes.Equal(expectBytes, resultBytes) {
		t.Fatalf("%s block mismatch, expect %v, result %v\n", prefix, expect, result)
	}
}
