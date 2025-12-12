package statesync

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Config holds statesync configuration values
type Config struct {
	Enable      bool
	RPCServers  string
	TrustHeight int64
	TrustHash   string
}

// blockResponse represents the RPC /block response
type blockResponse struct {
	Result struct {
		BlockID struct {
			Hash string `json:"hash"`
		} `json:"block_id"`
		Block struct {
			Header struct {
				Height string `json:"height"`
			} `json:"header"`
		} `json:"block"`
	} `json:"result"`
}

// statusResponse represents the RPC /status response
type statusResponse struct {
	Result struct {
		SyncInfo struct {
			LatestBlockHeight string `json:"latest_block_height"`
		} `json:"sync_info"`
	} `json:"result"`
}

// FetchConfig fetches statesync configuration from an RPC node
// snapshotInterval is used to calculate trust_height (typically 1000 or from node config)
func FetchConfig(rpcNode string, snapshotInterval int64) (*Config, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Ensure rpcNode has scheme
	if rpcNode[0:4] != "http" {
		rpcNode = "http://" + rpcNode
	}

	// 1. Get latest block height
	latestHeight, err := getLatestHeight(client, rpcNode)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest height: %w", err)
	}

	// 2. Calculate trust height (round down to snapshot interval)
	trustHeight := latestHeight - (latestHeight % snapshotInterval)
	if trustHeight <= 0 {
		return nil, fmt.Errorf("invalid trust height: %d", trustHeight)
	}

	// 3. Get block hash at trust height
	trustHash, err := getBlockHash(client, rpcNode, trustHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get block hash at height %d: %w", trustHeight, err)
	}

	return &Config{
		Enable:      true,
		RPCServers:  fmt.Sprintf("%s,%s", rpcNode, rpcNode), // Same server twice (can be improved)
		TrustHeight: trustHeight,
		TrustHash:   trustHash,
	}, nil
}

func getLatestHeight(client *http.Client, rpcNode string) (int64, error) {
	resp, err := client.Get(rpcNode + "/status")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var status statusResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return 0, err
	}

	height, err := strconv.ParseInt(status.Result.SyncInfo.LatestBlockHeight, 10, 64)
	if err != nil {
		return 0, err
	}

	return height, nil
}

func getBlockHash(client *http.Client, rpcNode string, height int64) (string, error) {
	url := fmt.Sprintf("%s/block?height=%d", rpcNode, height)
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var block blockResponse
	if err := json.Unmarshal(body, &block); err != nil {
		return "", err
	}

	return block.Result.BlockID.Hash, nil
}
