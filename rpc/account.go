package rpc

import (
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"net/http"
)

const (
	accountPath = "/account"
)

var (
	// AccountV1Path /v1/account
	AccountV1Path = version1Path + accountPath

	// QueryAccountV1Path GET /v1/account
	QueryAccountV1Path = AccountV1Path + "/query"

	accountHandlers = HTTPHandlers{
		{QueryAccountV1Path, getAccountInfo},
	}
)

/*
GET /v1/account/query?id=...
*/
type GetAccountResponse struct {
	TxsFrom []string `json:"txs_from"`
	TxsTo []string `json:"txs_to"`
	Balance    uint64   `json:"balance"`
	Credit int64	`json:"credit"`		// TODO
}

func getAccountInfo(w http.ResponseWriter, r *http.Request) {
	// 1. 获取ID参数
	id, ok := r.URL.Query()[GetIDParam]
	if !ok {
		badRequestResponse(w)
		return
	}

	// 2. 检查ID参数
	if len(id[0]) != crypto.ID_LEN_WITH_ROLE {
		badRequestResponse(w)
		return
	}

	// 3. 查询账户相关交易Id
	txFromHashes, txToHashes, balance, credit := globalSvr.en.QueryAccount(crypto.ID(id[0]))
	if txFromHashes == nil && txToHashes == nil && balance == 0 {
		failedResponse("Not found account", w)
		return
	}

	// 十六进制编码
	var hexFromHashes []string
	for _, h := range txFromHashes {
		hexFromHashes = append(hexFromHashes, encoding.ToHex(h))
	}
	var hexToHashes []string
	for _, h := range txToHashes {
		hexToHashes = append(hexToHashes, encoding.ToHex(h))
	}

	successWithDataResponse(&GetAccountResponse{
		TxsFrom:hexFromHashes,
		TxsTo:hexToHashes,
		Balance:    balance,
		Credit:credit,
	}, w)
}
