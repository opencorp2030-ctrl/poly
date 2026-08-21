package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/lockfile"
	"poly/internal/manifest"
)

var addCmd = &cobra.Command{
	Use:   "add [adapter:]package[@version] ...",
	Short: "Install one or more packages and add them to poly.json",
	Long: `Like "poly install", but also records what you installed in ./poly.json
(creating it if it doesn't exist yet) and pins the exact resolved
version -- plus, for tap/community packages, the checksum and source
URL -- in ./poly.lock. Mirrors npm/yarn/cargo's "add".`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		results := installSequential(args)

		m, err := manifest.Load()
		if err != nil {
			return err
		}
		f, _, err := lockfile.Load()
		if err != nil {
			return err
		}
		if f == nil {
			f = &lockfile.File{}
		}

		firstErr := recordResults(m, results, "added")

		for _, r := range results {
			if r.err != nil {
				continue
			}
			spec := fmt.Sprintf("%s:%s@%s", r.a.Name(), r.name, r.installedVersion)
			f.Packages = addOrReplaceSpec(f.Packages, r.name, spec)
		}

		if err := m.Save(); err != nil {
			return err
		}
		if err := lockfile.Save(f); err != nil {
			return err
		}
		if err := updateLock(results); err != nil {
			return err
		}

		return firstErr
	},
}

// addOrReplaceSpec inserts newSpec into specs, replacing any existing
// entry for the same package name so running "poly add" again for a
// package updates its pinned version in place instead of duplicating it.
func addOrReplaceSpec(specs []string, name, newSpec string) []string {
	for i, s := range specs {
		_, n, _ := parseSpec(s)
		if n == name {
			specs[i] = newSpec
			return specs
		}
	}
	return append(specs, newSpec)
}

func init() {
	rootCmd.AddCommand(addCmd)
}
