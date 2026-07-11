package adapters

// SearchResult describes whether a package was found by an adapter and,
// if so, basic metadata about it.
type SearchResult struct {
	Found    bool
	Version  string
	Summary  string
	Homepage string
}

// Adapter is implemented by every backend poly can install packages through
// (pip, npm, brew, tap/binary downloads, ...).
type Adapter interface {
	Name() string
	// Install installs name. version is "" for "latest".
	// It returns the version that actually got installed.
	Install(name, version string) (installedVersion string, err error)
	Remove(name string) error
	Search(name string) (SearchResult, error)
}

// All returns every auto-detected adapter, in the order they're tried
// when resolving a bare package name with no prefix.
func All() []Adapter {
	return []Adapter{
		Tap{},
		Brew{},
		Pip{},
		Npm{},
		Cargo{},
		Go{},
	}
}

// explicitOnly lists adapters reachable only via an explicit prefix,
// never auto-detected -- see Community's doc comment for why.
func explicitOnly() []Adapter {
	return []Adapter{
		Community{},
	}
}

// ByName looks up an adapter by its explicit prefix name (pip, npm,
// brew, cargo, go, tap, community).
func ByName(name string) (Adapter, bool) {
	for _, a := range All() {
		if a.Name() == name {
			return a, true
		}
	}
	for _, a := range explicitOnly() {
		if a.Name() == name {
			return a, true
		}
	}
	return nil, false
}
