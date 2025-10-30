package gateway

import (
	"encoding/json"
	"errors"
	"github.com/saiset-co/sai-service/service"
	"github.com/spf13/cast"
	"math/big"
	"strconv"
	"strings"
	"time"

	jsonrpc2 "github.com/KeisukeYamashita/go-jsonrpc"
	"go.uber.org/zap"

	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/types"
)

type EthereumGateway struct {
	*BaseGateway
	storage    types.Storage
	rpcProxies map[string]*jsonrpc2.RPCClient
}

var _ types.Gateway = (*EthereumGateway)(nil)

func newJsonRPCClients(chains map[string]string) map[string]*jsonrpc2.RPCClient {
	proxies := map[string]*jsonrpc2.RPCClient{}

	for chainId, url := range chains {
		proxies[chainId] = jsonrpc2.NewRPCClient(url)
	}

	return proxies
}

func NewEthereumGateway(ctx *service.Context, chains map[string]string, storage types.Storage, retryAttempts int, retryDelay time.Duration, rateLimit int) (*EthereumGateway, error) {
	return &EthereumGateway{
		BaseGateway: NewBaseGateway(ctx, retryAttempts, retryDelay, rateLimit),
		rpcProxies:  newJsonRPCClients(chains),
		storage:     storage,
	}, nil
}

func (g *EthereumGateway) Handle(data []byte) (interface{}, error) {
	var req types.InboundRequest

	if err := json.Unmarshal(data, &req); err != nil {
		logger.Logger.Error("EthereumGateway - Handle", zap.Error(err))
		return nil, err
	}

	chainId, method, err := g.convert(req.Path)
	if err != nil {
		logger.Logger.Error("EthereumGateway - Handle", zap.Error(err))
		return nil, err
	}

	client, ok := g.rpcProxies[chainId]
	if !ok {
		err = errors.New("chain not found")
		logger.Logger.Error("EthereumGateway - Handle", zap.Error(err))
		return nil, err
	}

	switch req.Path {
	case "/status":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err))
					return nil, err
				}
				return g.status(client, chainId)
			})
		}
	}

	return g.retry.Do(func() (interface{}, error) {
		if err := g.rateLimit.Wait(g.context.Context); err != nil {
			logger.Logger.Error("EthereumGateway - Handle", zap.Error(err))
			return nil, err
		}
		return client.Call(method, req.Payload)
	})
}

func (g *EthereumGateway) Close() {

}

func (g *EthereumGateway) convert(originalPath string) (chainId, method string, err error) {
	paths := strings.Split(originalPath, "/")
	if len(paths) < 3 {
		return "", "", err
	}

	return paths[2], paths[3], nil
}

func (g *EthereumGateway) status(client *jsonrpc2.RPCClient, chainId string) (interface{}, error) {
	chains := cast.ToStringMapString(g.context.GetConfig("ethereum.nodes", map[string]string{}))

	var response = types.EVMStatus{}

	response.NodeInfo.RPCAddress = chains[chainId]

	data, err := client.Call("eth_chainId")
	if err != nil {
		return nil, err
	}

	response.NodeInfo.Network, _ = strconv.ParseUint((chainId)[2:], 16, 64)

	data, err = client.Call("web3_clientVersion")
	if err != nil {
		return nil, err
	}
	clientVersion, err := data.GetString()
	if err != nil {
		clientVersion = ""
	}
	response.NodeInfo.Version.Web3 = clientVersion

	data, err = client.Call("net_version")
	if err != nil {
		return nil, err
	}
	netVersion, err := data.GetString()
	if err != nil {
		netVersion = ""
	}
	response.NodeInfo.Version.Net = netVersion

	data, err = client.Call("eth_protocolVersion")
	if err != nil {
		return nil, err
	}
	protocolVersion, err := data.GetString()
	if err != nil {
		protocolVersion = ""
	}
	response.NodeInfo.Version.Protocol = protocolVersion

	data, err = client.Call("eth_syncing")
	if err != nil {
		return nil, err
	}
	isSyncing, err := data.GetBool()
	if err != nil {
		isSyncing = true
	}
	response.SyncInfo.CatchingUp = isSyncing

	latestBlock := new(struct {
		Hash      string `json:"hash"`
		Number    string `json:"number"`
		Timestamp string `json:"timestamp"`
	})
	data, err = client.Call("eth_getBlockByNumber", "latest", true)
	if err != nil {
		return nil, err
	}
	err = data.GetObject(latestBlock)
	if err != nil {
		return nil, err
	}
	response.SyncInfo.LatestBlockHash = latestBlock.Hash
	response.SyncInfo.LatestBlockHeight, _ = strconv.ParseUint((latestBlock.Number)[2:], 16, 64)
	response.SyncInfo.LatestBlockTime, _ = strconv.ParseUint((latestBlock.Timestamp)[2:], 16, 64)

	earliestBlock := new(struct {
		Hash      string `json:"hash"`
		Number    string `json:"number"`
		Timestamp string `json:"timestamp"`
	})
	data, err = client.Call("eth_getBlockByNumber", "earliest", true)
	if err != nil {
		return nil, err
	}
	err = data.GetObject(earliestBlock)
	if err != nil {
		return nil, err
	}
	response.SyncInfo.EarliestBlockHash = earliestBlock.Hash
	response.SyncInfo.EarliestBlockHeight, _ = strconv.ParseUint((earliestBlock.Number)[2:], 16, 64)
	response.SyncInfo.EarliestBlockTime, _ = strconv.ParseUint((earliestBlock.Timestamp)[2:], 16, 64)

	data, err = client.Call("eth_gasPrice")
	if err != nil {
		return nil, err
	}
	gasPrice, err := data.GetString()
	if err != nil {
		return nil, err
	}
	gasPriceBig := *new(big.Int)
	gasPriceBig.SetString((gasPrice)[2:], 16)
	response.GasPrice = gasPriceBig.String()

	return response, nil
}
