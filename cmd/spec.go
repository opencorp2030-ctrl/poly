package cmd

import (
	"fmt"
	"strings"

	"poly/internal/adapters"
)

// parseSpec splits a package spec of the form "[adapter:]name[@version]"
// into its parts, e.g. "npm:lodash@4.17.21" -> ("npm", "lodash", "4.17.21").
func parseSpec(spec string) (adapterPrefix, name, version string) {
	rest := spec
	if idx := strings.Index(rest, ":"); idx != -1 {
		prefix := rest[:idx]
		if _, ok := adapters.ByName(prefix); ok {
			adapterPrefix = prefix
			rest = rest[idx+1:]
		}
	}

	if idx := strings.LastIndex(rest, "@"); idx != -1 {
		return adapterPrefix, rest[:idx], rest[idx+1:]
	}
	return adapterPrefix, rest, ""
}

// resolveAdapter finds which adapter owns a package name: the explicitly
// requested one if a prefix was given, or the first adapter (in
// tap -> pip -> npm order) that reports the package exists.
func resolveAdapter(adapterPrefix, name string) (adapters.Adapter, error) {
	if adapterPrefix != "" {
		a, ok := adapters.ByName(adapterPrefix)
		if !ok {
			return nil, fmt.Errorf("unknown adapter %q", adapterPrefix)
		}
		return a, nil
	}

	var tried []string
	for _, a := range adapters.All() {
		result, err := a.Search(name)
		if err != nil {
			tried = append(tried, fmt.Sprintf("%s (lookup error: %v)", a.Name(), err))
			continue
		}
		if result.Found {
			return a, nil
		}
		tried = append(tried, a.Name())
	}
	return nil, fmt.Errorf("%s not found via any adapter (tried: %s)", name, strings.Join(tried, ", "))
}
