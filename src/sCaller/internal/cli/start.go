package cli

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const sekaidPath = "/sekaid"

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start sekaid",
	Long: `Starts sekaid. By default uses syscall.Exec to replace this process.

With --restart flag, runs sekaid as a subprocess and restarts on failure.

Examples:
  scaller start                    # Start once (replaces process)
  scaller start --restart 5        # Restart up to 5 times on failure
  scaller start --restart always   # Restart indefinitely (max 10 retries)`,
	Run: runStart,
}

var (
	startHome    string
	startRestart string
)

func init() {
	startCmd.Flags().StringVar(&startHome, "home", "/sekai", "sekaid home directory")
	startCmd.Flags().StringVar(&startRestart, "restart", "", "Restart on failure: number (1-10) or 'always' (max 10)")
}

func runStart(cmd *cobra.Command, args []string) {
	Log("Starting sekaid with home=%s", startHome)

	// If restart is not set, use syscall.Exec (original behavior)
	if startRestart == "" {
		execSekaid()
		return
	}

	// Parse restart mode
	maxRetries := parseRestartMode(startRestart)
	Log("Restart mode enabled: max %d retries", maxRetries)

	runWithRestart(maxRetries)
}

// execSekaid replaces the current process with sekaid
func execSekaid() {
	argv := []string{"sekaid", "start", "--home", startHome}
	env := os.Environ()

	err := syscall.Exec(sekaidPath, argv, env)
	if err != nil {
		Fatal("Failed to exec sekaid: %v", err)
	}
}

// parseRestartMode parses the restart flag value
func parseRestartMode(mode string) int {
	if mode == "always" {
		return 10 // "always" means max 10 retries
	}

	// Parse as number
	var n int
	_, err := fmt.Sscanf(mode, "%d", &n)
	if err != nil || n < 1 {
		Fatal("Invalid restart value: %s (use 1-10 or 'always')", mode)
	}
	if n > 10 {
		n = 10
	}
	return n
}

// runWithRestart runs sekaid as a subprocess with restart logic
func runWithRestart(maxRetries int) {
	retryCount := 0
	backoffSeconds := []int{1, 2, 5, 10, 15, 30, 30, 30, 30, 30} // Progressive backoff

	for {
		Log("Starting sekaid (attempt %d/%d)...", retryCount+1, maxRetries)

		cmd := exec.Command(sekaidPath, "start", "--home", startHome)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		startTime := time.Now()
		err := cmd.Run()
		runDuration := time.Since(startTime)

		if err == nil {
			Log("sekaid exited normally")
			return
		}

		retryCount++
		Log("sekaid exited with error: %v (ran for %v)", err, runDuration)

		// If it ran for more than 60 seconds, reset retry count (was stable)
		if runDuration > 60*time.Second {
			Log("Node was stable, resetting retry count")
			retryCount = 1
		}

		if retryCount >= maxRetries {
			Fatal("Max retries (%d) exceeded, giving up", maxRetries)
		}

		// Backoff before retry
		backoff := backoffSeconds[retryCount-1]
		Log("Waiting %d seconds before retry...", backoff)
		time.Sleep(time.Duration(backoff) * time.Second)
	}
}
