package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/lockfile"
	"poly/internal/manifest"
	"poly/internal/ui"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Write poly.json and poly.lock from your currently poly-installed packages",
	Long: `Write poly.json in the current directory, listing every package poly
has installed (pinned to its installed version) so the environment can be
reproduced elsewhere with a plain "poly install". Also writes poly.lock,
pinning the same packages by exact version (and, for tap/community
installs, checksum and source URL).`,
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
		l := &lockfile.Lock{Packages: make(map[string]lockfile.LockEntry, len(names))}
		for _, name := range names {
			e := m.Packages[name]
			f.Packages = append(f.Packages, fmt.Sprintf("%s:%s@%s", e.Adapter, e.Name, e.Version))

			entry := lockfile.LockEntry{Adapter: e.Adapter, Version: e.Version}
			switch e.Adapter {
			case "tap":
				if url, sha, ok := adapters.ArtifactInfo(name); ok {
					entry.URL, entry.SHA256 = url, sha
				}
			case "community":
				if url, sha, ok := adapters.CommunityArtifactInfo(name); ok {
					entry.URL, entry.SHA256 = url, sha
				}
			}
			l.Packages[name] = entry
		}

		if err := lockfile.Save(f); err != nil {
			return err
		}
		if err := lockfile.SaveLock(l); err != nil {
			return err
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("wrote %s and %s with %d package(s)", lockfile.FileName, lockfile.LockFileName, len(f.Packages))))
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite an existing poly.json")
	rootCmd.AddCommand(initCmd)
}
