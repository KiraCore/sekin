package gateway

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	sekaitypes "github.com/KiraCore/sekai/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/types"
)

func (g *CosmosGateway) allValidators() (*types.ValidatorsResponse, error) {
	validators := new(types.ValidatorsResponse)
	limit := sekaitypes.PageIterationLimit - 1
	offset := 0

	for {
		gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/staking/validators", nil)
		if err != nil {
			logger.Logger.Error("[query-validators] Create request failed", zap.Error(err))
			return nil, err
		}

		q := gatewayReq.URL.Query()
		q.Add("pagination.offset", strconv.Itoa(offset))
		q.Add("pagination.limit", strconv.Itoa(limit))
		gatewayReq.URL.RawQuery = q.Encode()

		respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
		if err != nil {
			logger.Logger.Error("[query-validators] Serve request failed", zap.Error(err))
			return nil, err
		}

		subResult := new(types.ValidatorsResponse)
		err = json.Unmarshal(respBody, subResult)
		if err != nil {
			logger.Logger.Error("[query-validators] Unmarshal response failed", zap.Error(err))
			return nil, err
		}

		if len(subResult.Validators) == 0 {
			break
		}

		validators.Actors = subResult.Actors
		validators.Validators = append(validators.Validators, subResult.Validators...)
		offset += limit
	}

	return validators, nil
}

func (g *CosmosGateway) supply() (*types.TokenSupplyResponse, error) {
	var tokenSupplyResponse = new(types.TokenSupplyResponse)

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/cosmos/bank/v1beta1/supply", nil)
	if err != nil {
		logger.Logger.Error("[query-supply] Create request failed", zap.Error(err))
		return nil, err
	}

	respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-supply] Serve request failed", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(respBody, tokenSupplyResponse)
	if err != nil {
		logger.Logger.Error("[query-supply] Unmarshal response failed", zap.Error(err))
		return nil, err
	}

	return tokenSupplyResponse, nil
}

func (g *CosmosGateway) tokens() ([]string, error) {
	tokenRatesResponse := types.TokenAliasesGRPCResponse{}
	poolTokens := make([]string, 0)

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/tokens/infos", nil)
	if err != nil {
		logger.Logger.Error("[query-tokens] Create request failed", zap.Error(err))
		return nil, err
	}

	respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-tokens] Serve request failed", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(respBody, &tokenRatesResponse)
	if err != nil {
		logger.Logger.Error("[query-tokens] Unmarshal response failed", zap.Error(err))
		return nil, err
	}

	for _, tokenRate := range tokenRatesResponse.Data {
		poolTokens = append(poolTokens, tokenRate.Data.Denom)
	}

	return poolTokens, nil
}

func (g *CosmosGateway) signingInfos() (*types.ValidatorInfoResponse, error) {
	validatorInfosResponse := new(types.ValidatorInfoResponse)
	limit := sekaitypes.PageIterationLimit - 1
	offset := 0

	for {
		gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/slashing/v1beta1/signing_infos", nil)
		if err != nil {
			logger.Logger.Error("[query-signing-infos] Create request failed", zap.Error(err))
			return nil, err
		}

		q := gatewayReq.URL.Query()
		q.Add("pagination.offset", strconv.Itoa(offset))
		q.Add("pagination.limit", strconv.Itoa(limit))
		gatewayReq.URL.RawQuery = q.Encode()

		respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
		if err != nil {
			logger.Logger.Error("[query-signing-infos] Serve request failed", zap.Error(err))
			return nil, err
		}

		subResult := new(types.ValidatorInfoResponse)
		err = json.Unmarshal(respBody, subResult)
		if err != nil {
			logger.Logger.Error("[query-signing-infos] Unmarshal response failed", zap.Error(err))
			return nil, err
		}

		if len(subResult.ValValidatorInfos) == 0 {
			break
		}

		validatorInfosResponse.ValValidatorInfos = append(validatorInfosResponse.ValValidatorInfos, subResult.ValValidatorInfos...)
		offset += limit
	}

	return validatorInfosResponse, nil
}

