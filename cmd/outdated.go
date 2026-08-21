package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"poly/internal/manifest"
	"poly/internal/ui"
)

// outdatedPackage is one row of the --json output of "poly outdated".
type outdatedPackage struct {
	Name    string `json:"name"`
	Current string `json:"current"`
	Latest  string `json:"latest"`
	Adapter string `json:"adapter"`
}

var outdatedJSON bool

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "List installed packages that have a newer version available",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := manifest.Load()
		if err != nil {
			return err
		}
		if len(m.Packages) == 0 {
			if outdatedJSON {
				return printJSON([]outdatedPackage{})
			}
			fmt.Println(ui.Dim("no packages installed via poly yet"))
			return nil
		}

		rows, _, _ := findOutdated(m)

		if outdatedJSON {
			out := make([]outdatedPackage, 0, len(rows))
			for _, r := range rows {
				out = append(out, outdatedPackage{Name: r.name, Current: r.current, Latest: r.latest, Adapter: r.adapter})
			}
			return printJSON(out)
		}

		if len(rows) == 0 {
			fmt.Println(ui.Orange("everything is up to date"))
			return nil
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("%d package(s) outdated", len(rows))))
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "PACKAGE\tCURRENT\tLATEST\tADAPTER")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.name, r.current, r.latest, r.adapter)
		}
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Println(ui.Dim("run `poly upgrade` to update them"))
		return nil
	},
}

func init() {
	outdatedCmd.Flags().BoolVar(&outdatedJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(outdatedCmd)
}
