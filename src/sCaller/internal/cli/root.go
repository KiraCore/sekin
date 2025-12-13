package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scaller",
	Short: "Sekai Caller - Bootstrap and manage sekaid",
	Long: `scaller is a CLI tool for initializing and starting sekaid.

Commands:
  wait                - Wait indefinitely (container entrypoint)
  init                - Initialize new sekaid node
  keys-add            - Add key to keyring
  add-genesis-account - Add account to genesis
  gentx-claim         - Claim validator role in genesis
  join                - Initialize node and join existing network
  start               - Start sekaid (with optional restart)
  status              - Show node and network status`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(keysAddCmd)
	rootCmd.AddCommand(addGenesisAccountCmd)
	rootCmd.AddCommand(gentxClaimCmd)
	rootCmd.AddCommand(joinCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(versionCmd)
}

// Fatal prints error and exits
func Fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

// Log prints to stderr (stdout reserved for data)
func Log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
