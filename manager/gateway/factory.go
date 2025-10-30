package gateway

import (
	"encoding/json"
	"fmt"
	"github.com/saiset-co/sai-interx-manager/logger"
	"go.uber.org/zap"
	"time"

	saiService "github.com/saiset-co/sai-service/service"
	"github.com/spf13/cast"

	"github.com/saiset-co/sai-interx-manager/types"
)

type GatewayFactory struct {
	context *saiService.Context
	storage types.Storage
}

func NewGatewayFactory(context *saiService.Context, storage types.Storage) *GatewayFactory {
	return &GatewayFactory{
		context: context,
		storage: storage,
	}
}

func (f *GatewayFactory) CreateGateway(gatewayType string) (types.Gateway, error) {
	switch gatewayType {
	case "ethereum":
		return NewEthereumGateway(
			f.context,
			cast.ToStringMapString(f.context.GetConfig("ethereum.nodes", map[string]string{})),
			f.storage,
			cast.ToInt(f.context.GetConfig("ethereum.retries", 1)),
			time.Duration(cast.ToInt64(f.context.GetConfig("ethereum.retry_delay", 10))),
			cast.ToInt(f.context.GetConfig("ethereum.rate_limit", 10)),
		)
	case "cosmos":
		var cosmosConfig types.CosmosConfig

		configBytes, err := json.Marshal(f.context.GetConfig("cosmos", cosmosConfig))
		if err != nil {
			logger.Logger.Error("Invalid cosmos configuration format")
			return nil, err
		}

		err = json.Unmarshal(configBytes, &cosmosConfig)
		if err != nil {
			logger.Logger.Error("Invalid cosmos configuration format")
			return nil, err
		}

		return NewCosmosGateway(
			f.context,
			f.storage,
			cosmosConfig,
		)
	case "bitcoin":
		return NewBitcoinGateway(
			f.context,
			cast.ToString(f.context.GetConfig("bitcoin.url", "")),
			cast.ToInt(f.context.GetConfig("bitcoin.retries", 1)),
			time.Duration(cast.ToInt64(f.context.GetConfig("bitcoin.retry_delay", 10))),
			cast.ToInt(f.context.GetConfig("bitcoin.rate_limit", 10)),
		)
	case "storage":
		return NewStorageGateway(
			f.context,
			f.storage,
			cast.ToInt(f.context.GetConfig("storage.retries", 1)),
			time.Duration(cast.ToInt64(f.context.GetConfig("storage.retry_delay", 10))),
			cast.ToInt(f.context.GetConfig("storage.rate_limit", 10)),
		)
	default:
		err := fmt.Errorf("unknown gateway type: %s", gatewayType)
		logger.Logger.Error("GatewayFactory - CreateGateway", zap.Error(err))

		return nil, err
	}
}
