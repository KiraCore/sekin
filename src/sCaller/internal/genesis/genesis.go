package genesis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// genesisResponse represents the RPC /genesis response
type genesisResponse struct {
	Result struct {
		Genesis json.RawMessage `json:"genesis"`
	} `json:"result"`
}

// Fetch downloads genesis from an RPC node and saves to destPath
func Fetch(rpcNode, destPath string) error {
	client := &http.Client{Timeout: 60 * time.Second}

	// Ensure rpcNode has scheme
	if len(rpcNode) < 4 || rpcNode[0:4] != "http" {
		rpcNode = "http://" + rpcNode
	}

	url := rpcNode + "/genesis"
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch genesis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("genesis request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read genesis response: %w", err)
	}

	// Parse response to extract genesis
	var genResp genesisResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return fmt.Errorf("failed to parse genesis response: %w", err)
	}

	// Normalize and save genesis
	genesis, err := normalizeGenesis(genResp.Result.Genesis)
	if err != nil {
		return fmt.Errorf("failed to normalize genesis: %w", err)
	}

	if err := os.WriteFile(destPath, genesis, 0644); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	return nil
}

// normalizeGenesis ensures consistent JSON formatting
func normalizeGenesis(raw json.RawMessage) ([]byte, error) {
	var genesis map[string]interface{}
	if err := json.Unmarshal(raw, &genesis); err != nil {
		return nil, err
	}

	// Re-marshal with indentation for readability
	return json.MarshalIndent(genesis, "", "  ")
}
