package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var keysAddCmd = &cobra.Command{
	Use:   "keys-add",
	Short: "Add a new key to the keyring",
	Long: `Adds a new key to the sekaid keyring.

Example:
  scaller keys-add --name genesis`,
	Run: runKeysAdd,
}

var (
	keysAddHome    string
	keysAddName    string
	keysAddBackend string
)

func init() {
	keysAddCmd.Flags().StringVar(&keysAddHome, "home", "/sekai", "sekaid home directory")
	keysAddCmd.Flags().StringVar(&keysAddName, "name", "genesis", "Key name")
	keysAddCmd.Flags().StringVar(&keysAddBackend, "keyring-backend", "test", "Keyring backend (test|file|os)")
}

func runKeysAdd(cmd *cobra.Command, args []string) {
	Log("Adding key '%s' with keyring backend '%s'", keysAddName, keysAddBackend)

	sekaidCmd := exec.Command("/sekaid", "keys", "add", keysAddName,
		"--keyring-backend", keysAddBackend,
		"--home", keysAddHome)

	output, err := sekaidCmd.CombinedOutput()
	if err != nil {
		Fatal("Failed to add key: %v\n%s", err, string(output))
	}

	Log("Key added successfully")
	fmt.Print(string(output))
}
