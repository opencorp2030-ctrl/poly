// Package cache manages ~/.poly/cache, where tap and community downloads
// land keyed by a hash of their source URL. Repeated installs (or a
// reinstall after `poly remove`) of the same artifact reuse the cached
// file instead of re-downloading -- the sha256 check that already runs
// on every install doubles as cache-integrity verification, so a
// corrupted cache entry fails the same way a corrupted download would.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".poly", "cache"), nil
}

// KeyFor derives a stable, filesystem-safe cache key from a download URL.
func KeyFor(url string) string {
	h := sha256.Sum256([]byte(url))
	return hex.EncodeToString(h[:])
}

// Path returns the cache file path for key, creating the cache
// directory if it doesn't exist yet.
func Path(key string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, key), nil
}

// Lookup reports whether key is already cached.
func Lookup(key string) (path string, found bool, err error) {
	p, err := Path(key)
	if err != nil {
		return "", false, err
	}
	if info, err := os.Stat(p); err == nil && !info.IsDir() {
		return p, true, nil
	}
	return "", false, nil
}

// Size returns the total size in bytes of everything currently cached.
func Size() (int64, error) {
	dir, err := Dir()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var total int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		total += info.Size()
	}
	return total, nil
}

// Count returns how many files are currently cached.
func Count() (int, error) {
	dir, err := Dir()
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return len(entries), nil
}

// Clean removes every cached download.
func Clean() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}
