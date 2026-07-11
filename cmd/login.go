package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"poly/internal/account"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in to your Poly account (unlocks Pro features)",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Email: ")
		email, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		email = strings.TrimSpace(email)

		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return err
		}

		creds, err := account.Login(email, string(passwordBytes))
		if err != nil {
			return err
		}

		fmt.Printf("logged in as %s\n", creds.Email)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}
