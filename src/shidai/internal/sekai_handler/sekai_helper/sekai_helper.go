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
	log.Debug("Querying sekai status by url:", zap.String("url", url))

	body, err := httpexecutor.DoHttpQuery(ctx, client, url, "GET")
	if err != nil {
		return nil, err
	}

	var response *sekai.Status
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func CheckSekaiStart(ctx context.Context) error {
	timeout := time.Second * 60
	log.Debug("Checking if sekai is started with timeout ", zap.Duration("timeout", timeout))
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	requestCooldown := time.Second * 1

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
		time.Sleep(requestCooldown)
	}
}
