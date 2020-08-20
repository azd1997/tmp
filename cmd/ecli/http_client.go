/**********************************************************************
* @Author: Eiger (201820114847@mail.scut.edu.cn)
* @Date: 2020/4/6 15:48
* @Description: The file is for
***********************************************************************/

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/azd1997/ecoin/protocol/view"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/azd1997/ecoin/account"
	"github.com/azd1997/ecoin/common/crypto"
	"github.com/azd1997/ecoin/common/encoding"
	"github.com/azd1997/ecoin/common/utils"
	"github.com/azd1997/ecoin/protocol/core"
	"github.com/azd1997/ecoin/rpc"
)

type httpClient struct {
	serverIP   string
	serverPort string
	scheme     string
	acc *account.Account
	client     *http.Client
}

func newHTTPClient(ip string, port int, scheme string,
	acc *account.Account) *httpClient {
	return &httpClient{
		serverIP:   ip,
		serverPort: strconv.Itoa(port),
		scheme:     scheme,
		acc:acc,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (hc *httpClient) generateTx(typ, uncompleted uint8,
	from, to, description, payload, prevTxId string, amount uint32) (*core.Tx, error) {

	// 检查prevTxId
	prevTx, err := encoding.FromHex(prevTxId)
	if err != nil {
		return nil, fmt.Errorf("hex decode prevTxId failed:%v", err)
	}
	if len(prevTx) != crypto.HASH_LENGTH {
		return nil, fmt.Errorf("invalid hash length")
	}
	// 检查from
	fromID, err := encoding.FromHex(from)
	if err != nil {
		return nil, fmt.Errorf("hex decode fromID failed:%v", err)
	}
	if len(fromID) != crypto.ID_LEN_WITH_ROLE {
		return nil, fmt.Errorf("invalid id length")
	}
	// 检查to
	toID, err := encoding.FromHex(to)
	if err != nil {
		return nil, fmt.Errorf("hex decode toID failed:%v", err)
	}
	if len(toID) != crypto.ID_LEN_WITH_ROLE {
		return nil, fmt.Errorf("invalid id length")
	}

	tx := core.NewTx(typ, crypto.ID(fromID), crypto.ID(toID), amount, []byte(payload),
		prevTx, uncompleted, []byte(description))
	tx.Sign(hc.acc.PrivateKey)

	return tx, nil
}

func (hc *httpClient) uploadTx(tx *core.Tx) error {
	txJSON := &view.TxJSON{
		Version:     core.V1,
		Id:        encoding.ToHex(tx.Id),
		Description: string(tx.Description),
		From:     encoding.ToHex([]byte(tx.From)),
		To:     encoding.ToHex([]byte(tx.To)),
		Amount:tx.Amount,
		Payload:encoding.ToHex(tx.Payload),
		Sig:         encoding.ToHex(tx.Sig),

		// ignore other fileds
	}

	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse
	var requestBody []byte

	if requestBody, err = json.Marshal(
		&rpc.UploadTxsReq{
			Data: []*view.TxJSON{txJSON}}); err != nil {
		return err
	}

	if req, err = hc.genRequest(http.MethodPost, rpc.UploadTxV1Path, nil, nil, requestBody); err != nil {
		return err
	}

	if httpResp, err = hc.client.Do(req); err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	if rpcResp, err = hc.parseResponse(httpResp, nil); err != nil {
		return err
	}

	handler := func() {
		fmt.Println(">>> upload tx successfully")
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) queryAccount(accIDHex string) error {
	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse
	var accountID crypto.ID

	if accIDHex == "" {
		accountID = crypto.PrivateKey2ID(hc.acc.PrivateKey, hc.acc.RoleNo)
	} else  {
		accID, err := encoding.FromHex(accIDHex)
		if err != nil {
			log.Fatalf("please input a valid accountIDHex: %s\n", err)
		}
		accountID := crypto.ID(accID)
		if !accountID.IsValid() {
			log.Fatalf("please input a valid accountIDHex: %s\n", err)
		}
	}

	if req, err = hc.genRequest(http.MethodGet, rpc.QueryAccountV1Path,
		[]string{rpc.GetIDParam}, []string{accountID.ToHex()}, nil); err != nil {
		return err
	}

	if httpResp, err = hc.client.Do(req); err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	accountJSON := &rpc.GetAccountResponse{}
	if rpcResp, err = hc.parseResponse(httpResp, accountJSON); err != nil {
		return err
	}

	handler := func() {
		content := "Account\t<%s>\nBalance:\t%d\nCredit:\t%d\ntxsFrom:\n%s\ntxsTo:\n%s\n"

		var txFromContent string
		for i := 0; i < len(accountJSON.TxsFrom); i++ {
			record := fmt.Sprintf("\t%d.%s\n", i+1, accountJSON.TxsFrom[i])
			txFromContent += record
		}
		var txToContent string
		for i := 0; i < len(accountJSON.TxsTo); i++ {
			record := fmt.Sprintf("\t%d.%s\n", i+1, accountJSON.TxsTo[i])
			txToContent += record
		}

		fmt.Printf(content, accountID, accountJSON.Balance, accountJSON.Credit, txFromContent, txToContent)
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) queryTx(params string) error {
	hexHashes := strings.Split(params, ",")
	for _, hash := range hexHashes {
		h, err := encoding.FromHex(hash)
		if err != nil || len(h) != crypto.HASH_LENGTH {
			return fmt.Errorf("invalid tx hash %s", hash)
		}
	}

	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse
	var requestBody []byte

	if requestBody, err = json.Marshal(&rpc.QueryTxReq{Hashes: hexHashes}); err != nil {
		return err
	}

	if req, err = hc.genRequest(http.MethodPost, rpc.QueryTxV1Path, nil, nil, requestBody); err != nil {
		return err
	}

	httpResp, err = hc.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	queryTxResp := &rpc.QueryTxResp{}
	if rpcResp, err = hc.parseResponse(httpResp, queryTxResp); err != nil {
		return err
	}

	handler := func() {
		for _, tx := range queryTxResp.Data {
			content := "Tx <%s>\n[Version] %d\n[Type] %d\n[Uncompleted] %d\n[Time] %d\n[From] %s\n[To] %s\n[Amount] %d\n[Signature] %s\n[Payload] %s\n[Description] %s\n[PrevTxId] %s\n[Height] %d\n[Block] %s\n[BlockTime] %s\n\n"

			fmt.Println("--------------------------------------------------------")
			fmt.Printf(content, tx.Id, tx.Version, tx.Type, tx.Uncompleted, tx.Time,
				tx.From, tx.To, tx.Amount, tx.Sig, tx.Payload, tx.Description,
				tx.PrevTxId, tx.Height, tx.BlockHash, utils.TimeToString(tx.Time))
		}
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) queryBlocks(params string) error {
	var err error
	var req *http.Request
	var httpResp *http.Response
	var rpcResp *rpc.HTTPResponse

	if req, err = hc.genRequest(http.MethodGet, rpc.QueryBlockViaRangeV1Path,
		[]string{rpc.GetRangeParam}, []string{params}, nil); err != nil {
		return err
	}

	if httpResp, err = hc.client.Do(req); err != nil {
		return fmt.Errorf("do request err:%v", err)
	}
	defer httpResp.Body.Close()

	blocksResponse := &rpc.GetBlocksResponse{}
	if rpcResp, err = hc.parseResponse(httpResp, blocksResponse); err != nil {
		return err
	}

	handler := func() {
		for _, block := range blocksResponse.Data {
			blockContent := `
Block <%s> Height:%d
Time		%s
Version		%d
PrevHash	%s
MerkleRoot	%s
CreateBy	%s

Tx details:
No	Hash		Type							From               To          Amount
%s
`

			var txContent string
			for i := 0; i < len(block.Txs); i++ {
				txContent += fmt.Sprintf("[%d]\t%s\t%d\t%s\t%s\t%d\n",
					i, block.Txs[i].Id, block.Txs[i].Type, block.Txs[i].From, block.Txs[i].To, block.Txs[i].Amount)
			}

			fmt.Printf(blockContent, block.Hash, block.Height,
				utils.TimeToString(block.Time),
				block.Version,
				block.PrevHash,
				block.MerkleRoot,
				block.CreateBy,
				txContent,
			)

			fmt.Println("--------------------------------------------------------")
		}
	}
	hc.responseHandle(rpcResp, handler)

	return nil
}

func (hc *httpClient) genRequest(method string, path string, key, value []string, postData []byte) (*http.Request, error) {
	u, _ := url.Parse(hc.scheme + "://" + hc.serverIP + ":" + hc.serverPort)
	u.Path = path

	q := u.Query()
	for i := 0; i < len(key); i++ {
		q.Add(key[i], value[i])
	}
	u.RawQuery = q.Encode()

	var httpBody io.Reader
	if postData != nil {
		httpBody = bytes.NewBuffer(postData)
	}

	req, err := http.NewRequest(method, u.String(), httpBody)
	if err != nil {
		return nil, fmt.Errorf("generate query failed:%v", err)
	}

	return req, nil
}

func (hc *httpClient) parseResponse(resp *http.Response, data interface{}) (*rpc.HTTPResponse, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed, return:%d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read http body failed:%v", err)
	}

	httpResponse := rpc.ParseHTTPResponse(bodyBytes, data)
	if httpResponse == nil {
		return nil, fmt.Errorf("unmarshal response json failed")
	}

	return httpResponse, nil
}

func (hc *httpClient) responseHandle(httpResponse *rpc.HTTPResponse, f func()) {
	switch httpResponse.Code {
	case rpc.CodeSuccess:
		f()
	case rpc.CodeFailed:
		fmt.Printf("failed: %s\n", httpResponse.Message)
	case rpc.CodeBadRequest:
		fmt.Println("bad request, please check your input")
	default:
		fmt.Printf("response unknown code:%d\n", httpResponse.Code)
	}
}
