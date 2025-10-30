package gateway

import (
	"encoding/json"
	"errors"
	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-service/service"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"time"

	"github.com/saiset-co/sai-interx-manager/types"
)

type StorageGateway struct {
	*BaseGateway
	storage types.Storage
}

var _ types.Gateway = (*StorageGateway)(nil)

func NewStorageGateway(ctx *service.Context, storage types.Storage, retryAttempts int, retryDelay time.Duration, rateLimit int) (*StorageGateway, error) {
	return &StorageGateway{
		BaseGateway: NewBaseGateway(ctx, retryAttempts, retryDelay, rateLimit),
		storage:     storage,
	}, nil
}

func (g *StorageGateway) Handle(data []byte) (interface{}, error) {
	var req struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		logger.Logger.Error("StorageGateway - Handle", zap.Error(err))
		return nil, err
	}

	return g.retry.Do(func() (interface{}, error) {
		if err := g.rateLimit.Wait(g.context.Context); err != nil {
			return nil, err
		}
		switch req.Method {
		case "create":
			return g.storage.Create(cast.ToString(req.Params["collection"]), req.Params["data"])
		case "read":
			criteria, err := cast.ToStringMapE(req.Params["select"])
			if err != nil {
				logger.Logger.Error("StorageGateway - Handle", zap.Error(err))
				return nil, err
			}
			return g.storage.Read(cast.ToString(req.Params["collection"]), criteria, nil, []string{})
		case "update":
			criteria, err := cast.ToStringMapE(req.Params["select"])
			if err != nil {
				logger.Logger.Error("StorageGateway - Handle", zap.Error(err))
				return nil, err
			}
			return g.storage.Update(cast.ToString(req.Params["collection"]), criteria, req.Params["data"])
		case "delete":
			criteria, err := cast.ToStringMapE(req.Params["select"])
			if err != nil {
				logger.Logger.Error("StorageGateway - Handle", zap.Error(err))
				return nil, err
			}
			return g.storage.Delete(cast.ToString(req.Params["collection"]), criteria)
		}

		err := errors.New("method not found")
		logger.Logger.Error("StorageGateway - Handle", zap.Error(err))

		return nil, err
	})
}

func (g *StorageGateway) Close() {

}
