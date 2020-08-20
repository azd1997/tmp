/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/14 11:45
* @Description: The file is for
***********************************************************************/

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/store/db"
)

var output *os.File

func rangeView(r string) error {
	var begin, end uint64

	num, err := strconv.ParseInt(r, 10, 64)
	if err == nil {
		if num == -1 {
			height, err := db.GetLatestHeight()
			if err != nil {
				return fmt.Errorf("err %v", err)
			}
			begin = height
			end = height
		} else if num > 0 {
			begin = uint64(num)
			end = uint64(num)
		} else {
			return fmt.Errorf("invalid index %d", num)
		}
	} else {
		n, err := fmt.Sscanf(r, "%d-%d", &begin, &end)
		if err != nil || n != 2 || begin >= end {
			return fmt.Errorf("invalid range")
		}
	}

	for i := begin; i <= end; i++ {
		block, hash, err := db.GetBlockViaHeight(i)
		if err != nil {
			return fmt.Errorf("get height %d block failed", i)
		}
		formatOutputBlock(block, hash, i)
	}

	return nil
}

func blockView(hash string) error {
	decoded, err := encoding.FromHex(hash)
	if err != nil {
		return fmt.Errorf("decode %s failed", hash)
	}
	if len(decoded) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid hash size %d", len(decoded))
	}

	block, height, err := db.GetBlockViaHash(decoded)
	if err != nil {
		return fmt.Errorf("get block via %s failed", hash)
	}

	formatOutputBlock(block, decoded, height)
	return nil
}

func txView(hash string) error {
	decoded, err := encoding.FromHex(hash)
	if err != nil {
		return fmt.Errorf("decode %s failed", hash)
	}
	if len(decoded) != crypto.HASH_LENGTH {
		return fmt.Errorf("invalid hash size %d", len(decoded))
	}

	evidence, _, err := db.GetTxViaHash(decoded)
	if err != nil {
		return fmt.Errorf("get tx via %s failed", hash)
	}
	formatOutputTx(evidence)
	return nil
}

func wirte(format string, v ...interface{}) {
	if _, err := output.Write([]byte(fmt.Sprintf(format, v...))); err != nil {
		fmt.Printf("output err:%v\n", err)
		os.Exit(1)
	}
}

func formatOutputBlock(block *core.Block, hash []byte, height uint64) {
	format :=
		`>>>>> [Block %d] %X
Version		%d
Time		%s
PrevHash	%X
CreateBy	%X
MerkleRoot	%s

`

	var txRoot string
	empty := false
	if block.IsEmptyMerkleRoot() {
		txRoot = "EMPTY"
		empty = true
	} else {
		txRoot = encoding.ToHex(block.MerkleRoot)
	}

	wirte(format, height, hash,
		block.Version,
		utils.TimeToString(block.Time),
		block.PrevHash,
		block.CreateBy,
		txRoot)

	if empty {
		return
	}
	for _, tx := range block.Txs {
		formatOutputTx(tx)
	}
}

func formatOutputTx(tx *core.Tx) {
	format :=
		`[Tx] %X
version			%d
type			%d
uncompleted		%d
time			%s
hash			%X
from			%s
to				%s
amount			%d
signature		%X
payload			%s
prevtx			%X
description		%s

`
	wirte(format, tx.Hash,
		tx.Version,
		tx.Type,
		tx.Uncompleted,
		utils.TimeToString(tx.TimeUnix),
		tx.Id,
		tx.From,
		tx.To,
		tx.Amount,
		tx.Sig,
		tx.Payload,
		tx.PrevTxId,
		tx.Description)
}


