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

// VersionLimited is implemented by adapters whose available version is
// fixed up front (tap/community formulas), as opposed to delegating
// version resolution to an external tool. Auto-routing consults it so a
// package whose requested version an adapter can't offer is skipped
// rather than chosen and then failing the install.
type VersionLimited interface {
	AvailableVersion(name string) (version string, ok bool)
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
// never auto-detected -- see Community's doc comment for why. apt/dnf/
// pacman are there too because auto-resolving a bare package name to a
// system package would be a surprise on Linux, winget/choco for the same
// reason on Windows, gem because it's a single-language ecosystem best
// reached on purpose (like cargo/go), and gh because every owner/repo on
// GitHub is a possible package.
func explicitOnly() []Adapter {
	return []Adapter{
		Community{},
		Apt{},
		Dnf{},
		Pacman{},
		Winget{},
		Choco{},
		Gem{},
		Gh{},
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

// Names returns the explicit prefix name of every known adapter.
func Names() []string {
	var names []string
	for _, a := range All() {
		names = append(names, a.Name())
	}
	for _, a := range explicitOnly() {
		names = append(names, a.Name())
	}
	return names
}
