package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/saiset-co/sai-interx-manager/logger"
	"github.com/saiset-co/sai-service/service"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type BaseGateway struct {
	context   *service.Context
	client    *http.Client
	rateLimit *RateLimiter
	retry     *Retrier
}

func NewBaseGateway(ctx *service.Context, retryAttempts int, retryDelay time.Duration, rateLimit int) *BaseGateway {
	return &BaseGateway{
		context: ctx,
		client: &http.Client{
			Timeout: time.Second * 30,
		},
		rateLimit: NewRateLimiter(rateLimit),
		retry:     NewRetrier(retryAttempts, retryDelay),
	}
}

func (g *BaseGateway) makeSaiRequest(ctx context.Context, url string, payload interface{}) (interface{}, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New("non-200 status code: " + resp.Status)
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (g *BaseGateway) makeRequest(ctx context.Context, method, url string, payload interface{}) (interface{}, error) {
	payloadData, err := json.Marshal(payload)
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(payloadData))
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	resp, err := g.client.Do(req)
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err = errors.New("non-200 status code: " + resp.Status)
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Logger.Error("BaseGateway - makeRequest", zap.Error(err))
		return nil, err
	}

	return result, nil
}
