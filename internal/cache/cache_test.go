package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func pointUserHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
}

func TestKeyFor(t *testing.T) {
	// Deterministic, stable across calls.
	a := KeyFor("https://example.com/a.tar.gz")
	b := KeyFor("https://example.com/a.tar.gz")
	if a != b {
		t.Error("KeyFor is not deterministic")
	}
	if len(a) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(a))
	}
	if KeyFor("https://example.com/a.tar.gz") == KeyFor("https://example.com/b.tar.gz") {
		t.Error("distinct URLs collided")
	}
}

func TestPathLookupClean(t *testing.T) {
	pointUserHome(t)

	p, err := Path("abcdef")
	if err != nil {
		t.Fatal(err)
	}
	dir, _ := Dir()
	if filepath.Dir(p) != dir {
		t.Errorf("Path file is not inside the cache dir: %s", p)
	}

	if _, found, err := Lookup("abcdef"); err != nil || found {
		t.Errorf("Lookup on missing key: found=%v err=%v", found, err)
	}

	if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, found, err := Lookup("abcdef"); err != nil || !found {
		t.Errorf("Lookup on existing key: found=%v err=%v", found, err)
	}
}

func TestSizeCountClean(t *testing.T) {
	pointUserHome(t)

	if n, _ := Count(); n != 0 {
		t.Errorf("fresh cache Count = %d, want 0", n)
	}
	if n, _ := Size(); n != 0 {
		t.Errorf("fresh cache Size = %d, want 0", n)
	}

	p1, _ := Path("aaa")
	p2, _ := Path("bbb")
	os.WriteFile(p1, make([]byte, 10), 0o644)
	os.WriteFile(p2, make([]byte, 20), 0o644)

	if n, _ := Count(); n != 2 {
		t.Errorf("Count = %d, want 2", n)
	}
	if n, _ := Size(); n != 30 {
		t.Errorf("Size = %d, want 30", n)
	}

	if err := Clean(); err != nil {
		t.Fatal(err)
	}
	if n, _ := Count(); n != 0 {
		t.Errorf("after Clean, Count = %d, want 0", n)
	}
	if _, found, _ := Lookup("aaa"); found {
		t.Error("entry still present after Clean")
	}
}
