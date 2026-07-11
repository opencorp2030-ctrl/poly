package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Entry struct {
	Name        string    `json:"name"`
	Adapter     string    `json:"adapter"`
	Version     string    `json:"version"`
	InstalledAt time.Time `json:"installed_at"`
}

type Manifest struct {
	Packages map[string]Entry `json:"packages"`
	path     string
}

func dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".poly"), nil
}

// Load reads the manifest from ~/.poly/manifest.json, creating an empty one if it doesn't exist yet.
func Load() (*Manifest, error) {
	d, err := dir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(d, 0o755); err != nil {
		return nil, err
	}

	path := filepath.Join(d, "manifest.json")
	m := &Manifest{Packages: map[string]Entry{}, path: path}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	m.path = path
	return m, nil
}

func (m *Manifest) Save() error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0o644)
}

func (m *Manifest) Add(e Entry) {
	m.Packages[e.Name] = e
}

func (m *Manifest) Remove(name string) {
	delete(m.Packages, name)
}

func (m *Manifest) Get(name string) (Entry, bool) {
	e, ok := m.Packages[name]
	return e, ok
}
