package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show node and network status",
	Long:  `Displays a concise status table showing sekai, interx, and network health.`,
	Run:   runStatus,
}

var (
	statusRPCAddr   string
	statusInterxAddr string
)

func init() {
	statusCmd.Flags().StringVar(&statusRPCAddr, "rpc", "http://localhost:26657", "Sekai RPC address")
	statusCmd.Flags().StringVar(&statusInterxAddr, "interx", "http://proxy.local:8080", "Interx address")
}

// Status check results
type statusResult struct {
	Name   string
	Status string
	Detail string
}

func runStatus(cmd *cobra.Command, args []string) {
	results := []statusResult{}

	// Check Sekai RPC
	sekaiStatus, sekaiDetail := checkSekai(statusRPCAddr)
	results = append(results, statusResult{"Sekai", sekaiStatus, sekaiDetail})

	// Check Interx
	interxStatus, interxDetail := checkInterx(statusInterxAddr)
	results = append(results, statusResult{"Interx", interxStatus, interxDetail})

	// Get network info from Sekai
	netStatus := getNetworkStatus(statusRPCAddr)
	results = append(results, netStatus...)

	// Print table
	printStatusTable(results)
}

func checkSekai(rpcAddr string) (string, string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rpcAddr + "/status")
	if err != nil {
		return "DOWN", err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "ERROR", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "ERROR", "Failed to read response"
	}

	var status struct {
		Result struct {
			SyncInfo struct {
				CatchingUp         bool   `json:"catching_up"`
				LatestBlockHeight  string `json:"latest_block_height"`
				LatestBlockTime    string `json:"latest_block_time"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &status); err != nil {
		return "ERROR", "Invalid response"
	}

	if status.Result.SyncInfo.CatchingUp {
		return "SYNCING", fmt.Sprintf("height %s", status.Result.SyncInfo.LatestBlockHeight)
	}

	return "OK", fmt.Sprintf("height %s", status.Result.SyncInfo.LatestBlockHeight)
}

func checkInterx(addr string) (string, string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(addr + "/api/status")
	if err != nil {
		return "DOWN", err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "ERROR", fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return "OK", "responding"
}

func getNetworkStatus(rpcAddr string) []statusResult {
	results := []statusResult{}
	client := &http.Client{Timeout: 5 * time.Second}

	// Get net_info for peers
	resp, err := client.Get(rpcAddr + "/net_info")
	if err != nil {
		results = append(results, statusResult{"Peers", "N/A", "cannot fetch"})
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var netInfo struct {
			Result struct {
				NPeers string `json:"n_peers"`
				Peers  []struct {
					NodeInfo struct {
						Moniker string `json:"moniker"`
					} `json:"node_info"`
				} `json:"peers"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &netInfo); err == nil {
			peerCount := netInfo.Result.NPeers
			if peerCount == "" {
				peerCount = fmt.Sprintf("%d", len(netInfo.Result.Peers))
			}
			status := "OK"
			if peerCount == "0" {
				status = "WARN"
			}
			results = append(results, statusResult{"Peers", status, fmt.Sprintf("%s connected", peerCount)})
		}
	}

	// Get consensus state
	resp2, err := client.Get(rpcAddr + "/status")
	if err == nil {
		defer resp2.Body.Close()
		body, _ := io.ReadAll(resp2.Body)

		var status struct {
			Result struct {
				NodeInfo struct {
					ID      string `json:"id"`
					Network string `json:"network"`
					Moniker string `json:"moniker"`
				} `json:"node_info"`
				ValidatorInfo struct {
					VotingPower string `json:"voting_power"`
				} `json:"validator_info"`
			} `json:"result"`
		}

		if err := json.Unmarshal(body, &status); err == nil {
			results = append(results, statusResult{"Node ID", "INFO", status.Result.NodeInfo.ID})
			results = append(results, statusResult{"Chain", "INFO", status.Result.NodeInfo.Network})
			results = append(results, statusResult{"Moniker", "INFO", status.Result.NodeInfo.Moniker})

			vp := status.Result.ValidatorInfo.VotingPower
			if vp != "" && vp != "0" {
				results = append(results, statusResult{"Validator", "OK", fmt.Sprintf("power %s", vp)})
			} else {
				results = append(results, statusResult{"Validator", "INFO", "not active"})
			}
		}
	}

	return results
}

func printStatusTable(results []statusResult) {
	// Calculate column widths
	maxName := 10
	maxStatus := 7
	for _, r := range results {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
		if len(r.Status) > maxStatus {
			maxStatus = len(r.Status)
		}
	}

	// Print header
	fmt.Printf("\n%-*s  %-*s  %s\n", maxName, "SERVICE", maxStatus, "STATUS", "DETAIL")
	fmt.Printf("%s  %s  %s\n", repeat("-", maxName), repeat("-", maxStatus), repeat("-", 30))

	// Print rows
	for _, r := range results {
		statusIcon := getStatusIcon(r.Status)
		fmt.Printf("%-*s  %s %-*s  %s\n", maxName, r.Name, statusIcon, maxStatus-2, r.Status, r.Detail)
	}
	fmt.Println()
}

func getStatusIcon(status string) string {
	switch status {
	case "OK":
		return "[+]"
	case "DOWN", "ERROR":
		return "[X]"
	case "WARN", "SYNCING":
		return "[!]"
	default:
		return "[ ]"
	}
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
