package cli

import (
	"fmt"

	"github.com/kiracore/sekin/src/shidai/internal/api"
	"github.com/spf13/cobra"
)

// Version variable to be set by the main package or during the build
var Version string

// NewRootCmd creates and returns the root command
func NewRootCmd(v string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "shidai",
		Short: "Shidai is an Infra manager tool.",
		Long:  `Shidai is a part of infrastructure.`,
	}

	// Add version command
	rootCmd.AddCommand(versionCmd(v))
	rootCmd.AddCommand(startCmd())

	return rootCmd
}

// versionCmd returns a version command for Cobra
func versionCmd(v string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of the application",
		Long:  `The version starts from 1.x.x and follow semver system`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(v)
		},
	}
}

// startCmd returns a version command for Cobra
func startCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "start",
		Long:  "start",
		Run: func(cmd *cobra.Command, args []string) {

			api.Serve()
		},
	}
}
