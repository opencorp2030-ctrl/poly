package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/account"
	"poly/internal/ui"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Show your signed-in Poly account",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := account.GetProfile()
		if err != nil {
			return err
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(profile.Email))
		if profile.Username != "" {
			fmt.Printf("  username  @%s\n", profile.Username)
		}
		fmt.Printf("  plan      %s\n", planLabel(profile.Plan))
		if profile.CreatedAt != "" {
			fmt.Printf("  member since  %s\n", profile.CreatedAt[:10])
		}
		if profile.Bio != "" {
			fmt.Printf("  bio       %s\n", profile.Bio)
		}
		return nil
	},
}

func planLabel(plan string) string {
	if plan == "pro" {
		return ui.Orange("pro ✓")
	}
	return "free"
}

func init() {
	rootCmd.AddCommand(accountCmd)
}
