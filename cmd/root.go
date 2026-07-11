package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "poly",
	Short: "poly is one command for every package manager",
	Long:  "poly installs, removes, and tracks packages from pip, npm, and more behind a single unified command.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		switch cmd.Name() {
		case "self-update", "upgrade":
			return // avoid a background check re-triggering itself
		}
		maybeAutoUpdate()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
