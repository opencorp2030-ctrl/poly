package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
)

// specArgCompleter completes `adapter:` prefixes (and the known package
// managers) for commands that take [adapter:]package specs. It's wired
// into install/add/remove/info/search/run/exec/update for `poly
// completion` users -- tab-completing the adapter name is the common
// case, and package names are too backend-specific to guess here.
func specArgCompleter(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if strings.Contains(toComplete, ":") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var out []string
	for _, name := range adapters.Names() {
		prefix := name + ":"
		if strings.HasPrefix(prefix, toComplete) {
			out = append(out, prefix)
		}
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	for _, c := range []*cobra.Command{installCmd, addCmd, removeCmd, infoCmd, searchCmd, runCmd, execCmd, updateCmd} {
		c.ValidArgsFunction = specArgCompleter
	}
}
