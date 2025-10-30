package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"strconv"
	"time"

	sekaitypes "github.com/KiraCore/sekai/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/types"
	"github.com/saiset-co/sai-interx-manager/utils"
	"github.com/saiset-co/sai-storage-mongo/external/adapter"
)

func (g *CosmosGateway) tokenRates() (interface{}, error) {
	tokenAliasGRPCResponse := types.TokenAliasesGRPCResponse{}

	type TokenRatesResponse struct {
		Data []types.TokenAlias `json:"data"`
	}
	result := TokenRatesResponse{}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/tokens/infos", nil)
	if err != nil {
		logger.Logger.Error("[query-token-rates] Create request failed", zap.Error(err))
		return nil, err
	}

	respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-token-rates] Serve request failed", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(respBody, &tokenAliasGRPCResponse)
	if err != nil {
		logger.Logger.Error("[query-token-rates] Invalid response format", zap.Error(err))
		return nil, err
	}

	for index, tokenRate := range tokenAliasGRPCResponse.Data {
		tokenAliasGRPCResponse.Data[index].Data.FeeRate = utils.ConvertRate(tokenRate.Data.FeeRate)
		tokenAliasGRPCResponse.Data[index].Data.StakeCap = utils.ConvertRate(tokenRate.Data.StakeCap)
		tokenAliasGRPCResponse.Data[index].Data.StakeMin = utils.ConvertRate(tokenRate.Data.StakeMin)

		result.Data = append(result.Data, tokenAliasGRPCResponse.Data[index].Data)
	}

	return result, nil
}

func (g *CosmosGateway) customPrefixes() (*types.CustomPrefixesResponse, error) {
	var customPrefixesResponse = new(types.CustomPrefixesResponse)

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/custom_prefixes", nil)
	if err != nil {
		logger.Logger.Error("[query-custom-prefixes] Create request failed", zap.Error(err))
		return nil, err
	}

	respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-custom-prefixes] Serve request failed", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(respBody, &customPrefixesResponse)
	if err != nil {
		logger.Logger.Error("[query-custom-prefixes] Invalid response format", zap.Error(err))
		return nil, err
	}

	return customPrefixesResponse, nil
}

func (g *CosmosGateway) tokenAliases(req types.InboundRequest) (interface{}, error) {
	tokenAliasGRPCResponse := types.TokenAliasesGRPCResponse{}
	tokenAliasResponse := types.TokenAliasesResponse{}

	type TokenAliasRequest struct {
		Tokens     []string `json:"tokens,omitempty"`
		Limit      int      `json:"limit,string,omitempty"`
		Offset     int      `json:"offset,string,omitempty"`
		CountTotal int      `json:"count_total,string,omitempty"`
	}

	request := TokenAliasRequest{
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

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/tokens/infos", nil)
	if err != nil {
		logger.Logger.Error("[query-token-aliases] Create request failed", zap.Error(err))
		return nil, err
	}

	respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-token-aliases] Serve request failed", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(respBody, &tokenAliasGRPCResponse)
	if err != nil {
		logger.Logger.Error("[query-token-aliases] Invalid response format", zap.Error(err))
		return nil, err
	}

	prefixes, err := g.customPrefixes()
	if err != nil {
		logger.Logger.Error("[query-token-aliases] Failed to get custom prefixes", zap.Error(err))
		return nil, err
	}

	for _, alias := range tokenAliasGRPCResponse.Data {
		tokenAliasResponse.Data = append(tokenAliasResponse.Data, alias.Data)
	}

	if request.Limit > 0 {
		total := len(tokenAliasResponse.Data)
		count := int(math.Min(float64(request.Limit), float64(total)))

		if request.CountTotal > 0 {
			tokenAliasResponse.Pagination.Total = total
		}

		from := int(math.Min(float64(request.Offset), float64(total)))
		to := int(math.Min(float64(request.Offset+count), float64(total)))

		tokenAliasResponse.Data = tokenAliasResponse.Data[from:to]
	}

	tokenAliasResponse.Bech32Prefix = prefixes.Bech32Prefix
	tokenAliasResponse.DefaultDenom = prefixes.DefaultDenom

	return tokenAliasResponse, nil
}

