package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/ui"
)

var searchCmd = &cobra.Command{
	Use:   "search [adapter:]package",
	Short: "Check whether a package exists (across tap, pip, npm, or a forced adapter)",
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

		anyFound := false
		for _, a := range candidates {
			result, err := a.Search(name)
			if err != nil {
				fmt.Printf("%s: lookup error: %v\n", a.Name(), err)
				continue
			}
			if !result.Found {
				continue
			}
			anyFound = true
			fmt.Printf("%s %s\n", ui.Orange(name), ui.Dim(fmt.Sprintf("%s (%s)", result.Version, a.Name())))
			if result.Summary != "" {
				fmt.Printf("  %s\n", result.Summary)
			}
		}

		if !anyFound {
			fmt.Printf("no match for %q\n", name)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
