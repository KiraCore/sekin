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
	interxv2 "github.com/kiracore/sekin/src/shidai/internal/types/endpoints/interx_V2"
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
			status, err := GetInterxStatusV2(ctx, types.INTERX_CONTAINER_ADDRESS, types.DEFAULT_INTERX_PORT)
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

// func GetInterxStatus(ctx context.Context, ip, port string) (*interx.Status, error) {
// 	client := &http.Client{}
// 	ctxWithTO, c := context.WithTimeout(ctx, time.Second*10)
// 	defer c()
// 	// log.Printf("Getting net_info from: %v", ip)
// 	url := fmt.Sprintf("http://%v:%v/%v", ip, port, types.ENDPOINT_INTERX_STATUS)
// 	req, err := http.NewRequestWithContext(ctxWithTO, http.MethodGet, url, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	b, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var nodeStatus interx.Status
// 	err = json.Unmarshal(b, &nodeStatus)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &nodeStatus, nil
// }

// func GetNetInfo(ctx context.Context, ipAddress, port string) (*interx.NetInfo, error) {
// 	url := fmt.Sprintf("http://%s:%s/%s", ipAddress, port, types.ENDPOINT_INTERX_NET_INFO)
// 	client := &http.Client{}
// 	log.Debug("Querying sekai status by url:", zap.String("url", url))

// 	ctxWithTO, c := context.WithTimeout(ctx, time.Second*10)
// 	defer c()
// 	req, err := http.NewRequestWithContext(ctxWithTO, http.MethodGet, url, nil)
// 	if err != nil {
// 		return nil, err
// 	}

// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var response *interx.NetInfo
// 	err = json.Unmarshal(body, &response)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return response, nil
// }

// new interx compatible

func GetValopersV2(ctc context.Context, ip string, port int) (*interxv2.Valopers, error) {
	url := fmt.Sprintf("http://%s:%d/%v?all=true", ip, port, types.ENDPOINT_INTERX_VALOPERS)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var result interxv2.Valopers
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return &result, nil
}

func GetNetInfoV2(ctc context.Context, ip string, port int) (*interxv2.NetInfo, error) {
	url := fmt.Sprintf("http://%s:%d/%v", ip, port, types.ENDPOINT_INTERX_NET_INFO)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body failed: %w", err)
	}

	var result interxv2.NetInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return &result, nil
}

func GetInterxStatusV2(ctx context.Context, ip string, port int) (*interxv2.Status, error) {
	client := &http.Client{}
	ctxWithTO, c := context.WithTimeout(ctx, time.Second*10)
	defer c()
	// log.Printf("Getting net_info from: %v", ip)
	url := fmt.Sprintf("http://%v:%v/%v", ip, port, types.ENDPOINT_INTERX_STATUS)
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

	var nodeStatus interxv2.Status
	err = json.Unmarshal(b, &nodeStatus)
	if err != nil {
		return nil, err
	}
	return &nodeStatus, nil
}
