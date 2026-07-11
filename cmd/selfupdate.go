package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/selfupdate"
	"poly/internal/ui"
)

var selfUpdateCmd = &cobra.Command{
	Use:   "self-update",
	Short: "Update poly itself to the latest released version",
	RunE: func(cmd *cobra.Command, args []string) error {
		latest, hasUpdate, err := selfupdate.Check(Version)
		if err != nil {
			return err
		}
		if !hasUpdate {
			fmt.Println(ui.Dim("poly is already up to date (" + Version + ")"))
			return nil
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("updating poly %s → %s", Version, latest)))
		newVersion, err := selfupdate.Apply()
		if err != nil {
			return err
		}
		fmt.Println(ui.Orange("poly updated to " + newVersion))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(selfUpdateCmd)
}
