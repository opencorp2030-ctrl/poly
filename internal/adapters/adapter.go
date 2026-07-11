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

// All returns every adapter poly knows about, in the order they should be
// tried when auto-detecting which backend owns a package name.
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

// ByName looks up an adapter by its explicit prefix name (pip, npm, brew, cargo, go, tap).
func ByName(name string) (Adapter, bool) {
	for _, a := range All() {
		if a.Name() == name {
			return a, true
		}
	}
	return nil, false
}
