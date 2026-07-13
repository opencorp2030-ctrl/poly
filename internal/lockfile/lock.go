package lockfile

import (
	"encoding/json"
	"os"
)

const LockFileName = "poly.lock"

// LockEntry pins exactly what was installed for one package: which
// adapter resolved it, the exact version, and -- for tap and community
// installs, the only adapters poly itself downloads and verifies a
// checksum for -- the source URL and sha256 that were actually used.
// pip/npm/cargo/go/brew installs are pinned by exact version only, since
// poly delegates the download to those tools and never sees a checksum.
type LockEntry struct {
	Adapter string `json:"adapter"`
	Version string `json:"version"`
	SHA256  string `json:"sha256,omitempty"`
	URL     string `json:"url,omitempty"`
}

type Lock struct {
	Packages map[string]LockEntry `json:"packages"`
}

// LoadLock reads poly.lock from the current directory. found is false
// if it doesn't exist (not an error) -- the returned Lock is always
// usable either way.
func LoadLock() (l *Lock, found bool, err error) {
	data, err := os.ReadFile(LockFileName)
	if os.IsNotExist(err) {
		return &Lock{Packages: map[string]LockEntry{}}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	l = &Lock{}
	if err := json.Unmarshal(data, l); err != nil {
		return nil, false, err
	}
	if l.Packages == nil {
		l.Packages = map[string]LockEntry{}
	}
	return l, true, nil
}

func SaveLock(l *Lock) error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(LockFileName, data, 0o644)
}
