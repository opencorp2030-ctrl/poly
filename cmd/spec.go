package cmd

import (
	"fmt"
	"strings"
	"sync"

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

	// The version separator is the last "@" in the name -- except the
	// leading "@" of an npm scope ("@types/node"), which belongs to the
	// name and must not be treated as a version separator. A scoped
	// package with no version ("npm:@types/node") therefore parses
	// whole, and one with a version ("npm:@types/node@20.0.0") still
	// splits on the trailing @.
	if idx := strings.LastIndex(rest, "@"); idx > 0 {
		return adapterPrefix, rest[:idx], rest[idx+1:]
	}
	return adapterPrefix, rest, ""
}

// autoAdapters returns the adapters auto-detection tries in order. A
// package-level function variable so tests can substitute a stub list
// without hitting real registries.
var autoAdapters = adapters.All

// resolveAdapter finds which adapter owns a package name: the explicitly
// requested one if a prefix was given, or the first adapter (in
// tap -> pip -> npm order) that reports the package exists. The
// existence checks run concurrently -- adapters are mostly independent
// network round-trips -- but the result still honors priority order.
// When a version is requested, adapters whose available version is fixed
// up front (tap formulas) are skipped if they can't satisfy it, instead
// of being chosen and then failing the install.
func resolveAdapter(adapterPrefix, name, version string) (adapters.Adapter, error) {
	if adapterPrefix != "" {
		a, ok := adapters.ByName(adapterPrefix)
		if !ok {
			return nil, fmt.Errorf("unknown adapter %q", adapterPrefix)
		}
		return a, nil
	}

	ads := autoAdapters()
	results := make([]adapters.SearchResult, len(ads))
	errs := make([]error, len(ads))

	var wg sync.WaitGroup
	for i, a := range ads {
		wg.Add(1)
		go func(i int, a adapters.Adapter) {
			defer wg.Done()
			results[i], errs[i] = a.Search(name)
		}(i, a)
	}
	wg.Wait()

	var tried []string
	for i, a := range ads {
		if errs[i] != nil {
			tried = append(tried, fmt.Sprintf("%s (lookup error: %v)", a.Name(), errs[i]))
			continue
		}
		if !results[i].Found {
			tried = append(tried, a.Name())
			continue
		}
		if version != "" {
			if vl, ok := a.(adapters.VersionLimited); ok {
				if avail, ok := vl.AvailableVersion(name); ok && avail != version {
					tried = append(tried, fmt.Sprintf("%s (only has %s)", a.Name(), avail))
					continue
				}
			}
		}
		return a, nil
	}
	return nil, fmt.Errorf("%s not found via any adapter (tried: %s)", name, strings.Join(tried, ", "))
}
