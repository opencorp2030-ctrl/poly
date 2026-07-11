package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/account"
	"poly/internal/ui"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Sign out of your Poly account",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := account.Logout(); err != nil {
			return err
		}
		fmt.Println(ui.Orange("logged out"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
