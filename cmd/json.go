package cmd

import (
	"encoding/json"
	"os"
)

// printJSON writes v as pretty-printed JSON to stdout. Used by the
// --json flags on list/outdated (and anything else that wants
// machine-readable output) instead of the human-facing tables.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
