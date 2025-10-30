package gateway

import (
	"encoding/json"
	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-service/service"
	"go.uber.org/zap"
	"time"

	"github.com/saiset-co/sai-interx-manager/types"
)

type BitcoinGateway struct {
	*BaseGateway
	url string
}

var _ types.Gateway = (*BitcoinGateway)(nil)

func NewBitcoinGateway(ctx *service.Context, url string, retryAttempts int, retryDelay time.Duration, rateLimit int) (*BitcoinGateway, error) {
	return &BitcoinGateway{
		BaseGateway: NewBaseGateway(ctx, retryAttempts, retryDelay, rateLimit),
		url:         url,
	}, nil
}

func (g *BitcoinGateway) Handle(data []byte) (interface{}, error) {
	var req struct {
		Method   string      `json:"method"`
		Data     interface{} `json:"data"`
		Metadata struct {
			Token string `json:"token"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		logger.Logger.Error("CosmosGateway - Handle", zap.Error(err))
		return nil, err
	}

	return g.retry.Do(func() (interface{}, error) {
		if err := g.rateLimit.Wait(g.context.Context); err != nil {
			logger.Logger.Error("CosmosGateway - Handle", zap.Error(err))
			return nil, err
		}
		return g.makeSaiRequest(g.context.Context, g.url, req)
	})
}

func (g *BitcoinGateway) Close() {

}