func (g *CosmosGateway) validatorsPool() (*types.AllPools, error) {
	type ValidatorPoolsResponse struct {
		Pools []types.ValidatorPool `json:"pools,omitempty"`
	}

	pools := ValidatorPoolsResponse{}

	allPools := &types.AllPools{
		ValToPool: make(map[string]types.ValidatorPool),
		IdToPool:  make(map[int64]types.ValidatorPool),
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/multistaking/v1beta1/staking_pools", nil)
	if err != nil {
		logger.Logger.Error("[query-validators-pool] Create request failed", zap.Error(err))
		return nil, err
	}

	respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-validators-pool] Serve request failed", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(respBody, &pools)
	if err != nil {
		logger.Logger.Error("[query-validators-pool] Unmarshal response failed", zap.Error(err))
		return nil, err
	}

	for _, pool := range pools.Pools {
		allPools.ValToPool[pool.Validator] = pool
		allPools.IdToPool[pool.ID] = pool
	}

	return allPools, nil
}

func (g *CosmosGateway) dashboard() (*types.AllValidators, error) {
	allValidators := &types.AllValidators{
		AddrToValidator: make(map[string]string),
		PoolToValidator: make(map[int64]types.QueryValidator),
		PoolTokens:      make([]string, 0),
	}

	validatorsData, err := g.allValidators()
	if err != nil {
		logger.Logger.Error("[query-dashboard] validators", zap.Error(err))
		return nil, err
	}

	tokens, err := g.tokens()
	if err != nil {
		logger.Logger.Error("[query-dashboard] failed to get tokens", zap.Error(err))
		return nil, err
	}

	signingInfos, err := g.signingInfos()
	if err != nil {
		logger.Logger.Error("[query-dashboard] failed to get signingInfos", zap.Error(err))
		return nil, err
	}

	validatorsPool, err := g.validatorsPool()
	if err != nil {
		logger.Logger.Error("[query-dashboard] failed to get validatorsPool", zap.Error(err))
		return nil, err
	}

	for index, validator := range validatorsData.Validators {
		pubkeyHexString := validator.Pubkey[14 : len(validator.Pubkey)-1]
		bytes, _ := hex.DecodeString(pubkeyHexString)
		pubkey := ed25519.PubKey{
			Key: bytes,
		}
		address := sdk.ConsAddress(pubkey.Address()).String()
		allValidators.AddrToValidator[validator.Address] = validator.Valkey

		var valSigningInfo types.ValidatorSigningInfo

		for _, signingInfo := range signingInfos.ValValidatorInfos {
			if signingInfo.Address == address {
				valSigningInfo = signingInfo
				break
			}
		}

		for _, record := range validatorsData.Validators[index].Identity {
			if record.Key == "logo" || record.Key == "avatar" {
				validatorsData.Validators[index].Logo = record.Value
			} else if record.Key == "description" {
				validatorsData.Validators[index].Description = record.Value
			} else if record.Key == "website" {
				validatorsData.Validators[index].Website = record.Value
			} else if record.Key == "social" {
				validatorsData.Validators[index].Social = record.Value
			} else if record.Key == "contact" {
				validatorsData.Validators[index].Contact = record.Value
			} else if record.Key == "validator_node_id" {
				validatorsData.Validators[index].Validator_node_id = record.Value
			} else if record.Key == "sentry_node_id" {
				validatorsData.Validators[index].Sentry_node_id = record.Value
			}
		}

		validatorsData.Validators[index].Identity = nil
		validatorsData.Validators[index].StartHeight = valSigningInfo.StartHeight
		validatorsData.Validators[index].InactiveUntil = valSigningInfo.InactiveUntil
		validatorsData.Validators[index].Mischance = valSigningInfo.Mischance
		validatorsData.Validators[index].MischanceConfidence = valSigningInfo.MischanceConfidence
		validatorsData.Validators[index].LastPresentBlock = valSigningInfo.LastPresentBlock
		validatorsData.Validators[index].MissedBlocksCounter = valSigningInfo.MissedBlocksCounter
		validatorsData.Validators[index].ProducedBlocksCounter = valSigningInfo.ProducedBlocksCounter

		pool, found := validatorsPool.ValToPool[validator.Valkey]
		if found {
			validatorsData.Validators[index].StakingPoolId = pool.ID
			if pool.Enabled {
				validatorsData.Validators[index].StakingPoolStatus = "ENABLED"
			} else {
				validatorsData.Validators[index].StakingPoolStatus = "DISABLED"
			}

			allValidators.PoolToValidator[validatorsData.Validators[index].StakingPoolId] = validatorsData.Validators[index]
		}
	}

	sort.Sort(types.QueryValidators(validatorsData.Validators))
	for index := range validatorsData.Validators {
		validatorsData.Validators[index].Top = index + 1
	}

	allValidators.PoolTokens = tokens
	allValidators.Validators = validatorsData.Validators
	allValidators.Waiting = make([]string, 0)

	for _, actor := range validatorsData.Actors {
		isWaiting := true
		for _, validator := range validatorsData.Validators {
			if validator.Address == actor {
				isWaiting = false
				break
			}
		}

		if isWaiting {
			allValidators.Waiting = append(allValidators.Waiting, actor)
		}
	}

	allValidators.Status.TotalValidators = len(validatorsData.Validators)
	allValidators.Status.WaitingValidators = len(allValidators.Waiting)
	allValidators.Status.ActiveValidators = 0
	allValidators.Status.PausedValidators = 0
	allValidators.Status.InactiveValidators = 0
	allValidators.Status.JailedValidators = 0

	for _, validator := range validatorsData.Validators {
		if validator.Status == Active {
			allValidators.Status.ActiveValidators++
		}
		if validator.Status == Inactive {
			allValidators.Status.InactiveValidators++
		}
		if validator.Status == Paused {
			allValidators.Status.PausedValidators++
		}
		if validator.Status == Jailed {
			allValidators.Status.JailedValidators++
		}
	}

	return allValidators, nil
}

