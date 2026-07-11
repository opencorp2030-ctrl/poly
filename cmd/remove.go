package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/manifest"
)

var removeCmd = &cobra.Command{
	Use:     "remove [package]",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove an installed package",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, name, _ := parseSpec(args[0])

		m, err := manifest.Load()
		if err != nil {
			return err
		}
		entry, ok := m.Get(name)
		if !ok {
			return fmt.Errorf("%s is not tracked by poly (not installed via poly, or already removed)", name)
		}

		a, ok := adapters.ByName(entry.Adapter)
		if !ok {
			return fmt.Errorf("unknown adapter %q recorded for package %s", entry.Adapter, name)
		}
		if err := a.Remove(name); err != nil {
			return err
		}

		m.Remove(name)
		if err := m.Save(); err != nil {
			return err
		}

		fmt.Printf("removed %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
