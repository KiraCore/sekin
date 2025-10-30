package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	cosmosAuth "github.com/saiset-co/sai-interx-manager/proto-gen/cosmos/auth/v1beta1"
	cosmosBank "github.com/saiset-co/sai-interx-manager/proto-gen/cosmos/bank/v1beta1"
	cosmosTx "github.com/saiset-co/sai-interx-manager/proto-gen/cosmos/tx/v1beta1"
	kiraGov "github.com/saiset-co/sai-interx-manager/proto-gen/kira/gov"
	kiraMultiStaking "github.com/saiset-co/sai-interx-manager/proto-gen/kira/multistaking"
	kiraSlashing "github.com/saiset-co/sai-interx-manager/proto-gen/kira/slashing/v1beta1"
	kiraSpending "github.com/saiset-co/sai-interx-manager/proto-gen/kira/spending"
	kiraStaking "github.com/saiset-co/sai-interx-manager/proto-gen/kira/staking"
	kiraTokens "github.com/saiset-co/sai-interx-manager/proto-gen/kira/tokens"
	kiraUbi "github.com/saiset-co/sai-interx-manager/proto-gen/kira/ubi"
	kiraUpgrades "github.com/saiset-co/sai-interx-manager/proto-gen/kira/upgrade"

	sekaitypes "github.com/KiraCore/sekai/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-interx-manager/types"
	"github.com/saiset-co/sai-service/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Proxy struct {
	mux  *runtime.ServeMux
	conn *grpc.ClientConn
}

type CosmosGateway struct {
	*BaseGateway
	storage   types.Storage
	config    types.CosmosConfig
	grpcProxy *Proxy
	txConfig  client.TxConfig
	kRing     keyring.Keyring
	kName     string
	PubKey    *secp256k1.PubKey
}

const (
	Active   string = "ACTIVE"
	Inactive string = "INACTIVE"
	Paused   string = "PAUSED"
	Jailed   string = "JAILED"
)

var (
	AccountAddressPrefix   = "kira"
	AccountPubKeyPrefix    = "kirapub"
	ValidatorAddressPrefix = "kiravaloper"
	ValidatorPubKeyPrefix  = "kiravaloperpub"
	ConsNodeAddressPrefix  = "kiravalcons"
	ConsNodePubKeyPrefix   = "kiravalconspub"
)

var _ types.Gateway = (*CosmosGateway)(nil)

