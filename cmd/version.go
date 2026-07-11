package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"poly/internal/account"
	"poly/internal/ui"
)

// Version is injected at build time via -ldflags "-X poly/cmd.Version=x.y.z".
// It defaults to "dev" for local builds.
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the poly version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("%s %s (%s/%s)\n", ui.Orange("poly"), Version, runtime.GOOS, runtime.GOARCH)
		if email := account.Email(); email != "" {
			if account.IsPro() {
				fmt.Println(ui.Orange("signed in as " + email + " — pro ✓"))
			} else {
				fmt.Println(ui.Dim("signed in as " + email))
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
