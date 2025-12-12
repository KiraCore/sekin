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
  wait   - Wait indefinitely (container entrypoint)
  join   - Initialize node and join network
  start  - Start sekaid (replaces this process)`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(waitCmd)
	rootCmd.AddCommand(joinCmd)
	rootCmd.AddCommand(startCmd)
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
