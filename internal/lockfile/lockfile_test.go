package lockfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Missing file: found=false, no error.
	f, found, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("Load on missing file: found=true, want false")
	}
	if f != nil {
		t.Errorf("Load on missing file: got %+v, want nil", f)
	}

	f = &File{Packages: []string{"tap:ripgrep@15.1.0", "pip:requests@2.31.0"}}
	if err := Save(f); err != nil {
		t.Fatal(err)
	}

	got, found, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("Load after Save: found=false")
	}
	if len(got.Packages) != 2 || got.Packages[0] != "tap:ripgrep@15.1.0" {
		t.Errorf("roundtrip mismatch: %v", got.Packages)
	}

	if !Exists() {
		t.Error("Exists() = false after Save")
	}
}

func TestLoadLockAndSaveLock(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Missing lock: usable empty lock, found=false.
	l, found, err := LoadLock()
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("LoadLock on missing file: found=true, want false")
	}
	if l.Packages == nil {
		t.Fatal("LoadLock on missing file: nil Packages map")
	}

	l.Packages["ripgrep"] = LockEntry{Adapter: "tap", Version: "15.1.0", SHA256: "abc", URL: "https://example"}
	if err := SaveLock(l); err != nil {
		t.Fatal(err)
	}

	got, found, err := LoadLock()
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("LoadLock after SaveLock: found=false")
	}
	e := got.Packages["ripgrep"]
	if e.Adapter != "tap" || e.Version != "15.1.0" || e.SHA256 != "abc" {
		t.Errorf("roundtrip mismatch: %+v", e)
	}
}

func TestLockFileNames(t *testing.T) {
	if FileName != filepath.FromSlash("poly.json") {
		t.Errorf("FileName = %q", FileName)
	}
	if LockFileName != filepath.FromSlash("poly.lock") {
		t.Errorf("LockFileName = %q", LockFileName)
	}
}
