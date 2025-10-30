package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	sekaitypes "github.com/KiraCore/sekai/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/saiset-co/sai-storage-mongo/external/adapter"
	"go.uber.org/zap"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/types"
	"github.com/saiset-co/sai-interx-manager/utils"
)

func (g *CosmosGateway) txByHash(hash string) (interface{}, error) {
	req := types.InboundRequest{
		Payload: map[string]interface{}{
			"hash": hash,
		},
	}

	return g.transactions(req)
}

func (g *CosmosGateway) blockById(req types.InboundRequest, blockID string) (interface{}, error) {
	req.Payload["height"] = blockID

	result, err := g.blocks(req)
	if err != nil {
		logger.Logger.Error("[query-block-by-id] Failed to get blocks", zap.Error(err))
		return nil, err
	}

	if len(result.Blocks) < 1 {
		err = errors.New(fmt.Sprintf("Block %s not found", blockID))
		logger.Logger.Error("[query-block-by-id] Block not found", zap.Error(err))
		return nil, err
	}

	return result.Blocks[0], nil
}

func (g *CosmosGateway) txByBlock(req types.InboundRequest, blockID string) (interface{}, error) {
	req.Payload["height"] = blockID
	return g.transactions(req)
}

func (g *CosmosGateway) parseCoinString(input string) (*sdk.Coin, error) {
	denom := ""
	amount := 0

	tokens, err := g.tokens()
	if err != nil {
		logger.Logger.Error("[parse-coin-string] Failed to get tokens", zap.Error(err))
		return nil, err
	}

	for _, poolToken := range tokens {
		if strings.Contains(input, poolToken) {
			pattern := regexp.MustCompile("[^a-zA-Z0-9]+")
			amountStr := strings.ReplaceAll(input, poolToken, "")
			amountStr = pattern.ReplaceAllString(amountStr, "")

			denom = poolToken
			amount, _ = strconv.Atoi(amountStr)
		}
	}
	return &sdk.Coin{
		Denom:  denom,
		Amount: sdk.NewIntFromUint64(uint64(amount)),
	}, nil
}

