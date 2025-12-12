package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize sekaid node",
	Long: `Initializes a new sekaid node with the given chain-id and moniker.

Example:
  scaller init --chain-id testnet-1 --moniker "My Node"`,
	Run: runInit,
}

var (
	initHome    string
	initChainID string
	initMoniker string
)

func init() {
	initCmd.Flags().StringVar(&initHome, "home", "/sekai", "sekaid home directory")
	initCmd.Flags().StringVar(&initChainID, "chain-id", "testnet-1", "Chain ID")
	initCmd.Flags().StringVar(&initMoniker, "moniker", "Genesis", "Node moniker")
}

func runInit(cmd *cobra.Command, args []string) {
	Log("Initializing sekaid with chain-id: %s, moniker: %s", initChainID, initMoniker)

	sekaidCmd := exec.Command("/sekaid", "init", initMoniker,
		"--chain-id", initChainID,
		"--home", initHome,
		"--overwrite")

	output, err := sekaidCmd.CombinedOutput()
	if err != nil {
		Fatal("Failed to initialize sekaid: %v\n%s", err, string(output))
	}

	Log("Sekaid initialized successfully")
	fmt.Print(string(output))
}