func (g *CosmosGateway) proposalsCount() (int, error) {
	var totalCount = 0
	var response struct {
		Pagination struct {
			Total string `json:"total"`
		} `json:"pagination"`
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/proposals", nil)
	if err != nil {
		logger.Logger.Error("[query-proposals-count] Create request failed", zap.Error(err))
		return totalCount, err
	}

	q := gatewayReq.URL.Query()
	q.Add("pagination.offset", "0")
	q.Add("pagination.limit", "1")
	q.Add("pagination.count_total", "true")
	gatewayReq.URL.RawQuery = q.Encode()

	success, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[[query-proposals-count] Serve request failed", zap.Error(err))
		return totalCount, err
	}

	if err := json.Unmarshal(success, &response); err != nil {
		logger.Logger.Error("[query-proposals-count] Invalid response format", zap.Error(err))
		return totalCount, err
	}

	if response.Pagination.Total != "" {
		totalCount, _ = strconv.Atoi(response.Pagination.Total)
	}

	return totalCount, nil
}

func (g *CosmosGateway) getProposals(req types.InboundRequest) (interface{}, error) {
	proposals := new(types.ProposalsResponse)
	limit := sekaitypes.PageIterationLimit - 1
	offset := 0

	for {
		gatewayReq, err := http.NewRequestWithContext(g.context.Context, "GET", "/kira/gov/proposals", nil)
		if err != nil {
			logger.Logger.Error("[query-proposals] Create request failed", zap.Error(err))
			return nil, err
		}

		q := gatewayReq.URL.Query()
		q.Add("pagination.offset", strconv.Itoa(offset))
		q.Add("pagination.limit", strconv.Itoa(limit))
		gatewayReq.URL.RawQuery = q.Encode()

		respBody, err := g.grpcProxy.ServeGRPC(gatewayReq)
		if err != nil {
			logger.Logger.Error("[query-proposals] Serve request failed", zap.Error(err))
			return nil, err
		}

		subResult := new(types.ProposalsResponse)
		err = json.Unmarshal(respBody, subResult)
		if err != nil {
			logger.Logger.Error("[query-proposals] Invalid response format", zap.Error(err))
			return nil, err
		}

		if len(subResult.Proposals) == 0 {
			break
		}

		if afterProposalID, afterOk := req.Payload["afterProposalId"].(string); afterOk {
			for _, proposal := range proposals.Proposals {
				if proposal.ProposalID > afterProposalID {
					proposals.Proposals = append(proposals.Proposals, proposal)
				}
			}
		} else {
			proposals.Proposals = append(proposals.Proposals, subResult.Proposals...)
		}

		offset += limit
	}

	return proposals, nil
}

func (g *CosmosGateway) proposals(req types.InboundRequest) (interface{}, error) {
	var lastId = "0"

	var proposalsResponse = types.ProposalsResponse{
		Pagination: types.Pagination{
			Total: 0,
		},
	}

	var criteria = map[string]interface{}{
		"internal_id": map[string]interface{}{
			"$ne": nil,
		},
	}

	var proposals []types.Proposal

	var sortBy = map[string]interface{}{
		"proposalId": -1,
	}

	cachedTotal, err := g.storage.Read("proposals_cache", criteria, &adapter.Options{Limit: 1, Count: 1, Sort: sortBy}, []string{})
	if err != nil {
		logger.Logger.Error("[query-proposals] Failed to get cached proposals count", zap.Error(err))
		return proposalsResponse, err
	}

	count, err := g.proposalsCount()
	if err != nil {
		logger.Logger.Error("[query-proposals] Failed to count proposals", zap.Error(err))
		return proposalsResponse, err
	}

	if count < 0 {
		return proposalsResponse, nil
	}

	if len(cachedTotal.Result) > 0 {
		if lastIdI, ok := cachedTotal.Result[0]["proposalId"]; ok {
			if lastIdR, ok := lastIdI.(string); ok {
				lastId = lastIdR
			}
		}
	}

	if count > cachedTotal.Count {
		req.Payload["afterProposalId"] = lastId
		newProposals, err := g.getProposals(req)
		if err != nil {
			logger.Logger.Error("[query-proposals] Failed to get new proposals", zap.Error(err))
			return proposalsResponse, err
		}

		_, err = g.storage.Create("proposals_cache", newProposals)
		if err != nil {
			logger.Logger.Error("[query-proposals] Failed to save proposals cache", zap.Error(err))
			return proposalsResponse, err
		}
	}

	request := types.ProposalsRequest{
		Limit:  sekaitypes.PageIterationLimit - 1,
		Offset: 0,
	}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		return proposalsResponse, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		return proposalsResponse, err
	}

	options := &adapter.Options{
		Limit: request.Limit,
		Skip:  request.Offset,
		Count: request.CountTotal,
	}

	if request.SortBy == "dateASC" {
		options.Sort = map[string]interface{}{"timestamp": 1}
	} else {
		options.Sort = map[string]interface{}{"timestamp": -1}
	}

	if request.Proposer != "" {
		criteria["proposer"] = request.Proposer
	}

	if request.DateStart > 0 || request.DateEnd > 0 {
		timeQuery := make(map[string]interface{})
		if request.DateStart > 0 {
			timeQuery["$gte"] = request.DateStart
		}
		if request.DateEnd > 0 {
			timeQuery["$lte"] = request.DateEnd
		}
		criteria["timestamp"] = timeQuery
	}

	if len(request.Types) > 0 {
		criteria["type"] = map[string]interface{}{"$in": request.Types}
	}

	if len(request.Statuses) > 0 {
		criteria["result"] = map[string]interface{}{"$in": request.Statuses}
	}

	if request.Voter != "" {
		criteria["voter"] = request.Voter
	}

	response, err := g.storage.Read("proposals_cache", criteria, options, []string{})
	if err != nil {
		logger.Logger.Error("[query-proposals] Failed to get proposal from cache", zap.Error(err))
		return proposalsResponse, err
	}

	if len(response.Result) == 0 {
		return proposalsResponse, nil
	}

	responseJsonData, err := json.Marshal(response.Result)
	if err != nil {
		return proposalsResponse, err
	}

	err = json.Unmarshal(responseJsonData, &proposals)
	if err != nil {
		return proposalsResponse, err
	}

	proposalsResponse.Proposals = proposals
	proposalsResponse.Pagination.Total = response.Count

	return proposals, nil
}

