package interxhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx"
	"go.uber.org/zap"
)

var log = logger.GetLogger()

func CheckInterxStart(ctx context.Context) error {
	timeout := time.Second * 60
	log.Debug("Checking if interx is started with timeout ", zap.Duration("timeout", timeout))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			status, err := GetInterxStatus(ctx, types.INTERX_CONTAINER_ADDRESS, strconv.Itoa(types.DEFAULT_INTERX_PORT))
			if err != nil {
				log.Warn("ERROR when getting interx status:", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}
			latestBlock, err := strconv.Atoi(status.InterxInfo.LatestBlockHeight)
			log.Debug("Latest block:", zap.Int("latestBlock", latestBlock))
			if err != nil {
				log.Warn("ERROR when converting latest block to string", zap.Error(err))
				continue
				// return err
			}
			if latestBlock > 0 {
				return nil
			}
		}
	}
	return nil
}

func GetInterxStatus(ctx context.Context, ip, port string) (*interx.Status, error) {
	client := &http.Client{}
	ctxWithTO, c := context.WithTimeout(ctx, time.Second*10)
	defer c()
	// log.Printf("Getting net_info from: %v", ip)
	url := fmt.Sprintf("http://%v:%v/api/status", ip, port)
	req, err := http.NewRequestWithContext(ctxWithTO, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var nodeStatus interx.Status
	err = json.Unmarshal(b, &nodeStatus)
	if err != nil {
		return nil, err
	}
	return &nodeStatus, nil
}
