package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/ui"
)

// searchMatch mirrors one adapter's hit for the --json output.
type searchMatch struct {
	Adapter string `json:"adapter"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`
}

var searchJSON bool

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

		var matches []searchMatch
		anyFound := false
		for _, a := range candidates {
			result, err := a.Search(name)
			if err != nil {
				if searchJSON {
					matches = append(matches, searchMatch{Adapter: a.Name(), Name: name, Error: err.Error()})
				} else {
					fmt.Printf("%s: lookup error: %v\n", a.Name(), err)
				}
				continue
			}
			if !result.Found {
				continue
			}
			anyFound = true
			if searchJSON {
				matches = append(matches, searchMatch{Adapter: a.Name(), Name: name, Version: result.Version, Summary: result.Summary})
				continue
			}
			fmt.Printf("%s %s\n", ui.Orange(name), ui.Dim(fmt.Sprintf("%s (%s)", result.Version, a.Name())))
			if result.Summary != "" {
				fmt.Printf("  %s\n", result.Summary)
			}
		}

		if searchJSON {
			if matches == nil {
				matches = []searchMatch{}
			}
			return printJSON(matches)
		}

		if !anyFound {
			fmt.Printf("no match for %q\n", name)
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(searchCmd)
}
