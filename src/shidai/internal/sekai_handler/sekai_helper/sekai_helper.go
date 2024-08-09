package sekaihelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	httpexecutor "github.com/kiracore/sekin/src/shidai/internal/http_executor"
	"github.com/kiracore/sekin/src/shidai/internal/logger"
	"github.com/kiracore/sekin/src/shidai/internal/types"
	"github.com/kiracore/sekin/src/shidai/internal/types/endpoints/sekai"
	"go.uber.org/zap"
)

const endpointStatus string = "status"

var (
	log = logger.GetLogger()
)

func GetSekaidStatus(ctx context.Context, ipAddress, rpcPort string) (*sekai.Status, error) {
	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, rpcPort, endpointStatus)
	client := &http.Client{}

	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return nil, fmt.Errorf("error when getting sekaid status: %w", err)
	}

	var response *sekai.Status
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("error when getting sekaid status %w", err)
	}

	return response, nil
}

func CheckSekaiStart(ctx context.Context) error {
	timeout := time.Second * 60
	log.Debug("Checking if sekai is started with timeout ", zap.Duration("timeout", timeout))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			status, err := GetSekaidStatus(ctx, types.SEKAI_CONTAINER_ADDRESS, "26657")
			if err != nil {
				log.Warn("ERROR when getting sekai status:", zap.Error(err))
				time.Sleep(time.Second)
				continue
			}
			latestBlock, err := strconv.Atoi(status.Result.SyncInfo.LatestBlockHeight)
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
}

// check during n-period of time every 5 seconds if network producing blocks
func CheckConsensus(ctx context.Context, ipAddress, rpcPort string, timePeriod time.Duration) (bool, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	log.Debug("Checking consensus")
	timer := time.NewTimer(timePeriod)
	defer timer.Stop()
	checkStatus, err := GetSekaidStatus(ctx, ipAddress, rpcPort)
	if err != nil {
		return false, err
	}
	for {
		select {
		case <-ticker.C:
			currentStatus, err := GetSekaidStatus(ctx, ipAddress, rpcPort)
			if err != nil {
				return false, err
			}
			log.Debug("check block height", zap.String("check height", checkStatus.Result.SyncInfo.LatestBlockHeight), zap.String("current height", currentStatus.Result.SyncInfo.LatestBlockHeight))
			if currentStatus.Result.SyncInfo.LatestBlockHeight > checkStatus.Result.SyncInfo.LatestBlockHeight {
				log.Debug("Blocks are producing")
				return true, nil
			}

		case <-timer.C:
			log.Debug("Blocks are not minting")
			return false, nil
		}
	}

}
