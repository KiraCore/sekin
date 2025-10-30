package gateway

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/saiset-co/sai-storage-mongo/external/adapter"
	"math"
	"net/http"
	"strconv"
	"strings"

	sekaitypes "github.com/KiraCore/sekai/types"
	tmjson "github.com/cometbft/cometbft/libs/json"
	types2 "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cast"
	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/types"
	"github.com/saiset-co/sai-interx-manager/utils"
)

func (g *CosmosGateway) statusAPI() (interface{}, error) {
	result := types.InterxStatus{
		ID: cast.ToString(g.context.GetConfig("p2p.id", "")),
	}

	genesis, err := g.genesis()
	if err != nil {
		logger.Logger.Error("[query-status] Failed to query genesis", zap.Error(err))
		return nil, err
	}

	result.InterxInfo.ChainID = genesis.GenesisDoc.ChainID
	result.InterxInfo.GenesisChecksum = fmt.Sprintf("%x", sha256.Sum256(genesis.GenesisData))

	sentryStatus, err := g.status()
	if err != nil {
		logger.Logger.Error("[query-status] Failed to query status", zap.Error(err))
		return nil, err
	}

	result.NodeInfo = sentryStatus.NodeInfo
	result.SyncInfo = sentryStatus.SyncInfo
	result.ValidatorInfo = sentryStatus.ValidatorInfo
	result.InterxInfo.Moniker = sentryStatus.NodeInfo.Moniker
	result.InterxInfo.LatestBlockHeight = sentryStatus.SyncInfo.LatestBlockHeight
	result.InterxInfo.CatchingUp = sentryStatus.SyncInfo.CatchingUp

	//result.InterxInfo.Node = config.Config.Node
	//result.InterxInfo.KiraAddr = g.address
	result.InterxInfo.KiraPubKey = g.PubKey.String()
	result.InterxInfo.FaucetAddr = g.PubKey.Address().String()
	//result.InterxInfo.InterxVersion = config.Config.InterxVersion
	//result.InterxInfo.SekaiVersion = config.Config.SekaiVersion

	return result, nil
}

