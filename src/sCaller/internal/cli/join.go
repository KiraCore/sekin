package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"scaller/internal/config"
	"scaller/internal/genesis"
	"scaller/internal/statesync"

	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Initialize node and join network",
	Long: `Initializes sekaid, fetches genesis, configures node, and starts it.

Mnemonic is read from stdin.

Example:
  echo "word1 word2 ..." | scaller join --rpc-node 8.8.8.8:26657 --statesync`,
	Run: runJoin,
}

var (
	joinRPCNode     string
	joinHome        string
	joinMoniker     string
	joinChainID     string
	joinStateSync   bool
	joinPrune       string
	joinConfigFile  string
	joinAutoStart   bool
	joinSnapshotInt int64
)

func init() {
	joinCmd.Flags().StringVar(&joinRPCNode, "rpc-node", "", "RPC node address (required)")
	joinCmd.Flags().StringVar(&joinHome, "home", "/sekai", "sekaid home directory")
	joinCmd.Flags().StringVar(&joinMoniker, "moniker", "node", "Node moniker")
	joinCmd.Flags().StringVar(&joinChainID, "chain-id", "", "Chain ID (auto-detect if empty)")
	joinCmd.Flags().BoolVar(&joinStateSync, "statesync", false, "Enable state sync")
	joinCmd.Flags().StringVar(&joinPrune, "prune", "default", "Pruning mode: default|nothing|everything|custom")
	joinCmd.Flags().StringVar(&joinConfigFile, "config", "", "Path to scall.toml for additional overrides")
	joinCmd.Flags().BoolVar(&joinAutoStart, "start", true, "Auto-start sekaid after join")
	joinCmd.Flags().Int64Var(&joinSnapshotInt, "snapshot-interval", 1000, "Snapshot interval for statesync trust height calculation")

	joinCmd.MarkFlagRequired("rpc-node")
}

func runJoin(cmd *cobra.Command, args []string) {
	// 1. Read mnemonic from stdin
	Log("Reading mnemonic from stdin...")
	mnemonic, err := readMnemonicFromStdin()
	if err != nil {
		Fatal("Failed to read mnemonic: %v", err)
	}

	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		Fatal("Invalid mnemonic")
	}
	Log("Mnemonic validated")

	// 2. Initialize sekaid
	Log("Initializing sekaid...")
	if err := initSekaid(joinHome, joinChainID, joinMoniker); err != nil {
		Fatal("Failed to initialize sekaid: %v", err)
	}

	// 3. Add validator key from mnemonic
	Log("Adding validator key...")
	if err := addValidatorKey(joinHome, mnemonic); err != nil {
		Fatal("Failed to add validator key: %v", err)
	}

	// 4. Fetch genesis
	Log("Fetching genesis from %s...", joinRPCNode)
	genesisPath := filepath.Join(joinHome, "config", "genesis.json")
	if err := genesis.Fetch(joinRPCNode, genesisPath); err != nil {
		Fatal("Failed to fetch genesis: %v", err)
	}
	Log("Genesis saved to %s", genesisPath)

	// 5. Build scall config with overrides
	scall := make(config.ScallConfig)

	// Set RPC node as seed (fetches node ID and formats as nodeID@ip:p2pPort)
	Log("Fetching seed node info...")
	seedAddr := formatSeed(joinRPCNode)
	Log("Using seed: %s", seedAddr)
	scall.SetValue("config.p2p.seeds", seedAddr)

	// 6. Configure statesync if enabled
	if joinStateSync {
		Log("Configuring statesync...")
		ssConfig, err := statesync.FetchConfig(joinRPCNode, joinSnapshotInt)
		if err != nil {
			Fatal("Failed to fetch statesync config: %v", err)
		}
		scall.SetValue("config.statesync.enable", true)
		scall.SetValue("config.statesync.rpc_servers", ssConfig.RPCServers)
		scall.SetValue("config.statesync.trust_height", ssConfig.TrustHeight)
		scall.SetValue("config.statesync.trust_hash", ssConfig.TrustHash)
		Log("Statesync configured: height=%d hash=%s", ssConfig.TrustHeight, ssConfig.TrustHash)
	}

	// 7. Configure pruning
	applyPruningConfig(scall, joinPrune)

	// 8. Load additional overrides from scall.toml if provided
	if joinConfigFile != "" {
		Log("Loading config overrides from %s...", joinConfigFile)
		fileConfig, err := config.LoadScall(joinConfigFile)
		if err != nil {
			Fatal("Failed to load config file: %v", err)
		}
		// Merge file config into scall
		for k, v := range fileConfig {
			scall.SetValue(k, v)
		}
	}

	// 9. Apply config overrides
	configTomlPath := filepath.Join(joinHome, "config", "config.toml")
	appTomlPath := filepath.Join(joinHome, "config", "app.toml")

	Log("Applying config overrides...")
	if err := scall.ApplyToConfigToml(configTomlPath); err != nil {
		Fatal("Failed to apply config.toml overrides: %v", err)
	}
	if err := scall.ApplyToAppToml(appTomlPath); err != nil {
		Fatal("Failed to apply app.toml overrides: %v", err)
	}

	Log("Node configured successfully")

	// 10. Start if requested
	if joinAutoStart {
		Log("Starting sekaid...")
		runStart(nil, nil)
	} else {
		Log("Join complete. Run 'scaller start' to start the node.")
	}
}

func readMnemonicFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func initSekaid(home, chainID, moniker string) error {
	args := []string{"init", moniker, "--home", home}
	if chainID != "" {
		args = append(args, "--chain-id", chainID)
	}
	args = append(args, "--overwrite")

	cmd := exec.Command("/sekaid", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

func addValidatorKey(home, mnemonic string) error {
	Log("Mnemonic length: %d words", len(strings.Fields(mnemonic)))

	// Write mnemonic to temp file to avoid stdin pipe issues
	tmpFile := "/tmp/mnemonic.tmp"
	mnemonicData := []byte(mnemonic + "\n")
	if err := os.WriteFile(tmpFile, mnemonicData, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %v", err)
	}
	defer secureDelete(tmpFile, len(mnemonicData))

	// Use shell to pipe file content to sekaid
	shellCmd := fmt.Sprintf("cat %s | /sekaid keys add validator --home %s --keyring-backend test --recover", tmpFile, home)
	cmd := exec.Command("/bin/sh", "-c", shellCmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	Log("Key added: %s", string(output))
	return nil
}

// secureDelete overwrites file with zeros before removing it
func secureDelete(path string, size int) {
	// Overwrite with zeros
	zeros := make([]byte, size)
	if f, err := os.OpenFile(path, os.O_WRONLY, 0600); err == nil {
		f.Write(zeros)
		f.Sync()
		f.Close()
	}
	os.Remove(path)
}

// statusNodeResponse represents the RPC /status response for node info
type statusNodeResponse struct {
	Result struct {
		NodeInfo struct {
			ID string `json:"id"`
		} `json:"node_info"`
	} `json:"result"`
}

// fetchNodeID fetches the node ID from an RPC endpoint
func fetchNodeID(rpcNode string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Ensure rpcNode has scheme
	url := rpcNode
	if !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	resp, err := client.Get(url + "/status")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var status statusNodeResponse
	if err := json.Unmarshal(body, &status); err != nil {
		return "", err
	}

	if status.Result.NodeInfo.ID == "" {
		return "", fmt.Errorf("empty node ID in response")
	}

	return status.Result.NodeInfo.ID, nil
}

// formatSeed formats the RPC node address as a proper seed address
// Input: ip:rpcPort (e.g., 3.123.154.245:26657)
// Output: nodeID@ip:p2pPort (e.g., abc123@3.123.154.245:26656)
func formatSeed(rpcNode string) string {
	// Fetch node ID
	nodeID, err := fetchNodeID(rpcNode)
	if err != nil {
		Log("Warning: Failed to fetch node ID: %v, using address only", err)
		return rpcNode
	}

	// Extract IP from rpcNode (remove port)
	ip := rpcNode
	if idx := strings.LastIndex(rpcNode, ":"); idx != -1 {
		ip = rpcNode[:idx]
	}

	// P2P port is typically 26656 (RPC is 26657)
	p2pPort := "26656"

	return fmt.Sprintf("%s@%s:%s", nodeID, ip, p2pPort)
}

func applyPruningConfig(scall config.ScallConfig, mode string) {
	switch mode {
	case "nothing":
		scall.SetValue("app.pruning", "nothing")
	case "everything":
		scall.SetValue("app.pruning", "everything")
	case "custom":
		scall.SetValue("app.pruning", "custom")
		scall.SetValue("app.pruning-keep-recent", "100")
		scall.SetValue("app.pruning-interval", "10")
	default: // "default"
		scall.SetValue("app.pruning", "default")
	}
}
