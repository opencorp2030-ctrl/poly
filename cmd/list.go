package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"poly/internal/account"
	"poly/internal/manifest"
	"poly/internal/ui"
)

// listPackage mirrors one manifest entry for the --json output.
type listPackage struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Adapter     string    `json:"adapter"`
	InstalledAt time.Time `json:"installed_at"`
}

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages installed via poly",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := manifest.Load()
		if err != nil {
			return err
		}

		if len(m.Packages) == 0 {
			if listJSON {
				return printJSON([]listPackage{})
			}
			fmt.Println(ui.Dim("no packages installed via poly yet"))
			return nil
		}

		names := make([]string, 0, len(m.Packages))
		for name := range m.Packages {
			names = append(names, name)
		}
		sort.Strings(names)

		if listJSON {
			out := make([]listPackage, 0, len(names))
			for _, name := range names {
				e := m.Packages[name]
				out = append(out, listPackage{Name: e.Name, Version: e.Version, Adapter: e.Adapter, InstalledAt: e.InstalledAt})
			}
			return printJSON(out)
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("%d packages installed via poly", len(names))))

		// Coloring cells here would break tabwriter's column alignment
		// (it measures raw byte width, escape codes included), so the
		// table itself stays plain -- the arrow line above carries the color.
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tADAPTER\tINSTALLED")
		for _, name := range names {
			e := m.Packages[name]
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Name, e.Version, e.Adapter, e.InstalledAt.Format("2006-01-02 15:04"))
		}
		if err := w.Flush(); err != nil {
			return err
		}

		if account.IsPro() {
			fmt.Println(ui.Orange("pro ✓ " + account.Email()))
		}
		return nil
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(listCmd)
}