func (g *CosmosGateway) status() (*types.KiraStatus, error) {
	success, err := g.makeTendermintRPCRequest(g.context.Context, "/status", "")
	if err != nil {
		logger.Logger.Error("[kira-status] Invalid response format", zap.Error(err))
		return nil, err
	}

	result := new(types.KiraStatus)

	byteData, err := json.Marshal(success)
	if err != nil {
		logger.Logger.Error("[kira-status] Invalid response format", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(byteData, result)
	if err != nil {
		logger.Logger.Error("[kira-status] Invalid response format", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (g *CosmosGateway) genesisChunked(chunk int) (*types.GenesisChunkedResponse, error) {
	data, _ := g.makeTendermintRPCRequest(g.context.Context, "/genesis_chunked", fmt.Sprintf("chunk=%d", chunk))

	genesis := new(types.GenesisChunkedResponse)
	byteData, err := json.Marshal(data)
	if err != nil {
		logger.Logger.Error("[genesis-chunked] Invalid response format", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(byteData, genesis)
	if err != nil {
		logger.Logger.Error("[genesis-chunked] Invalid response format", zap.Error(err))
		return nil, err
	}

	return genesis, nil
}

func (g *CosmosGateway) genesis() (*types.GenesisInfo, error) {
	gInfo := new(types.GenesisInfo)
	gInfo.GenesisDoc = new(types2.GenesisDoc)

	genesisData, err := g.genesisChunked(0)
	if err != nil {
		logger.Logger.Error("[query-genesis] Failed to get genesis part", zap.Error(err))
		return nil, err
	}

	total, err := strconv.Atoi(genesisData.Total)
	if err != nil {
		logger.Logger.Error("[query-genesis] Invalid response format", zap.Error(err))
		return nil, err
	}

	if total > 1 {
		for i := 1; i < total; i++ {
			nextData, err := g.genesisChunked(i)
			if err != nil {
				logger.Logger.Error("[query-genesis] Failed to get genesis part", zap.Error(err))
				return nil, err
			}

			genesisData.Data = append(genesisData.Data, nextData.Data...)
		}
	}

	gInfo.GenesisData = genesisData.Data

	err = tmjson.Unmarshal(genesisData.Data, gInfo.GenesisDoc)
	if err != nil {
		logger.Logger.Error("[query-genesis] Invalid response format", zap.Error(err))
		return nil, err
	}

	err = gInfo.GenesisDoc.ValidateAndComplete()
	if err != nil {
		logger.Logger.Error("[query-genesis] Genesis not valid", zap.Error(err))
		return nil, err
	}

	return gInfo, nil
}

func (g *CosmosGateway) blocks(req types.InboundRequest) (*types.BlocksResultResponse, error) {
	var result types.BlocksResultResponse
	var criteria = map[string]interface{}{}

	type BlocksRequest struct {
		Height string `json:"height,omitempty"`
		Limit  int    `json:"limit,string,omitempty"`
		Offset int    `json:"offset,string,omitempty"`
		HasTxs int    `json:"has_txs,string,omitempty"`
		Order  string `json:"order_by,omitempty"`
	}

	request := BlocksRequest{
		Limit:  sekaitypes.PageIterationLimit - 1,
		Offset: 0,
		HasTxs: 0,
		Order:  "asc",
	}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		logger.Logger.Error("[query-blocks] Invalid request format", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		logger.Logger.Error("[query-blocks] Invalid request format", zap.Error(err))
		return nil, err
	}

	options := &adapter.Options{
		Count: 1,
		Limit: int64(request.Limit),
		Skip:  int64(request.Offset),
	}

	if request.Height != "" {
		criteria["block.header.height"] = request.Height
	}

	if request.HasTxs == 1 {
		criteria["block.data.txs"] = map[string]interface{}{
			"$ne": []interface{}{},
		}
	}

	blocksResponse, err := g.storage.Read("cosmos_blocks", criteria, options, []string{})
	if err != nil {
		logger.Logger.Error("[query-blocks] Failed to get blocks", zap.Error(err))
		return nil, err
	}

	result.Blocks = blocksResponse.Result
	result.Pagination.Total = blocksResponse.Count

	return &result, nil
}

func (g *CosmosGateway) balances(req types.InboundRequest, accountID string) ([]sdk.Coin, error) {
	type BalancesRequest struct {
		Limit      int `json:"limit,string,omitempty"`
		Offset     int `json:"offset,string,omitempty"`
		CountTotal int `json:"count_total,string,omitempty"`
	}

	request := BalancesRequest{
		Limit:      sekaitypes.PageIterationLimit - 1,
		Offset:     0,
		CountTotal: 0,
	}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return nil, err
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/cosmos/bank/v1beta1/balances/"+accountID, nil)
	if err != nil {
		logger.Logger.Error("[query-balances] Create request failed", zap.Error(err))
		return nil, err
	}

	q := gatewayReq.URL.Query()
	q.Add("pagination.offset", strconv.Itoa(request.Offset))
	q.Add("pagination.limit", strconv.Itoa(request.Limit))
	if request.CountTotal > 0 {
		q.Add("pagination.count_total", "true")
	}
	gatewayReq.URL.RawQuery = q.Encode()

	grpcBytes, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-balances] Serve request failed", zap.Error(err))
		return nil, err
	}

	var result types.BalancesResponse

	err = json.Unmarshal(grpcBytes, &result)
	if err != nil {
		logger.Logger.Error("[query-balances] Invalid response format", zap.Error(err))
		return nil, err
	}

	return result.Balances, nil
}

func (g *CosmosGateway) delegations(req types.InboundRequest) (interface{}, error) {
	var response = new(types.QueryDelegationsResult)

	type DelegationsRequest struct {
		Account    string `json:"delegatorAddress,omitempty"`
		Limit      int    `json:"limit,string,omitempty"`
		Offset     int    `json:"offset,string,omitempty"`
		CountTotal int    `json:"count_total,string,omitempty"`
	}

	request := DelegationsRequest{
		Account:    "",
		Limit:      sekaitypes.PageIterationLimit - 1,
		Offset:     0,
		CountTotal: 0,
	}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return nil, err
	}

	if request.Account != "" {
		req.Path = "/cosmos/bank/v1beta1/balances/" + request.Account
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", req.Path, nil)
	if err != nil {
		logger.Logger.Error("[query-delegations] Create request failed", zap.Error(err))
		return nil, err
	}

	success, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-delegations] Serve request failed", zap.Error(err))
		return nil, err
	}

	allPools, err := g.validatorsPool()
	if err != nil {
		logger.Logger.Error("[query-delegations] Error getting validators pool", zap.Error(err))
		return nil, err
	}

	tokens, err := g.tokens()
	if err != nil {
		logger.Logger.Error("[query-delegations] Error getting tokens", zap.Error(err))
		return nil, err
	}

	validators, err := g.dashboard()
	if err != nil {
		logger.Logger.Error("[query-delegations] Error getting validators", zap.Error(err))
		return nil, err
	}

	result := types.QueryBalancesResponse{}

	err = json.Unmarshal(success, &result)
	if err != nil {
		logger.Logger.Error("[query-delegations] Invalid response format", zap.Error(err))
		return nil, err
	}

	for _, balance := range result.Balances {
		delegation := types.Delegation{}
		denomParts := strings.Split(balance.Denom, "/")
		// if denom format is v{N}/XXX,
		if len(denomParts) == 2 && denomParts[0][0] == 'v' {
			// fetch pool id from denom
			poolID, err := strconv.Atoi(denomParts[0][1:])
			if err != nil {
				continue
			}

			// get pool data from id
			pool, found := allPools.IdToPool[int64(poolID)]
			if !found {
				continue
			}
			// fill up PoolInfo
			delegation.PoolInfo.ID = pool.ID
			delegation.PoolInfo.Commission = utils.ConvertRate(pool.Commission)
			if pool.Enabled {
				delegation.PoolInfo.Status = "ENABLED"
			} else {
				delegation.PoolInfo.Status = "DISABLED"
			}
			delegation.PoolInfo.Tokens = tokens

			// fill up ValidatorInfo
			validator, found := validators.PoolToValidator[pool.ID]
			if found {
				delegation.ValidatorInfo.Address = validator.Address
				delegation.ValidatorInfo.ValKey = validator.Valkey
				delegation.ValidatorInfo.Moniker = validator.Moniker
				delegation.ValidatorInfo.Website = validator.Website
				delegation.ValidatorInfo.Logo = validator.Logo
			}
			response.Delegations = append(response.Delegations, delegation)
		}
	}

	if request.Limit > 0 {
		// apply pagination
		total := len(response.Delegations)
		count := int(math.Min(float64(request.Limit), float64(total)))

		if request.CountTotal > 0 {
			response.Pagination.Total = total
		}

		from := int(math.Min(float64(request.Offset), float64(total)))
		to := int(math.Min(float64(request.Offset+count), float64(total)))

		response.Delegations = response.Delegations[from:to]
	}

	return response, nil
}

func (g *CosmosGateway) identityRecords(address string) (interface{}, error) {
	accAddr, _ := sdk.AccAddressFromBech32(address)

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/identity_records/"+base64.URLEncoding.EncodeToString(accAddr.Bytes()), nil)
	if err != nil {
		logger.Logger.Error("[query-identity-records] Create request failed", zap.Error(err))
		return nil, err
	}

	grpcBytes, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-identity-records] Serve request failed", zap.Error(err))
		return nil, err
	}

	var result interface{}

	err = json.Unmarshal(grpcBytes, &result)
	if err != nil {
		logger.Logger.Error("[query-identity-records] Invalid response format", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (g *CosmosGateway) identityVerifyRequestsByApprover(req types.InboundRequest, approver string) (interface{}, error) {
	type IdentityVerifyRequestsByApproverRequest struct {
		Key        int `json:"key,string,omitempty"`
		Limit      int `json:"limit,string,omitempty"`
		Offset     int `json:"offset,string,omitempty"`
		CountTotal int `json:"count_total,string,omitempty"`
	}

	request := IdentityVerifyRequestsByApproverRequest{
		Key:        0,
		Limit:      sekaitypes.PageIterationLimit - 1,
		Offset:     0,
		CountTotal: 0,
	}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return nil, err
	}

	accAddr, _ := sdk.AccAddressFromBech32(approver)
	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/identity_verify_requests_by_approver/"+base64.URLEncoding.EncodeToString(accAddr.Bytes()), nil)
	if err != nil {
		logger.Logger.Error("[query-identity-record-verify-requests-by-approver] Create request failed", zap.Error(err))
		return nil, err
	}

	q := gatewayReq.URL.Query()
	q.Add("pagination.offset", strconv.Itoa(request.Offset))
	q.Add("pagination.limit", strconv.Itoa(request.Limit))
	if request.Key > 0 {
		q.Add("pagination.key", strconv.Itoa(request.Key))
	}
	if request.CountTotal > 0 {
		q.Add("pagination.count_total", "true")
	}
	gatewayReq.URL.RawQuery = q.Encode()
	response, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-identity-record-verify-requests-by-approver] Serve request failed", zap.Error(err))
		return nil, err
	}

	res := types.IdVerifyResponse{}

	err = json.Unmarshal(response, &res)
	if err != nil {
		logger.Logger.Error("[query-identity-record-verify-requests-by-approver] Invalid response format", zap.Error(err))
		return nil, err
	}

	for idx, record := range res.VerifyRecords {
		coin, err := g.parseCoinString(record.Tip)
		if err != nil {
			logger.Logger.Error("[query-identity-record-verify-requests-by-approver] Coin can not be parsed", zap.Error(err))
			return nil, err
		}

		res.VerifyRecords[idx].Tip = coin.String()
	}

	return res, nil
}

func (g *CosmosGateway) identityVerifyRequestsByRequester(req types.InboundRequest, requester string) (interface{}, error) {
	type IdentityVerifyRequestsByRequesterRequest struct {
		Key        int `json:"key,string,omitempty"`
		Limit      int `json:"limit,string,omitempty"`
		Offset     int `json:"offset,string,omitempty"`
		CountTotal int `json:"count_total,string,omitempty"`
	}

	request := IdentityVerifyRequestsByRequesterRequest{
		Key:        0,
		Limit:      sekaitypes.PageIterationLimit - 1,
		Offset:     0,
		CountTotal: 0,
	}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return nil, err
	}

	accAddr, _ := sdk.AccAddressFromBech32(requester)
	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/identity_verify_requests_by_requester/"+base64.URLEncoding.EncodeToString(accAddr.Bytes()), nil)
	if err != nil {
		logger.Logger.Error("[query-identity-record-verify-requests-by-requester] Create request failed", zap.Error(err))
		return nil, err
	}

	q := gatewayReq.URL.Query()
	q.Add("pagination.offset", strconv.Itoa(request.Offset))
	q.Add("pagination.limit", strconv.Itoa(request.Limit))
	if request.Key > 0 {
		q.Add("pagination.key", strconv.Itoa(request.Key))
	}
	if request.CountTotal > 0 {
		q.Add("pagination.count_total", "true")
	}

	gatewayReq.URL.RawQuery = q.Encode()
	response, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-identity-record-verify-requests-by-requester] Serve request failed", zap.Error(err))
		return nil, err
	}

	res := types.IdVerifyResponse{}
	err = json.Unmarshal(response, &res)
	if err != nil {
		logger.Logger.Error("[query-identity-record-verify-requests-by-requester] Invalid response format", zap.Error(err))
		return nil, err
	}

	for idx, record := range res.VerifyRecords {
		coin, err := g.parseCoinString(record.Tip)
		if err != nil {
			logger.Logger.Error("[query-identity-record-verify-requests-by-approver] Coin can not be parsed", zap.Error(err))
			continue
		}

		res.VerifyRecords[idx].Tip = coin.String()
	}

	return res, nil
}
