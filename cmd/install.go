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

	a, err := resolveAdapter(adapterPrefix, name, version)
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

// planInstall resolves a spec to the adapter that would install it and
// reports the version it would install, without actually installing
// anything. Only VersionLimited adapters (tap/community) know their
// exact version up front; for the others the real version only exists
// once the delegate tool runs, so installedVersion stays "".
func planInstall(spec string) (installResult, error) {
	adapterPrefix, name, version := parseSpec(spec)
	a, err := resolveAdapter(adapterPrefix, name, version)
	if err != nil {
		return installResult{}, err
	}
	r := installResult{spec: spec, adapterPrefix: adapterPrefix, name: name, version: version, a: a}
	if vl, ok := a.(adapters.VersionLimited); ok {
		if v, ok := vl.AvailableVersion(name); ok {
			r.installedVersion = v
		}
	}
	return r, nil
}

var installDryRun bool

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

		if installDryRun {
			var firstErr error
			for _, spec := range args {
				r, err := planInstall(spec)
				if err != nil {
					fmt.Println(ui.Red(fmt.Sprintf("would not be able to install %s: %v", spec, err)))
					if firstErr == nil {
						firstErr = err
					}
					continue
				}
				v := r.installedVersion
				if v == "" {
					v = "latest"
				}
				fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("would install %s %s (via %s)", r.name, v, r.a.Name())))
			}
			if firstErr == nil {
				fmt.Println(ui.Dim("dry run: nothing was installed"))
			}
			return firstErr
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
		firstErr := recordResults(m, results, "installed")

		if err := m.Save(); err != nil {
			return err
		}

		usedTap := false
		for _, r := range results {
			if r.err == nil && r.a.Name() == "tap" {
				usedTap = true
				break
			}
		}
		if usedTap {
			binDir, err := adapters.BinDir()
			if err == nil {
				fmt.Println(ui.Dim(fmt.Sprintf("note: tap binaries are installed to %s — make sure it's on your PATH", binDir)))
			}
		}

		if err := updateLock(results); err != nil {
			return err
		}

		return firstErr
	},
}

// recordResults reports each install result (success or failure) and
// records successful installs into the manifest. It returns the first
// error encountered, if any, without aborting the remaining results --
// so a failing package doesn't prevent the others from being recorded.
func recordResults(m *manifest.Manifest, results []installResult, verb string) error {
	var firstErr error
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
		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("%s %s %s (via %s)", verb, r.name, r.installedVersion, r.a.Name())))
	}
	return firstErr
}

// updateLock records exact resolved versions (and, for tap/community,
// the checksum and source URL) for successful results into poly.lock --
// but only if this directory is already a poly project (has a
// poly.json). A bare `poly install somepkg` outside a project doesn't
// spontaneously create project files.
func updateLock(results []installResult) error {
	if !lockfile.Exists() {
		return nil
	}
	l, _, err := lockfile.LoadLock()
	if err != nil {
		return err
	}
	for _, r := range results {
		if r.err != nil {
			continue
		}
		entry := lockfile.LockEntry{Adapter: r.a.Name(), Version: r.installedVersion}
		switch r.a.Name() {
		case "tap":
			if url, sha, ok := adapters.ArtifactInfo(r.name); ok {
				entry.URL, entry.SHA256 = url, sha
			}
		case "community":
			if url, sha, ok := adapters.CommunityArtifactInfo(r.name); ok {
				entry.URL, entry.SHA256 = url, sha
			}
		}
		l.Packages[r.name] = entry
	}
	return lockfile.SaveLock(l)
}

func init() {
	installCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "show what would be installed without installing anything")
	rootCmd.AddCommand(installCmd)
}
