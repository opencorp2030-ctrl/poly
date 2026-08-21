package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/manifest"
	"poly/internal/ui"
	"poly/internal/version"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Update installed packages to their latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgrade()
	},
}

var upgradeDryRun bool

// outdatedRow is a package with a strictly newer version available.
type outdatedRow struct {
	name    string
	adapter string
	current string
	latest  string
}

// findOutdated checks every installed package against its adapter's
// latest known version. It returns the outdated packages (sorted by
// name) plus the number of packages that were already current and the
// number that could not be checked (unknown adapter, or the adapter
// failed to determine a latest version).
func findOutdated(m *manifest.Manifest) (rows []outdatedRow, upToDate, skipped int) {
	for name, e := range m.Packages {
		a, ok := adapters.ByName(e.Adapter)
		if !ok {
			skipped++
			continue
		}
		result, err := a.Search(name)
		if err != nil || !result.Found {
			skipped++
			continue
		}
		// Only a strictly newer available version counts as outdated:
		// a newer-than-latest installed version (e.g. a pre-release)
		// should not be reported as something to "fix".
		if !version.Newer(result.Version, e.Version) {
			upToDate++
			continue
		}
		rows = append(rows, outdatedRow{name, e.Adapter, e.Version, result.Version})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].name < rows[j].name })
	return rows, upToDate, skipped
}

// runUpgrade checks every package poly has installed against its
// adapter's latest known version, and reinstalls whatever is outdated.
func runUpgrade() error {
	m, err := manifest.Load()
	if err != nil {
		return err
	}
	if len(m.Packages) == 0 {
		fmt.Println(ui.Dim("no packages installed via poly"))
		return nil
	}

	outdated, upToDate, skipped := findOutdated(m)

	if upgradeDryRun {
		if len(outdated) == 0 {
			fmt.Println(ui.Orange("everything is up to date"))
			return nil
		}
		for _, row := range outdated {
			e := m.Packages[row.name]
			fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("would upgrade %s %s → %s", row.name, e.Version, row.latest)))
		}
		fmt.Println(ui.Dim("dry run: nothing was changed"))
		return nil
	}

	var upgraded int
	for _, row := range outdated {
		e := m.Packages[row.name]
		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("upgrading %s %s → %s", row.name, e.Version, row.latest)))
		a, ok := adapters.ByName(row.adapter)
		if !ok {
			skipped++
			continue
		}
		installedVersion, err := a.Install(row.name, "")
		if err != nil {
			fmt.Println(ui.Red(fmt.Sprintf("failed to upgrade %s: %v", row.name, err)))
			skipped++
			continue
		}

		e.Version = installedVersion
		e.InstalledAt = time.Now()
		m.Add(e)
		upgraded++
	}

	if err := m.Save(); err != nil {
		return err
	}

	fmt.Printf("%s %d upgraded, %d up to date, %d skipped\n", ui.Arrow(), upgraded, upToDate, skipped)
	return nil
}

func init() {
	upgradeCmd.Flags().BoolVar(&upgradeDryRun, "dry-run", false, "show what would be upgraded without upgrading anything")
	rootCmd.AddCommand(upgradeCmd)
}
