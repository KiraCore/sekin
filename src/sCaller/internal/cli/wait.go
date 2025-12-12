package cli

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var waitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait indefinitely (container entrypoint)",
	Long:  `Waits for signals. Use as container entrypoint to keep container alive for docker exec commands.`,
	Run:   runWait,
}

func runWait(cmd *cobra.Command, args []string) {
	Log("scaller waiting for commands... (use docker exec to run join/start)")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	Log("Received signal %v, shutting down", sig)
}