func newGRPCGatewayProxy(ctx *service.Context, address string) (*Proxy, error) {
	conn, err := grpc.DialContext(
		ctx.Context,
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	mux := runtime.NewServeMux()

	if err := registerHandlers(ctx, mux, conn); err != nil {
		conn.Close()
		logger.Logger.Error("newGRPCGatewayProxy", zap.Error(err))
		return nil, err
	}

	return &Proxy{
		mux:  mux,
		conn: conn,
	}, nil
}

func (p *Proxy) ServeGRPC(r *http.Request) ([]byte, error) {
	r.Header.Set("Content-Type", "application/json")

	//Todo: add cache here

	recorder := httptest.NewRecorder()
	p.mux.ServeHTTP(recorder, r)
	resp := recorder.Result()

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error("CosmosGateway - Handle - gRPC gateway error response", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var result = new(types.GRPCResponse)
		err := json.Unmarshal(bodyBytes, &result)
		if err != nil {
			logger.Logger.Error("[query-network-properties] Invalid response format", zap.Error(err))
			return nil, err
		}

		errMsg := fmt.Sprintf("gRPC gateway error: url=%s status=%d, code=%f, message=%s, details=%s", r.URL.Path, resp.StatusCode, result.Code, result.Message, string(result.Details))

		logger.Logger.Error("CosmosGateway - Handle - gRPC gateway error response",
			zap.Int("status", resp.StatusCode),
			zap.Any("code", result.Code),
			zap.Any("message", result.Message),
			zap.Any("details", result.Details))

		return nil, fmt.Errorf(errMsg)
	}

	return bodyBytes, nil
}

func registerHandlers(ctx *service.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if err := cosmosTx.RegisterServiceHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := cosmosBank.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := cosmosAuth.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraGov.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraStaking.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraMultiStaking.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraSlashing.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraTokens.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraUpgrades.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraSpending.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	if err := kiraUbi.RegisterQueryHandler(ctx.Context, mux, conn); err != nil {
		logger.Logger.Error("registerHandlers", zap.Error(err))
		return err
	}

	return nil
}

func NewCosmosGateway(ctx *service.Context, storage types.Storage, cosmosConfig types.CosmosConfig) (*CosmosGateway, error) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(AccountAddressPrefix, AccountPubKeyPrefix)
	config.SetBech32PrefixForValidator(ValidatorAddressPrefix, ValidatorPubKeyPrefix)
	config.SetBech32PrefixForConsensusNode(ConsNodeAddressPrefix, ConsNodePubKeyPrefix)
	config.Seal()

	mnemonic := loadMnemonicFile()

	if mnemonic == "" {
		entropy, _ := bip39.NewEntropy(256)
		mnemonic, _ = bip39.NewMnemonic(entropy)
		saveMnemonicFile([]byte(mnemonic))
	}

	if !bip39.IsMnemonicValid(mnemonic) {
		fmt.Println("Invalid Interx Mnemonic: ", mnemonic)
		panic("Invalid Interx Mnemonic")
	}

	hdPath := sdk.GetConfig().GetFullBIP44Path()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	interfaceRegistry.RegisterInterface("types.PubKey", (*cryptotypes.PubKey)(nil), &secp256k1.PubKey{})
	interfaceRegistry.RegisterInterface("types.PrivKey", (*cryptotypes.PrivKey)(nil), &secp256k1.PrivKey{})
	interfaceRegistry.RegisterInterface("types.Msg", (*sdk.Msg)(nil), &banktypes.MsgSend{})
	_codec := codec.NewProtoCodec(interfaceRegistry)
	txConfig := authtx.NewTxConfig(_codec, authtx.DefaultSignModes)
	kRing := keyring.NewInMemory(_codec)
	kName := uuid.New().String()

	kInfo, err := kRing.NewAccount(kName, mnemonic, "", hdPath, hd.Secp256k1)
	if err != nil {
		logger.Logger.Error("Failed to create account from mnemonic", zap.Error(err))
		return nil, err
	}

	pubKey, err := kInfo.GetPubKey()
	if err != nil {
		logger.Logger.Error("Failed to create account from mnemonic", zap.Error(err))
		return nil, err
	}

	faucetPubKey := pubKey.(*secp256k1.PubKey)

	proxy, err := newGRPCGatewayProxy(ctx, cosmosConfig.Node.JsonRpc)
	if err != nil {
		logger.Logger.Error("NewCosmosGateway", zap.Error(err))
		return nil, err
	}

	return &CosmosGateway{
		BaseGateway: NewBaseGateway(ctx, cosmosConfig.Retries, time.Duration(cosmosConfig.RetryDelay)*time.Second, cosmosConfig.RateLimit),
		storage:     storage,
		config:      cosmosConfig,
		grpcProxy:   proxy,
		txConfig:    txConfig,
		kRing:       kRing,
		kName:       kName,
		PubKey:      faucetPubKey,
	}, nil
}