func (g *CosmosGateway) executionFee(req types.InboundRequest) (interface{}, error) {
	type ExecutionFeeRequest struct {
		Message string `json:"message,omitempty"`
	}

	request := ExecutionFeeRequest{}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return nil, err
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/execution_fee/"+request.Message, nil)
	if err != nil {
		logger.Logger.Error("[execution-fee] Create request failed", zap.Error(err))
		return nil, err
	}

	grpcBytes, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[execution-fee] Serve request failed", zap.Error(err))
		return nil, err
	}

	var result interface{}

	err = json.Unmarshal(grpcBytes, &result)
	if err != nil {
		logger.Logger.Error("[execution-fee] Invalid response format", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (g *CosmosGateway) networkProperties() (interface{}, error) {
	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/network_properties", nil)
	if err != nil {
		logger.Logger.Error("[query-network-properties] Create request failed", zap.Error(err))
		return nil, err
	}

	response, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-network-properties] Serve request failed", zap.Error(err))
		return nil, err
	}

	result, err := utils.QueryNetworkPropertiesFromGrpcResult(response)
	if err != nil {
		logger.Logger.Error("[query-network-properties] Invalid response format", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (g *CosmosGateway) stakingPool(req types.InboundRequest) (interface{}, error) {
	type StakingPoolRequest struct {
		Account string `json:"validatorAddress,omitempty"`
	}

	request := StakingPoolRequest{}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return nil, err
	}

	tokens, err := g.tokens()
	if err != nil {
		logger.Logger.Error("[query-staking-pool] Getting tokens failed", zap.Error(err))
		return nil, err
	}

	validators, err := g.dashboard()
	if err != nil {
		logger.Logger.Error("[query-staking-pool] Getting validators failed", zap.Error(err))
		return nil, err
	}

	if request.Account == "" {
		err = fmt.Errorf("[query-staking-pool] validatorAddress required")
		return nil, err
	}

	valAddr, found := validators.AddrToValidator[request.Account]
	if !found {
		err = fmt.Errorf("[query-staking-pool] validatorAddress not found")
		return nil, err
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/multistaking/v1beta1/staking_pool_delegators/"+valAddr, nil)
	if err != nil {
		logger.Logger.Error("[query-staking-pool] Create request failed", zap.Error(err))
		return nil, err
	}

	response, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-staking-pool] Serve request failed", zap.Error(err))
		return nil, err
	}

	responseResult := types.QueryStakingPoolDelegatorsResponse{}

	err = json.Unmarshal(response, &responseResult)
	if err != nil {
		logger.Logger.Error("[query-staking-pool] Invalid response format", zap.Error(err))
		return nil, err
	}

	newResponse := types.QueryValidatorPoolResult{}
	newResponse.ID = responseResult.Pool.ID
	newResponse.Slashed = utils.ConvertRate(responseResult.Pool.Slashed)
	newResponse.Commission = utils.ConvertRate(responseResult.Pool.Commission)

	newResponse.VotingPower = []sdk.Coin{}
	for _, coinStr := range responseResult.Pool.TotalStakingTokens {
		coin, err := g.parseCoinString(coinStr)
		if err != nil {
			logger.Logger.Error("[query-staking-pool] Coin can not be parsed", zap.Error(err))
			continue
		}
		newResponse.VotingPower = append(newResponse.VotingPower, *coin)
	}

	newResponse.TotalDelegators = int64(len(responseResult.Delegators))
	newResponse.Tokens = []string{}
	newResponse.Tokens = tokens

	return newResponse, nil
}

func (g *CosmosGateway) undelegations(req types.InboundRequest) (interface{}, error) {
	type Undelegation struct {
		ID            int `json:"id,omitempty"`
		ValidatorInfo struct {
			Moniker string `json:"moniker,omitempty"`
			Address string `json:"address,omitempty"`
			ValKey  string `json:"valkey,omitempty"`
			Logo    string `json:"logo,omitempty"`
		} `json:"validator_info"`
		Tokens sdk.Coins `json:"tokens"`
		Expiry string    `json:"expiry,omitempty"`
	}

	type QueryUndelegationsResponse struct {
		Undelegations []Undelegation `json:"undelegations"`
		Pagination    struct {
			Total int `json:"total,string,omitempty"`
		} `json:"pagination,omitempty"`
	}

	type UndelegationsRequest struct {
		Account    string `json:"undelegatorAddress,omitempty"`
		Limit      int    `json:"limit,string,omitempty"`
		Offset     int    `json:"offset,string,omitempty"`
		CountTotal int    `json:"count_total,string,omitempty"`
	}

	request := UndelegationsRequest{}
	response := QueryUndelegationsResponse{}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return nil, err
	}

	if request.Account == "" {
		err = fmt.Errorf("[query-undelegations] validatorAddress required")
		return nil, err
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/multistaking/v1beta1/undelegations", nil)
	if err != nil {
		logger.Logger.Error("[query-undelegations] Create request failed", zap.Error(err))
		return nil, err
	}

	q := gatewayReq.URL.Query()
	q.Add("delegator", request.Account)
	gatewayReq.URL.RawQuery = q.Encode()

	success, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-undelegations] Serve request failed", zap.Error(err))
		return nil, err
	}

	validators, err := g.allValidators()
	if err != nil {
		logger.Logger.Error("[query-undelegations] Getting validators failed", zap.Error(err))
		return nil, err
	}

	result := types.QueryUndelegationsResult{}

	err = json.Unmarshal(success, &result)
	if err != nil {
		logger.Logger.Error("[query-undelegations] Invalid response format", zap.Error(err))
		return nil, err
	}

	for _, undelegation := range result.Undelegations {
		undelegationData := Undelegation{}

		validator := types.QueryValidator{}

		for _, _validator := range validators.Validators {
			if _validator.Valkey == undelegation.ValAddress {
				validator = _validator
			}
		}

		if validator.Address == "" {
			continue
		}

		undelegationData.ID = int(undelegation.ID)

		undelegationData.ValidatorInfo.Address = validator.Address
		undelegationData.ValidatorInfo.Logo = validator.Logo
		undelegationData.ValidatorInfo.Moniker = validator.Moniker
		undelegationData.ValidatorInfo.ValKey = validator.Valkey
		undelegationData.Expiry = undelegation.Expiry

		for _, token := range undelegation.Amount {
			coin, err := g.parseCoinString(token)
			if err != nil {
				logger.Logger.Error("[query-undelegations] Parsing coin failed", zap.Error(err))
				continue
			}
			undelegationData.Tokens = append(undelegationData.Tokens, *coin)
		}

		response.Undelegations = append(response.Undelegations, undelegationData)
	}

	if request.Limit > 0 {
		total := len(response.Undelegations)
		count := int(math.Min(float64(request.Limit), float64(total)))

		if request.CountTotal > 0 {
			response.Pagination.Total = total
		}

		from := int(math.Min(float64(request.Offset), float64(total)))
		to := int(math.Min(float64(request.Offset+count), float64(total)))

		response.Undelegations = response.Undelegations[from:to]
	}

	return response, nil
}

