package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/lockfile"
	"poly/internal/manifest"
	"poly/internal/ui"
)

var removeCmd = &cobra.Command{
	Use:     "remove [package]",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove an installed package",
	Long: `Remove an installed package. If this directory has a poly.json/poly.lock
(e.g. you got here via "poly add"), the package is stripped from those too.`,
	Args: cobra.ExactArgs(1),
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

		if err := removeFromProjectFiles(name); err != nil {
			return err
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange("removed "+name))
		return nil
	},
}

// removeFromProjectFiles strips name from ./poly.json and ./poly.lock,
// if either exists. A no-op if this directory isn't a poly project.
func removeFromProjectFiles(name string) error {
	if lockfile.Exists() {
		f, _, err := lockfile.Load()
		if err != nil {
			return err
		}
		kept := f.Packages[:0]
		for _, s := range f.Packages {
			_, n, _ := parseSpec(s)
			if n != name {
				kept = append(kept, s)
			}
		}
		f.Packages = kept
		if err := lockfile.Save(f); err != nil {
			return err
		}
	}

	l, found, err := lockfile.LoadLock()
	if err != nil {
		return err
	}
	if found {
		delete(l.Packages, name)
		if err := lockfile.SaveLock(l); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
