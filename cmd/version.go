package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version is injected at build time via -ldflags "-X poly/cmd.Version=x.y.z".
// It defaults to "dev" for local builds.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the poly version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("poly %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