func (g *CosmosGateway) transactions(req types.InboundRequest) (interface{}, error) {
	var result types.TxsResultResponse
	var criteria = map[string]interface{}{}
	var includeConfirmed = true
	var includeFailed = false

	request := types.QueryTxsParams{
		Offset: 0,
		Limit:  sekaitypes.PageIterationLimit - 1,
	}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		logger.Logger.Error("[query-transactions] Invalid request format", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		logger.Logger.Error("[query-transactions] Invalid request format", zap.Error(err))
		return nil, err
	}

	options := &adapter.Options{
		Count: 1,
		Limit: int64(request.Limit),
		Skip:  int64(request.Offset),
	}

	if request.Hash != "" {
		criteria["hash"] = request.Hash
	}

	if request.Height != "" {
		criteria["height"] = request.Height
	} else {
		criteria["_id"] = map[string]interface{}{"$ne": nil}
	}

	if len(request.Types) > 0 {
		criteria["messages"] = map[string]interface{}{
			"typeUrl": map[string]interface{}{
				"$in": request.Types,
			},
		}
	}

	if request.StartDate > 0 {
		criteria["timestamp"] = map[string]interface{}{
			"$gt": request.StartDate,
		}
		//startTime := time.Unix(request.StartDate, 0).UTC()
		//params.Add("tx.mintime", startTime.Format(time.RFC3339))
	}

	if request.EndDate > 0 {
		criteria["timestamp"] = map[string]interface{}{
			"$lt": request.StartDate,
		}
		//endTime := time.Unix(request.EndDate, 0).UTC()
		//params.Add("tx.maxtime", endTime.Format(time.RFC3339))
	}

	if len(request.Statuses) > 0 {
		includeConfirmed = false
		includeFailed = false

		for _, status := range request.Statuses {
			if status == "success" {
				includeConfirmed = true
			} else if status == "failed" {
				includeFailed = true
			}
		}
	}

	if includeConfirmed && !includeFailed {
		criteria["tx_result.code"] = 0
	} else if !includeConfirmed && includeFailed {
		criteria["tx_result.code"] = map[string]interface{}{"$gt": 0}
	}

	txsResponse, err := g.storage.Read("cosmos_txs", criteria, options, []string{})
	if err != nil {
		logger.Logger.Error("[query-transactions] Failed to get transactions", zap.Error(err))
		return nil, err
	}

	result.Transactions = txsResponse.Result
	result.Pagination.Total = txsResponse.Count

	return result, nil
}