func (g *CosmosGateway) Handle(data []byte) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(g.config.GWTimeout)*time.Second)
	defer cancel()

	g.context.Context = ctx

	var req types.InboundRequest

	if err := json.Unmarshal(data, &req); err != nil {
		logger.Logger.Error("CosmosGateway - Handle - Unmarshal request failed", zap.Error(err))
		return nil, err
	}

	accountsRegex := regexp.MustCompile(`^/kira/accounts/(.+)$`)

	if matches := accountsRegex.FindStringSubmatch(req.Path); matches != nil {
		accountID := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}

			return g.account(accountID)
		})
	}

	balancesRegex := regexp.MustCompile(`^/kira/balances/(.+)$`)

	if matches := balancesRegex.FindStringSubmatch(req.Path); matches != nil {
		accountID := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}
			return g.balances(req, accountID)
		})
	}

	identityRecordsRegex := regexp.MustCompile(`^/kira/gov/identity_records/(.+)$`)

	if matches := identityRecordsRegex.FindStringSubmatch(req.Path); matches != nil {
		address := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}
			return g.identityRecords(address)
		})
	}

	identityVerifyRequestsByApproverRegex := regexp.MustCompile(`^/kira/gov/identity_verify_requests_by_approver/(.+)$`)

	if matches := identityVerifyRequestsByApproverRegex.FindStringSubmatch(req.Path); matches != nil {
		approver := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}
			return g.identityVerifyRequestsByApprover(req, approver)
		})
	}

	identityVerifyRequestsByRequesterRegex := regexp.MustCompile(`^/kira/gov/identity_verify_requests_by_requester/(.+)$`)

	if matches := identityVerifyRequestsByRequesterRegex.FindStringSubmatch(req.Path); matches != nil {
		approver := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}
			return g.identityVerifyRequestsByRequester(req, approver)
		})
	}

	txByHashRegex := regexp.MustCompile(`^/transactions/(.+)$`)

	if matches := txByHashRegex.FindStringSubmatch(req.Path); matches != nil {
		hash := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}
			return g.txByHash(hash)
		})
	}

	txByBlockRegex := regexp.MustCompile(`^/blocks/(.+)/transactions$`)

	if matches := txByBlockRegex.FindStringSubmatch(req.Path); matches != nil {
		blockID := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}
			return g.txByBlock(req, blockID)
		})
	}

	blockById := regexp.MustCompile(`^/blocks/(.+)$`)

	if matches := blockById.FindStringSubmatch(req.Path); matches != nil {
		blockID := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}
			return g.blockById(req, blockID)
		})
	}

	proposalById := regexp.MustCompile(`^/kira/gov/proposal/(.+)$`)

	if matches := proposalById.FindStringSubmatch(req.Path); matches != nil {
		proposalId := matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}

			req.Path = "/kira/gov/proposals/" + proposalId

			return g.proxy(req)
		})
	}

	tendermint := regexp.MustCompile(`^/tendermint/(.+)$`)

	if matches := tendermint.FindStringSubmatch(req.Path); matches != nil {
		req.Path = "/" + matches[1]

		return g.retry.Do(func() (interface{}, error) {
			if err := g.rateLimit.Wait(g.context.Context); err != nil {
				logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
				return nil, err
			}

			return g.tendermint(req)
		})
	}

	switch req.Path {
	case "/dashboard":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.dashboard()
			})
		}
	case "/status":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.statusAPI()
			})
		}
	case "/valopers":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.validators(req)
			})
		}
	case "/kira/txs":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.txs(req)
			})
		}
	case "/kira/delegations":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.delegations(req)
			})
		}
	case "/kira/gov/execution_fee":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.executionFee(req)
			})
		}
	case "/kira/gov/network_properties":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.networkProperties()
			})
		}
	case "/kira/staking-pool":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.stakingPool(req)
			})
		}
	case "/kira/undelegations":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.undelegations(req)
			})
		}
	case "/transactions":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.transactions(req)
			})
		}
	case "/blocks":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.blocks(req)
			})
		}
	case "/kira/status":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.status()
			})
		}
	case "/kira/tokens/rates":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.tokenRates()
			})
		}
	case "/kira/gov/proposals":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.proposals(req)
			})
		}
	case "/kira/tokens/aliases":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.tokenAliases(req)
			})
		}
	case "/kira/faucet":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.faucet(req)
			})
		}

	case "/tendermint":
		{
			return g.retry.Do(func() (interface{}, error) {
				if err := g.rateLimit.Wait(g.context.Context); err != nil {
					logger.Logger.Error("EthereumGateway - Handle", zap.Error(err), zap.Any("ctx", g.context.Context))
					return nil, err
				}
				return g.tendermint(req)
			})
		}
	}

	return g.retry.Do(func() (interface{}, error) {
		if err := g.rateLimit.Wait(g.context.Context); err != nil {
			logger.Logger.Error("CosmosGateway - Handle - Rate limit exceeded", zap.Error(err))
			return nil, err
		}

		return g.proxy(req)
	})
}