func (g *CosmosGateway) faucet(req types.InboundRequest) (interface{}, error) {
	request := types.FaucetRequest{}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		logger.Logger.Error("[query-faucet] Invalid request format", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		logger.Logger.Error("[query-faucet] Invalid request format", zap.Error(err))
		return nil, err
	}

	if request.Claim == "" && request.Token == "" {
		faucetAddress := sdk.AccAddress(g.PubKey.Address().Bytes()).String()

		balances, err := g.balances(req, faucetAddress)
		if err != nil {
			logger.Logger.Error("[query-faucet] Failed to get faucet balance", zap.Error(err))
			return nil, err
		}

		var info = types.FaucetAccountInfo{
			Address:  faucetAddress,
			Balances: balances,
		}

		return info, nil
	} else if request.Claim != "" && request.Token != "" {
		return g.processFaucet(req)
	} else {
		err = errors.New("[query-faucet] both claim and token parameters are required")
	}

	return nil, nil
}

func (g *CosmosGateway) processFaucet(req types.InboundRequest) (interface{}, error) {
	request := types.FaucetRequest{}

	jsonData, err := json.Marshal(req.Payload)
	if err != nil {
		logger.Logger.Error("[query-faucet] Invalid request format", zap.Error(err))
		return nil, err
	}

	err = json.Unmarshal(jsonData, &request)
	if err != nil {
		logger.Logger.Error("[query-faucet] Invalid request format", zap.Error(err))
		return nil, err
	}

	result, err := g.storage.Read("cosmos_faucet", map[string]interface{}{"address": request.Claim}, &adapter.Options{Sort: map[string]interface{}{"timestamp": -1}}, []string{})
	if err != nil {
		logger.Logger.Error("[faucet] Failed to get faucet history", zap.Any("CAddress", request.Claim), zap.Error(err))
		return nil, err
	}

	if len(result.Result) > 0 {
		lastTimeFloat, ok := result.Result[0]["timestamp"]
		if !ok {
			err = errors.New("[faucet] Invalid faucet history response")
			logger.Logger.Error("[faucet] Invalid faucet history response", zap.Any("Address", request.Claim))
			return nil, err
		}

		lastTime, ok := lastTimeFloat.(float64)
		if !ok {
			err = errors.New("[faucet] Invalid faucet history response")
			logger.Logger.Error("[faucet] Invalid faucet history response", zap.Any("Address", request.Claim))
			return nil, err
		}

		left := (int64(lastTime) + g.config.Faucet.TimeLimit) - time.Now().UTC().Unix()
		if left > 0 {
			err = errors.New(fmt.Sprintf("[faucet] Claim time left: %d", left))
			logger.Logger.Error("[faucet] Claim time left", zap.Any("Address", request.Claim), zap.Any("Time left", left))
			return nil, err
		}
	}

	faucetAddress := sdk.AccAddress(g.PubKey.Address().Bytes()).String()

	faucetBalances, err := g.balances(req, faucetAddress)
	if err != nil {
		logger.Logger.Error("[faucet] Failed to get faucet balance", zap.Error(err))
		return nil, err
	}

	claimBalances, err := g.balances(req, request.Claim)
	if err != nil {
		logger.Logger.Error("[faucet] Failed to get faucet balance", zap.Error(err))
		return nil, err
	}

	availableAmount := new(big.Int)
	availableAmount.SetString("0", 10)

	for _, balance := range faucetBalances {
		if balance.Denom == request.Token {
			availableAmount.Set(balance.Amount.BigInt())
		}
	}

	claimAmount := new(big.Int)
	claimAmount.SetString("0", 10)
	for _, balance := range claimBalances {
		if balance.Denom == request.Token {
			claimAmount.Set(balance.Amount.BigInt())
		}
	}

	faucetAmount := new(big.Int)
	faucetAmountInt64, ok := g.config.Faucet.FaucetAmounts[request.Token]
	if !ok {
		err = errors.New("[faucet] Failed to get faucet amount from the configuration")
		logger.Logger.Error("[faucet] Failed to get faucet amount from the configuration")
		return nil, err
	}
	faucetAmount.SetInt64(faucetAmountInt64)

	faucetMinimumAmount := new(big.Int)
	faucetMinimumAmountInt64, ok := g.config.Faucet.FaucetMinimumAmounts[request.Token]
	if !ok {
		err = errors.New("[faucet] Failed to get faucet minimum amount from the configuration")
		logger.Logger.Error("[faucet] Failed to get faucet minimum amount from the configuration")
		return nil, err
	}
	faucetMinimumAmount.SetInt64(faucetMinimumAmountInt64)

	feeInt64, ok := g.config.Faucet.FeeAmounts[request.Token]
	if !ok {
		err = errors.New("[faucet] Failed to get fee amount from the configuration")
		logger.Logger.Error("[faucet] Failed to get fee amount from the configuration")
		return nil, err
	}

	if faucetAmount.Cmp(claimAmount) <= 0 {
		err = errors.New("[faucet] No need to send tokens: faucetAmount <= claimAmount")
		logger.Logger.Error("[faucet] No need to send tokens: faucetAmount <= claimAmount")
		return nil, err
	}

	claimingAmount := new(big.Int)
	claimingAmount.SetString("0", 10)
	claimingAmount = claimingAmount.Sub(faucetAmount, claimAmount)
	if claimingAmount.Cmp(faucetMinimumAmount) <= 0 {
		err = errors.New("[faucet] No need to send tokens: faucetAmount <= claimAmount")
		logger.Logger.Error("[faucet] No need to send tokens: faucetAmount <= claimAmount")
		return nil, err
	}

	remainingAmount := new(big.Int)
	remainingAmount.SetString("0", 10)
	remainingAmount = remainingAmount.Sub(availableAmount, faucetMinimumAmount)
	if claimingAmount.Cmp(remainingAmount) > 0 {
		err = errors.New("[faucet] Not enough tokens: faucetAmount-claimAmount > availableAmount-faucetMininumAmount")
		logger.Logger.Error("[faucet] Not enough tokens: faucetAmount-claimAmount > availableAmount-faucetMininumAmount")
		return nil, err
	}

	accountInfo, err := g.account(faucetAddress)
	if err != nil {
		logger.Logger.Error("[faucet] Failed to get account info", zap.Error(err))
		return nil, err
	}

	accountNumber, err := strconv.ParseUint(accountInfo.Account.AccountNumber, 10, 64)
	if err != nil {
		logger.Logger.Error("[faucet] Invalid account response format", zap.Error(err))
		return nil, err
	}

	sequence, err := strconv.ParseUint(accountInfo.Account.Sequence, 10, 64)
	if err != nil {
		logger.Logger.Error("[faucet] Invalid account response format", zap.Error(err))
		return nil, err
	}

	status, err := g.status()
	if err != nil {
		logger.Logger.Error("[faucet] Failed to get node status", zap.Error(err))
		return nil, err
	}

	msgSend := &bank.MsgSend{
		FromAddress: faucetAddress,
		ToAddress:   request.Claim,
		Amount:      sdk.NewCoins(sdk.NewCoin(request.Token, sdk.NewIntFromBigInt(claimingAmount))),
	}

	txBuilder := g.txConfig.NewTxBuilder()

	err = txBuilder.SetMsgs(msgSend)
	if err != nil {
		logger.Logger.Error("[faucet] Failed to set tx msgs", zap.Error(err))
		return nil, err
	}

	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(request.Token, sdk.NewInt(feeInt64))))
	txBuilder.SetGasLimit(200000)
	txBuilder.SetMemo("Faucet Transfer")

	signMode := g.txConfig.SignModeHandler().DefaultMode()

	signerData := authsigning.SignerData{
		Address:       faucetAddress,
		ChainID:       status.NodeInfo.Network,
		AccountNumber: accountNumber,
		Sequence:      sequence,
		PubKey:        g.PubKey,
	}

	signBytes, err := g.txConfig.SignModeHandler().GetSignBytes(
		signMode,
		signerData,
		txBuilder.GetTx(),
	)
	if err != nil {
		logger.Logger.Error("[faucet] Failed to get sign bytes", zap.Error(err))
		return nil, err
	}

	sig, _, err := g.kRing.Sign(g.kName, signBytes)
	if err != nil {
		logger.Logger.Error("[faucet] Failed to sign transaction", zap.Error(err))
		return nil, err
	}

	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: sig,
	}

	sigV2 := signing.SignatureV2{
		PubKey:   g.PubKey,
		Data:     &sigData,
		Sequence: signerData.Sequence,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		logger.Logger.Error("[faucet] Failed to set signatures", zap.Error(err))
		return nil, err
	}

	txBytes, err := g.txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		logger.Logger.Error("[faucet] Failed to encode transaction", zap.Error(err))
		return nil, err
	}

	tHash, err := g.txs(types.InboundRequest{
		Method: "POST",
		Payload: map[string]interface{}{
			"tx":   txBytes,
			"mode": "sync",
		},
	})
	if err != nil {
		logger.Logger.Error("[faucet] Failed to write faucet claim to database", zap.Error(err))
		return tHash, err
	}

	_, err = g.storage.Create("cosmos_faucet", []interface{}{map[string]interface{}{
		"address":   request.Claim,
		"timestamp": time.Now().UTC().Unix(),
		"amount":    claimingAmount.String(),
		"token":     request.Token,
	}})
	if err != nil {
		logger.Logger.Error("[faucet] Failed to write faucet claim to database", zap.Error(err))
	}

	return tHash, nil
}
