package rpc

import (
	"encoding/json"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/enode/bc"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/protocol/raw"
	"github.com/azd1997/ecoin/protocol/view"
	"io/ioutil"
	"net/http"
)

const (
	txPath          = "/tx"
	maxBatchQueryNum = 40
)

var (

	// TxV1Path /v1/tx
	TxV1Path = version1Path + txPath

	// UploadTxV1Path POST /v1/tx/upload
	UploadTxV1Path = TxV1Path + "/upload"

	// UploadTxRawV1Path POST /v1/tx/upload-raw
	UploadTxRawV1Path = version1Path + txPath + "/upload-raw"

	// QueryTxV1Path POST /v1/tx/query
	QueryTxV1Path = TxV1Path + "/query"

	txHandlers = HTTPHandlers{
		{UploadTxV1Path, uploadTxs},
		{UploadTxRawV1Path, uploadRaw},
		{QueryTxV1Path, queryTx},
	}
)

/*
POST /v1/tx/upload
{
	"data": [{
		"version": 1,
		"hash": "xxxx",
		"description": "xxxx",
		"user_id": "xxxx",
		"signature": "xxxx",
		...
	}]
}
*/

type UploadTxsReq struct {
	Data []*view.TxJSON `json:"data"`
}

// 批量上传交易的handler
func uploadTxs(w http.ResponseWriter, r *http.Request) {
	// 1. 读取r.Body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		badRequestResponse(w)
		return
	}
	// 2. 解码UploadTxsReq
	query := &UploadTxsReq{}
	if err := json.Unmarshal(body, query); err != nil {
		badRequestResponse(w)
		return
	}
	// 3. 转为core.Tx列表
	var txs []*core.Tx
	for _, data := range query.Data {
		tx := data.ToCoreTx()
		if tx == nil {
			badRequestResponse(w)
			return
		}
		txs = append(txs, tx)
	}
	// 4. 构建Tx
	if err := globalSvr.en.BuildTx(txs); err != nil {
		if existErr, ok := err.(bc.ErrTxAlreadyExist); ok {
			failedResponse(existErr.Error(), w)
			return
		}

		logger.Info("build failed:%v\n", err)
		badRequestResponse(w)
		return
	}

	successResponse(w)
}

/*
POST /v1/tx/upload-raw
{
   "txs":[
      {
         "hash":"xxx",
         "description":"yyy"
      },
      {
         "hash":"xxx",
         "description":"yyy"
      }
   ]
}
*/


type uploadRawReq struct {
	Txs []*view.RawTxJSON `json:"txs"`
}

// 上传rawTx(原始交易)的handler。 换句话说就是以本机账户作为发起方创建交易的handler
func uploadRaw(w http.ResponseWriter, r *http.Request) {
	// 1. 读取r.Body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		badRequestResponse(w)
		return
	}
	// 2. 解码UploadRawReq
	query := &uploadRawReq{}
	if err := json.Unmarshal(body, query); err != nil {
		badRequestResponse(w)
		return
	}
	// 3. 转为raw.Tx
	var rTxs []*raw.Tx
	for _, tx := range query.Txs {
		rTx, err := tx.ToRawTx()
		if err != nil {
			badRequestResponse(w)
			return
		}

		rTxs = append(rTxs, rTx)
	}
	// 4. 根据raw.Tx创建交易
	if err := globalSvr.en.BuildTxByRaw(rTxs); err != nil {
		logger.Info("build raw failed:%v\n", err)
		badRequestResponse(w)
		return
	}

	successResponse(w)
}

/*

POST /v1/tx/query
{
	"hash":["xxxx", "xxxx", "xxxx"]
}
*/

type QueryTxReq struct {
	Hashes []string `json:"hash"`	 // hex(hash)
}

type QueryTxResp struct {
	Data []*view.TxJSON `json:"data"`
}

func queryTx(w http.ResponseWriter, r *http.Request) {
	// 1. 读取r.Body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		badRequestResponse(w)
		return
	}

	// 2. 解码QueryTxReq
	query := &QueryTxReq{}
	if err := json.Unmarshal(body, query); err != nil {
		badRequestResponse(w)
		return
	}
	// 3. 检查请求项长度
	if len(query.Hashes) == 0 || len(query.Hashes) > maxBatchQueryNum {
		badRequestResponse(w)
		return
	}
	// 4. 检查哈希长度合法性
	for _, hexHash := range query.Hashes {
		h, err := encoding.FromHex(hexHash)
		if err != nil || len(h) != crypto.HASH_LENGTH {
			badRequestResponse(w)
			return
		}
	}
	// 5. 查询TxInfos
	txInfos := globalSvr.en.QueryTx(query.Hashes)
	if txInfos == nil {
		failedResponse("Not found tx", w)
		return
	}
	// 6. 转为TxJSON
	resp := &QueryTxResp{}
	for _, t := range txInfos {
		txJSON := &view.TxJSON{}
		txJSON.FromTxInfo(t)
		resp.Data = append(resp.Data, txJSON)
	}

	successWithDataResponse(resp, w)
	return
}
