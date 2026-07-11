package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"poly/internal/manifest"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List packages installed via poly",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := manifest.Load()
		if err != nil {
			return err
		}

		if len(m.Packages) == 0 {
			fmt.Println("no packages installed via poly yet")
			return nil
		}

		names := make([]string, 0, len(m.Packages))
		for name := range m.Packages {
			names = append(names, name)
		}
		sort.Strings(names)

		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tADAPTER\tINSTALLED")
		for _, name := range names {
			e := m.Packages[name]
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Name, e.Version, e.Adapter, e.InstalledAt.Format("2006-01-02 15:04"))
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
