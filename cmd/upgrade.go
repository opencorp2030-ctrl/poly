package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/manifest"
	"poly/internal/ui"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Update installed packages to their latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpgrade()
	},
}

// runUpgrade checks every package poly has installed against its
// adapter's latest known version, and reinstalls whatever is outdated.
// Returns the number of packages actually upgraded.
func runUpgrade() error {
	m, err := manifest.Load()
	if err != nil {
		return err
	}
	if len(m.Packages) == 0 {
		fmt.Println(ui.Dim("no packages installed via poly"))
		return nil
	}

	var upgraded, upToDate, skipped int
	for name, entry := range m.Packages {
		a, ok := adapters.ByName(entry.Adapter)
		if !ok {
			skipped++
			continue
		}

		result, err := a.Search(name)
		if err != nil || !result.Found {
			skipped++
			continue
		}
		if result.Version == entry.Version {
			upToDate++
			continue
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("upgrading %s %s → %s", name, entry.Version, result.Version)))
		installedVersion, err := a.Install(name, "")
		if err != nil {
			fmt.Println(ui.Red(fmt.Sprintf("failed to upgrade %s: %v", name, err)))
			skipped++
			continue
		}

		entry.Version = installedVersion
		entry.InstalledAt = time.Now()
		m.Add(entry)
		upgraded++
	}

	if err := m.Save(); err != nil {
		return err
	}

	fmt.Printf("%s %d upgraded, %d up to date, %d skipped\n", ui.Arrow(), upgraded, upToDate, skipped)
	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