func (g *CosmosGateway) txs(req types.InboundRequest) (interface{}, error) {
	type PostTxReq struct {
		Tx   string `json:"tx"`
		Mode string `json:"mode"`
	}

	var request = new(PostTxReq)

	reqParams, err := json.Marshal(req.Payload)
	if err != nil {
		logger.Logger.Error("txs", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(reqParams, &request)
	if err != nil {
		logger.Logger.Error("txs", zap.Error(err))
		return nil, err
	}

	if request.Mode != "" {
		if allowed, ok := g.config.TxModes[request.Mode]; !ok || !allowed {
			err = errors.New("[post-transaction] Invalid transaction mode")
			return nil, err
		}
	}

	_url := "/broadcast_tx_sync"

	switch request.Mode {
	case "block":
		_url = "/broadcast_tx_commit"
	case "async":
		_url = "/broadcast_tx_async"
	}

	txBytes, err := base64.StdEncoding.DecodeString(request.Tx)
	if err != nil {
		logger.Logger.Error("txs", zap.Error(err))
		return nil, err
	}

	return g.makeTendermintRPCRequest(g.context.Context, _url, fmt.Sprintf("tx=0x%X", txBytes))
}

func (g *CosmosGateway) validators(req types.InboundRequest) (*types.ValidatorsResponse, error) {
	validatorsResponse, err := g.allValidators()
	if err != nil {
		logger.Logger.Error("[query-validators] allValidators failed", zap.Error(err))
		return nil, err
	}

	return g.filterAndPaginateValidators(validatorsResponse, req.Payload)
}

func (g *CosmosGateway) account(address string) (*types.AccountResponse, error) {
	accountReq := types.InboundRequest{
		Method:  "GET",
		Path:    "/cosmos/auth/v1beta1/accounts/" + address,
		Payload: map[string]interface{}{},
	}

	accountInfoBytes, err := g.proxy(accountReq)
	if err != nil {
		logger.Logger.Error("[query-account] Failed getting account info", zap.Error(err))
		return nil, err
	}

	var accountResponse = new(types.AccountResponse)

	err = json.Unmarshal(accountInfoBytes, accountResponse)
	if err != nil {
		logger.Logger.Error("[query-account] Invalid response format", zap.Error(err))
		return nil, err
	}

	return accountResponse, nil
}
