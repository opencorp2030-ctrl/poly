package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"poly/internal/account"
)

// Version is injected at build time via -ldflags "-X poly/cmd.Version=x.y.z".
// It defaults to "dev" for local builds.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the poly version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("poly %s (%s/%s)\n", Version, runtime.GOOS, runtime.GOARCH)
		if email := account.Email(); email != "" {
			if account.IsPro() {
				fmt.Printf("signed in as %s — pro ✓\n", email)
			} else {
				fmt.Printf("signed in as %s\n", email)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
