package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/account"
	"poly/internal/ui"
)

var sendDescription string

var sendCmd = &cobra.Command{
	Use:   "send name version path",
	Short: "Publish your own package to the Poly community registry",
	Long: `Publish a file or directory to Poly's community registry under your account.

path can be a single binary/archive, or a directory (it gets tar.gz'd
automatically) -- either way it must contain (or be) a file named exactly
"name" (".exe" appended on Windows), which is what gets installed.

Requires "poly login". Names are first-come-first-served: only the
account that first published a name can push a new version over it.
Community packages are never auto-detected -- anyone installing yours
does it explicitly with:

  poly install community:name`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, version, path := args[0], args[1], args[2]

		if account.Email() == "" {
			return fmt.Errorf("not signed in — run `poly login` first")
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("publishing %s@%s from %s", name, version, path)))

		if err := account.PublishPackage(name, version, path, sendDescription); err != nil {
			return err
		}

		fmt.Println(ui.Orange(fmt.Sprintf("published — anyone can now install it with: poly install community:%s", name)))
		return nil
	},
}

func init() {
	sendCmd.Flags().StringVar(&sendDescription, "description", "", "short description shown in search results")
	rootCmd.AddCommand(sendCmd)
}
