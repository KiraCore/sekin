package cli

import (
	"os"
	"syscall"

	"github.com/spf13/cobra"
)

const sekaidPath = "/sekaid"

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start sekaid (replaces this process)",
	Long:  `Starts sekaid using syscall.Exec, replacing this process entirely.`,
	Run:   runStart,
}

var startHome string

func init() {
	startCmd.Flags().StringVar(&startHome, "home", "/sekai", "sekaid home directory")
}

func runStart(cmd *cobra.Command, args []string) {
	Log("Starting sekaid with home=%s", startHome)

	argv := []string{"sekaid", "start", "--home", startHome}
	env := os.Environ()

	// This replaces the current process with sekaid
	err := syscall.Exec(sekaidPath, argv, env)
	if err != nil {
		Fatal("Failed to exec sekaid: %v", err)
	}

	// This line is never reached if Exec succeeds
}
