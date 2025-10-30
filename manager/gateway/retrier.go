package gateway

import (
	"github.com/saiset-co/sai-interx-manager/logger"
	"go.uber.org/zap"
	"time"
)

type RetryFunc func() (interface{}, error)

type Retrier struct {
	attempts int
	delay    time.Duration
}

func NewRetrier(attempts int, delay time.Duration) *Retrier {
	return &Retrier{
		attempts: attempts,
		delay:    delay,
	}
}

func (r *Retrier) Do(fn RetryFunc) (interface{}, error) {
	var lastError error

	for i := 0; i < r.attempts; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		} else {
			lastError = err
		}

		if i < r.attempts-1 {
			time.Sleep(r.delay)
		}
	}

	logger.Logger.Error("Retrier - all retry attempts failed", zap.Error(lastError))

	return nil, lastError
}
