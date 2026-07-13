package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/manifest"
	"poly/internal/ui"
)

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "List installed packages that have a newer version available",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := manifest.Load()
		if err != nil {
			return err
		}
		if len(m.Packages) == 0 {
			fmt.Println(ui.Dim("no packages installed via poly yet"))
			return nil
		}

		type row struct{ name, current, latest, adapter string }
		var rows []row

		names := make([]string, 0, len(m.Packages))
		for name := range m.Packages {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			e := m.Packages[name]
			a, ok := adapters.ByName(e.Adapter)
			if !ok {
				continue
			}
			result, err := a.Search(name)
			if err != nil || !result.Found {
				continue
			}
			if result.Version != e.Version {
				rows = append(rows, row{name, e.Version, result.Version, e.Adapter})
			}
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
	rootCmd.AddCommand(outdatedCmd)
}