func (g *CosmosGateway) Close() {
	g.grpcProxy.conn.Close()
}

func (g *CosmosGateway) makeTendermintRPCRequest(ctx context.Context, url string, query string) (interface{}, error) {
	endpoint := fmt.Sprintf("%s%s?%s", g.config.Node.Tendermint, url, query)

	//Todo: add cache here

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		logger.Logger.Error("MakeTendermintRPCRequest - [rpc-call] Unable to connect to server", zap.Error(err))
		return nil, err
	}

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		logger.Logger.Error("MakeTendermintRPCRequest - [rpc-call] Unable to connect to server", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	response := new(types.RPCResponse)
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil {
		logger.Logger.Debug("MakeTendermintRPCRequest - [rpc-call] Unable to decode response",
			zap.Any("body", resp.Body), zap.Any("endpoint", endpoint), zap.Error(err))
		return nil, err
	}

	if response.Error.Code != 0 {
		return nil, errors.New(response.Error.Data)
	}

	return response.Result, nil
}

func (g *CosmosGateway) proxy(req types.InboundRequest) ([]byte, error) {
	dataBytes, err := json.Marshal(req.Payload)
	if err != nil {
		logger.Logger.Error("[query-proxy] Marshal payload failed", zap.Error(err))
		return nil, err
	}

	gatewayReq, err := http.NewRequestWithContext(g.context.Context, req.Method, req.Path, strings.NewReader(string(dataBytes)))
	if err != nil {
		logger.Logger.Error("[query-proxy] Create request failed", zap.Error(err))
		return nil, err
	}

	gatewayReq = g.encodeQuery(gatewayReq, req)
	grpcBytes, err := g.grpcProxy.ServeGRPC(gatewayReq)
	if err != nil {
		logger.Logger.Error("[query-proxy] Serve request failed", zap.Error(err))
		return nil, err
	}

	//var result interface{}
	//
	//err = json.Unmarshal(grpcBytes, &result)
	//if err != nil {
	//	logger.Logger.Error("CosmosGateway - validators - Unmarshal response failed", zap.Error(err))
	//	return nil, err
	//}

	return grpcBytes, nil
}

func (g *CosmosGateway) tendermint(req types.InboundRequest) (interface{}, error) {
	query := mapToQuery(req.Payload)
	return g.makeTendermintRPCRequest(g.context.Context, req.Path, query.Encode())
}

