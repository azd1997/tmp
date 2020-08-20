package rpc

import (
	"fmt"
	"github.com/azd1997/ecoin/protocol/view"
	"net/http"
	"strconv"
	"strings"
)

const (
	blockPath = "/block"
)

var (
	// BlocksV1Path /v1/block
	BlocksV1Path = version1Path + blockPath

	// QueryBlockViaRangeV1Path GET /v1/block/query-via-range
	QueryBlockViaRangeV1Path = BlocksV1Path + "/query-via-range"

	// QueryBlockViaHashV1Path GET /v1/block/query-via-hash
	QueryBlockViaHashV1Path = BlocksV1Path + "/query-via-hash"

	blockHandler = HTTPHandlers{
		{QueryBlockViaRangeV1Path, getBlockViaRange},
		{QueryBlockViaHashV1Path, getBlockViaHash},
	}
)

type GetBlocksResponse struct {
	Data []*view.BlockJSON `json:"data"`
}



func responseBlocks(w http.ResponseWriter, blocks []*view.BlockInfo) {
	if len(blocks) == 0 {
		failedResponse("not found", w)
		return
	}

	if len(blocks) == 1 && blocks[0] == nil {
		failedResponse("not found", w)
		return
	}

	resp := &GetBlocksResponse{}
	for _, info := range blocks {
		blockJSON := &view.BlockJSON{}
		blockJSON.FromBlockInfo(info)
		resp.Data = append(resp.Data, blockJSON)
	}

	successWithDataResponse(resp, w)
}

/*
GET /v1/block/query-via-range?range=...

three kinds of range format:
1. from 1 to 100: 1-100
2. the specified height: 128 or 1,50,200 (separate with ,)
3. the latest block: -1
*/

type getBlockViaRangeResponse = GetBlocksResponse

// 通过范围查询区块的handler
func getBlockViaRange(w http.ResponseWriter, r *http.Request) {
	param, ok := r.URL.Query()[GetRangeParam]
	if !ok {
		badRequestResponse(w)
		return
	}

	// ..?range=xx
	height, err := strconv.ParseInt(param[0], 10, 64)
	if err == nil {
		if height == -1 {
			result := globalSvr.en.QueryLatestBlock()
			responseBlocks(w, []*view.BlockInfo{result})
			return
		}

		if height <= 0 {
			badRequestResponse(w)
			return
		}

		result := globalSvr.en.QueryBlockViaHeights([]uint64{uint64(height)})
		responseBlocks(w, result)
		return
	}

	// ..?range=xx-yy
	if strings.Contains(param[0], "-") {
		var begin, end uint64
		n, err := fmt.Sscanf(param[0], "%d-%d", &begin, &end)
		if err != nil || n != 2 || begin >= end {
			badRequestResponse(w)
			return
		}

		result := globalSvr.en.QueryBlockViaRange(begin, end)
		responseBlocks(w, result)
		return
	}

	// ..?range=xx,yy,zz
	if strings.Contains(param[0], ",") {
		heightsStr := strings.Split(param[0], ",")
		var heights []uint64
		for _, str := range heightsStr {
			height, err := strconv.ParseUint(str, 10, 64)
			if err != nil || height == 0 {
				badRequestResponse(w)
				return
			}
			heights = append(heights, height)
		}

		result := globalSvr.en.QueryBlockViaHeights(heights)
		responseBlocks(w, result)
		return
	}

	badRequestResponse(w)
}

/*
GET /v1/block/query-via-hash?hash=...

format: xxx or xxx,xxx,xxx (seperate with ,)
*/

type getBlockViaHashResponse = GetBlocksResponse

func getBlockViaHash(w http.ResponseWriter, r *http.Request) {
	param, ok := r.URL.Query()[GetHashParam]
	if !ok {
		badRequestResponse(w)
		return
	}

	var queryHash []string
	if strings.Contains(param[0], ",") {
		queryHash = strings.Split(param[0], ",")
	} else {
		queryHash = []string{param[0]}
	}

	result := globalSvr.en.QueryBlockViaHash(queryHash)
	responseBlocks(w, result)
	return
}
