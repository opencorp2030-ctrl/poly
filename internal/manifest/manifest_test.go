package manifest

import (
	"testing"
	"time"
)

// pointUserHome redirects os.UserHomeDir() to a temp dir for the test.
func pointUserHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
}

func TestLoadAndSave(t *testing.T) {
	pointUserHome(t)

	m, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Packages) != 0 {
		t.Error("fresh manifest should be empty")
	}

	now := time.Now()
	m.Add(Entry{Name: "ripgrep", Adapter: "tap", Version: "15.1.0", InstalledAt: now})
	m.Add(Entry{Name: "requests", Adapter: "pip", Version: "2.31.0", InstalledAt: now})
	if err := m.Save(); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Packages) != 2 {
		t.Fatalf("expected 2 packages, got %d", len(got.Packages))
	}
	e, ok := got.Get("ripgrep")
	if !ok || e.Version != "15.1.0" || e.Adapter != "tap" {
		t.Errorf("roundtrip mismatch: %+v, ok=%v", e, ok)
	}

	got.Remove("requests")
	if _, ok := got.Get("requests"); ok {
		t.Error("Remove did not delete the package")
	}
}
