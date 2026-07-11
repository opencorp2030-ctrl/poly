package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/manifest"
	"poly/internal/ui"
)

var infoCmd = &cobra.Command{
	Use:   "info [adapter:]package",
	Short: "Show details about a package: version, summary, homepage, and whether it's installed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		adapterPrefix, name, _ := parseSpec(args[0])

		candidates := adapters.All()
		if adapterPrefix != "" {
			a, ok := adapters.ByName(adapterPrefix)
			if !ok {
				return fmt.Errorf("unknown adapter %q", adapterPrefix)
			}
			candidates = []adapters.Adapter{a}
		}

		m, err := manifest.Load()
		if err != nil {
			return err
		}
		installed, isInstalled := m.Get(name)

		anyFound := false
		for _, a := range candidates {
			result, err := a.Search(name)
			if err != nil {
				fmt.Println(ui.Red(fmt.Sprintf("%s: lookup error: %v", a.Name(), err)))
				continue
			}
			if !result.Found {
				continue
			}
			anyFound = true

			fmt.Printf("%s %s\n", ui.Orange(name), ui.Dim("via "+a.Name()))
			fmt.Printf("  version    %s\n", result.Version)
			if result.Summary != "" {
				fmt.Printf("  summary    %s\n", result.Summary)
			}
			if result.Homepage != "" {
				fmt.Printf("  homepage   %s\n", result.Homepage)
			}
			if isInstalled && installed.Adapter == a.Name() {
				fmt.Printf("  installed  %s (%s)\n", ui.Orange(installed.Version), installed.InstalledAt.Format("2006-01-02"))
			}
			fmt.Println()
		}

		if !anyFound {
			fmt.Printf("no match for %q\n", name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
