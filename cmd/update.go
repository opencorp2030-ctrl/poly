package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/selfupdate"
	"poly/internal/ui"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update poly itself, then update installed packages",
	Long: `Shorthand for the two commands people instinctively reach for:
self-update (poly itself) followed by upgrade (everything poly installed).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		latest, hasUpdate, err := selfupdate.Check(Version)
		if err != nil {
			fmt.Println(ui.Dim("could not check for a poly update: " + err.Error()))
		} else if hasUpdate {
			fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("updating poly %s → %s", Version, latest)))
			if newVersion, err := selfupdate.Apply(); err != nil {
				fmt.Println(ui.Red("failed to update poly: " + err.Error()))
			} else {
				fmt.Println(ui.Orange("poly updated to " + newVersion))
			}
		} else {
			fmt.Println(ui.Dim("poly is already up to date (" + Version + ")"))
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange("updating installed packages..."))
		return runUpgrade()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
