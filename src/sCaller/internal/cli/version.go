package cli

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
)

var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run:   runVersion,
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("scaller version: %s\n", Version)

	// Also show sekaid version if available
	sekaidCmd := exec.Command("/sekaid", "version")
	output, err := sekaidCmd.CombinedOutput()
	if err == nil {
		fmt.Printf("sekaid version: %s", string(output))
	}
}
