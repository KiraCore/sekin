package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var addGenesisAccountCmd = &cobra.Command{
	Use:   "add-genesis-account",
	Short: "Add an account to genesis",
	Long: `Adds an account with initial token allocation to genesis.

Example:
  scaller add-genesis-account --name genesis --coins 300000000000000ukex`,
	Run: runAddGenesisAccount,
}

var gentxClaimCmd = &cobra.Command{
	Use:   "gentx-claim",
	Short: "Generate genesis transaction to claim validator role",
	Long: `Generates a genesis transaction to claim validator role.

Example:
  scaller gentx-claim --name genesis --moniker "Genesis Validator"`,
	Run: runGentxClaim,
}

var (
	// add-genesis-account flags
	addGenesisHome    string
	addGenesisName    string
	addGenesisCoins   string
	addGenesisBackend string

	// gentx-claim flags
	gentxHome    string
	gentxName    string
	gentxMoniker string
	gentxBackend string
)

func init() {
	// add-genesis-account flags
	addGenesisAccountCmd.Flags().StringVar(&addGenesisHome, "home", "/sekai", "sekaid home directory")
	addGenesisAccountCmd.Flags().StringVar(&addGenesisName, "name", "genesis", "Key name")
	addGenesisAccountCmd.Flags().StringVar(&addGenesisCoins, "coins", "300000000000000ukex", "Initial coin allocation")
	addGenesisAccountCmd.Flags().StringVar(&addGenesisBackend, "keyring-backend", "test", "Keyring backend")

	// gentx-claim flags
	gentxClaimCmd.Flags().StringVar(&gentxHome, "home", "/sekai", "sekaid home directory")
	gentxClaimCmd.Flags().StringVar(&gentxName, "name", "genesis", "Validator key name")
	gentxClaimCmd.Flags().StringVar(&gentxMoniker, "moniker", "Genesis Validator", "Validator moniker")
	gentxClaimCmd.Flags().StringVar(&gentxBackend, "keyring-backend", "test", "Keyring backend")
}

func runAddGenesisAccount(cmd *cobra.Command, args []string) {
	Log("Adding genesis account '%s' with %s", addGenesisName, addGenesisCoins)

	sekaidCmd := exec.Command("/sekaid", "add-genesis-account", addGenesisName, addGenesisCoins,
		"--keyring-backend", addGenesisBackend,
		"--home", addGenesisHome)

	output, err := sekaidCmd.CombinedOutput()
	if err != nil {
		Fatal("Failed to add genesis account: %v\n%s", err, string(output))
	}

	Log("Genesis account added successfully")
	fmt.Print(string(output))
}

func runGentxClaim(cmd *cobra.Command, args []string) {
	Log("Generating gentx-claim for '%s' as '%s'", gentxName, gentxMoniker)

	sekaidCmd := exec.Command("/sekaid", "gentx-claim", gentxName,
		"--keyring-backend", gentxBackend,
		"--moniker", gentxMoniker,
		"--home", gentxHome)

	output, err := sekaidCmd.CombinedOutput()
	if err != nil {
		Fatal("Failed to generate gentx-claim: %v\n%s", err, string(output))
	}

	Log("Genesis transaction claimed successfully")
	fmt.Print(string(output))
}