func (g *CosmosGateway) filterAndPaginateValidators(response *types.ValidatorsResponse, payload map[string]interface{}) (*types.ValidatorsResponse, error) {
	type ValidatorsRequest struct {
		Address           string `json:"address,omitempty"`
		Valkey            string `json:"valkey,omitempty"`
		Pubkey            string `json:"pubkey,omitempty"`
		Moniker           string `json:"moniker,omitempty"`
		Status            string `json:"status,omitempty"`
		Offset            int    `json:"offset,string,omitempty"`
		Limit             int    `json:"limit,string,omitempty"`
		Proposer          string `json:"proposer,omitempty"`
		All               bool   `json:"all,string,omitempty"`
		StatusOnly        bool   `json:"status_only,string,omitempty"`
		CountTotal        bool   `json:"count_total,string,omitempty"`
		StakingPoolStatus string `json:"staking_pool_status,omitempty"`
		ValidatorNodeId   string `json:"validator_node_id,omitempty"`
		SentryNodeId      string `json:"sentry_node_id,omitempty"`
	}

	var result = new(types.ValidatorsResponse)
	var filteredValidators []types.QueryValidator

	result.Actors = response.Actors

	params := ValidatorsRequest{
		Limit: sekaitypes.PageIterationLimit - 1,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonData, &params)
	if err != nil {
		return nil, err
	}

	if params.All {
		return response, nil
	}

	for _, validator := range response.Validators {
		match := true

		if params.Address != "" && validator.Address != params.Address {
			match = false
		}

		if params.Valkey != "" && validator.Valkey != params.Valkey {
			match = false
		}

		if params.Pubkey != "" && validator.Pubkey != params.Pubkey {
			match = false
		}

		if params.Moniker != "" && !strings.Contains(strings.ToLower(validator.Moniker), strings.ToLower(params.Moniker)) {
			match = false
		}

		if params.Status != "" && validator.Status != params.Status {
			match = false
		}

		if params.Proposer != "" && validator.Proposer != params.Proposer {
			match = false
		}

		if params.StakingPoolStatus != "" && validator.StakingPoolStatus != params.StakingPoolStatus {
			match = false
		}

		if params.ValidatorNodeId != "" && validator.Validator_node_id != params.ValidatorNodeId {
			match = false
		}

		if params.SentryNodeId != "" && validator.Sentry_node_id != params.SentryNodeId {
			match = false
		}

		if match {
			if params.StatusOnly {
				simplifiedValidator := types.QueryValidator{
					Address: validator.Address,
					Status:  validator.Status,
				}
				filteredValidators = append(filteredValidators, simplifiedValidator)
			} else {
				filteredValidators = append(filteredValidators, validator)
			}
		}
	}

	totalValidators := len(filteredValidators)

	if params.CountTotal {
		result.Pagination.Total = totalValidators
	}

	if params.Offset >= totalValidators {
		result.Validators = []types.QueryValidator{}
	} else {
		endIndex := params.Offset + params.Limit
		if endIndex > totalValidators {
			endIndex = totalValidators
		}

		result.Validators = filteredValidators[params.Offset:endIndex]
	}

	return result, nil
}

func (g *CosmosGateway) encodeQuery(gatewayReq *http.Request, req types.InboundRequest) *http.Request {
	if req.Method == http.MethodGet && len(req.Payload) > 0 {
		q := gatewayReq.URL.Query()
		for k, v := range req.Payload {
			switch val := v.(type) {
			case string:
				q.Add(k, val)
			case float64:
				q.Add(k, fmt.Sprintf("%v", val))
			case bool:
				q.Add(k, fmt.Sprintf("%v", val))
			case []interface{}:
				for _, item := range val {
					q.Add(k, fmt.Sprintf("%v", item))
				}
			case map[string]interface{}:
				for subKey, subVal := range val {
					q.Add(k+"."+subKey, fmt.Sprintf("%v", subVal))
				}
			default:
				q.Add(k, fmt.Sprintf("%v", v))
			}
		}
		gatewayReq.URL.RawQuery = q.Encode()
	}

	return gatewayReq
}

func loadMnemonicFile() string {
	mnemonic, err := ioutil.ReadFile("mnemonic.data")
	if err != nil {
		return ""
	}

	return string(mnemonic)
}

func saveMnemonicFile(data []byte) {
	err := ioutil.WriteFile("mnemonic.data", data, 0644)
	if err != nil {
		panic(err)
	}
}

func mapToQuery(m map[string]interface{}) url.Values {
	query := url.Values{}

	for key, value := range m {
		switch v := value.(type) {
		case string:
			query.Set(key, v)
		case int:
			query.Set(key, strconv.Itoa(v))
		case int64:
			query.Set(key, strconv.FormatInt(v, 10))
		case float64:
			query.Set(key, strconv.FormatFloat(v, 'f', -1, 64))
		case bool:
			query.Set(key, strconv.FormatBool(v))
		case []string:
			for _, item := range v {
				query.Add(key, item)
			}
		case []interface{}:
			for _, item := range v {
				query.Add(key, fmt.Sprintf("%v", item))
			}
		default:
			query.Set(key, fmt.Sprintf("%v", v))
		}
	}

	return query
}
