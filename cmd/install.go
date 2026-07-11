package cmd

import (
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"poly/internal/account"
	"poly/internal/adapters"
	"poly/internal/lockfile"
	"poly/internal/manifest"
	"poly/internal/ui"
)

type installResult struct {
	spec             string
	adapterPrefix    string
	name             string
	version          string
	a                adapters.Adapter
	installedVersion string
	err              error
}

func installOne(spec string) installResult {
	adapterPrefix, name, version := parseSpec(spec)
	r := installResult{spec: spec, adapterPrefix: adapterPrefix, name: name, version: version}

	a, err := resolveAdapter(adapterPrefix, name)
	if err != nil {
		r.err = err
		return r
	}
	r.a = a

	installedVersion, err := a.Install(name, version)
	r.installedVersion = installedVersion
	r.err = err
	return r
}

func installSequential(specs []string) []installResult {
	results := make([]installResult, len(specs))
	for i, spec := range specs {
		results[i] = installOne(spec)
	}
	return results
}

// installParallel runs installs concurrently -- a Pro perk. Adapters write
// their own progress straight to stdout/stderr, so output from
// simultaneous installs can interleave; that's a known rough edge we
// accept in exchange for the real wall-clock speedup.
func installParallel(specs []string) []installResult {
	results := make([]installResult, len(specs))
	var wg sync.WaitGroup
	for i, spec := range specs {
		wg.Add(1)
		go func(i int, spec string) {
			defer wg.Done()
			results[i] = installOne(spec)
		}(i, spec)
	}
	wg.Wait()
	return results
}

var installCmd = &cobra.Command{
	Use:   "install [[adapter:]package[@version] ...]",
	Short: "Install one or more packages, or everything listed in poly.json",
	Long: `Install one or more packages.

Examples:
  poly install ripgrep          # auto-detected: tap, then pip, then npm
  poly install requests@2.31.0  # pinned version
  poly install pip:requests     # force the pip adapter
  poly install npm:lodash       # force the npm adapter
  poly install tap:ripgrep      # force a direct binary download
  poly install ripgrep requests npm:lodash   # multiple packages; Pro installs them in parallel
  poly install                  # no args: installs everything listed in ./poly.json`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			f, found, err := lockfile.Load()
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("no packages given and no %s in this directory (run `poly init` to create one)", lockfile.FileName)
			}
			if len(f.Packages) == 0 {
				fmt.Println(ui.Dim(lockfile.FileName + " has no packages"))
				return nil
			}
			args = f.Packages
			fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("installing %d package(s) from %s", len(args), lockfile.FileName)))
		}

		var results []installResult
		if len(args) > 1 {
			if account.IsPro() {
				fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("installing %d packages in parallel (pro)", len(args))))
				results = installParallel(args)
			} else {
				results = installSequential(args)
				fmt.Println(ui.Dim("note: poly pro installs multiple packages in parallel — see the site's Pro section"))
			}
		} else {
			results = installSequential(args)
		}

		m, err := manifest.Load()
		if err != nil {
			return err
		}

		var firstErr error
		usedTap := false
		for _, r := range results {
			if r.err != nil {
				fmt.Println(ui.Red(fmt.Sprintf("failed to install %s: %v", r.spec, r.err)))
				if firstErr == nil {
					firstErr = r.err
				}
				continue
			}

			m.Add(manifest.Entry{
				Name:        r.name,
				Adapter:     r.a.Name(),
				Version:     r.installedVersion,
				InstalledAt: time.Now(),
			})
			fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("installed %s %s (via %s)", r.name, r.installedVersion, r.a.Name())))
			if r.a.Name() == "tap" {
				usedTap = true
			}
		}

		if err := m.Save(); err != nil {
			return err
		}

		if usedTap {
			binDir, err := adapters.BinDir()
			if err == nil {
				fmt.Println(ui.Dim(fmt.Sprintf("note: tap binaries are installed to %s — make sure it's on your PATH", binDir)))
			}
		}

		return firstErr
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
