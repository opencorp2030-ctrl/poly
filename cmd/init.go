package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"poly/internal/lockfile"
	"poly/internal/manifest"
	"poly/internal/ui"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write poly.json from your currently poly-installed packages",
	Long: `Write poly.json in the current directory, listing every package poly
has installed (pinned to its installed version) so the environment can be
reproduced elsewhere with a plain "poly install".`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if lockfile.Exists() && !initForce {
			return fmt.Errorf("%s already exists (use --force to overwrite)", lockfile.FileName)
		}

		m, err := manifest.Load()
		if err != nil {
			return err
		}

		names := make([]string, 0, len(m.Packages))
		for name := range m.Packages {
			names = append(names, name)
		}
		sort.Strings(names)

		f := &lockfile.File{Packages: make([]string, 0, len(names))}
		for _, name := range names {
			e := m.Packages[name]
			f.Packages = append(f.Packages, fmt.Sprintf("%s:%s@%s", e.Adapter, e.Name, e.Version))
		}

		if err := lockfile.Save(f); err != nil {
			return err
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("wrote %s with %d package(s)", lockfile.FileName, len(f.Packages))))
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite an existing poly.json")
	rootCmd.AddCommand(initCmd)
}
