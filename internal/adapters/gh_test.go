package adapters

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGHRepoParts(t *testing.T) {
	cases := []struct {
		name, owner, repo string
		wantErr           bool
	}{
		{"cli/cli", "cli", "cli", false},
		{"BurntSushi/ripgrep", "BurntSushi", "ripgrep", false},
		{"single", "", "", true},
		{"a/b/c", "", "", true},
		{"/norepo", "", "", true},
		{"noowner/", "", "", true},
		{"", "", "", true},
	}
	for _, c := range cases {
		owner, repo, err := ghRepoParts(c.name)
		if c.wantErr {
			if err == nil {
				t.Errorf("ghRepoParts(%q): expected error", c.name)
			}
			continue
		}
		if err != nil {
			t.Errorf("ghRepoParts(%q): %v", c.name, err)
			continue
		}
		if owner != c.owner || repo != c.repo {
			t.Errorf("ghRepoParts(%q) = (%q, %q), want (%q, %q)", c.name, owner, repo, c.owner, c.repo)
		}
	}
}

func TestGHAssetFiltering(t *testing.T) {
	if !isSourceCodeAsset("Source code (zip)", "https://github.com/a/b/archive/refs/tags/v1.0.0.zip") {
		t.Error("source code archive not detected")
	}
	if isSourceCodeAsset("cli-v1.0.0-linux-amd64.zip", "https://github.com/a/b/releases/download/v1.0.0/cli-v1.0.0-linux-amd64.zip") {
		t.Error("real release asset flagged as source code")
	}
	if !isChecksumAsset("cli_1.0.0_checksums.txt") {
		t.Error("checksums.txt not detected as checksum asset")
	}
	if isChecksumAsset("cli_1.0.0_linux_amd64.tar.gz") {
		t.Error("binary archive flagged as checksum asset")
	}
	if !isArchiveURL("x.tar.gz") || !isArchiveURL("x.zip") || !isArchiveURL("x.tgz") || !isArchiveURL("x.tar.xz") {
		t.Error("archive extensions not detected")
	}
	if isArchiveURL("jq-macos-arm64") {
		t.Error("raw binary URL flagged as archive")
	}
}

func TestSelectGHAsset(t *testing.T) {
	mkAsset := func(name string) ghReleaseAsset {
		return ghReleaseAsset{Name: name, BrowserDownloadURL: "https://github.com/a/b/releases/download/v1.0.0/" + name}
	}
	rel := ghRelease{TagName: "v1.0.0", Assets: []ghReleaseAsset{
		mkAsset("Source code (zip)"),
		mkAsset("tool-" + runtime.GOOS + "-amd64.tar.gz"),
		mkAsset("tool-" + runtime.GOOS + "-arm64.tar.gz"),
		mkAsset("tool-checksums.txt"),
	}}
	asset, checksum, err := selectGHAsset(rel)
	if err != nil {
		t.Fatalf("selectGHAsset: %v", err)
	}
	if runtime.GOARCH == "arm64" {
		if !strings.Contains(asset, "arm64") {
			t.Errorf("asset %q not the arm64 build", asset)
		}
	} else {
		if !strings.Contains(asset, "amd64") {
			t.Errorf("asset %q not the amd64 build", asset)
		}
	}
	if !strings.Contains(checksum, "checksum") {
		t.Errorf("checksum asset %q unexpected", checksum)
	}
	if isSourceCodeAsset(asset, asset) {
		t.Error("source code asset selected as the download")
	}

	// No platform match at all.
	rel2 := ghRelease{TagName: "v1.0.0", Assets: []ghReleaseAsset{
		mkAsset("tool-plan9-mips.tar.gz"),
	}}
	if _, _, err := selectGHAsset(rel2); err == nil {
		t.Error("expected error when no asset matches the platform")
	}
}

func TestGHAssetOverlappingKeywords(t *testing.T) {
	mkAsset := func(name string) ghReleaseAsset {
		return ghReleaseAsset{Name: name, BrowserDownloadURL: "https://github.com/a/b/releases/download/v1.0.0/" + name}
	}

	// "darwin" contains "win" and "i386" contains "386"; the longest
	// keyword must win so the wrong platform is never selected.
	rel := ghRelease{TagName: "v1.0.0", Assets: []ghReleaseAsset{
		mkAsset("tool-darwin-arm64.tar.gz"),
		mkAsset("tool-windows-amd64.tar.gz"),
		mkAsset("tool-windows-i386.tar.gz"),
	}}
	asset, _, err := selectGHAsset(rel)
	if err != nil {
		t.Fatalf("selectGHAsset: %v", err)
	}
	if runtime.GOOS == "darwin" {
		if asset != "tool-darwin-arm64.tar.gz" {
			t.Errorf("on darwin got %q, want the darwin build", asset)
		}
	} else if runtime.GOARCH == "386" {
		if asset != "tool-windows-i386.tar.gz" {
			t.Errorf("on windows/386 got %q, want the i386 build", asset)
		}
	} else {
		if asset != "tool-windows-amd64.tar.gz" {
			t.Errorf("got %q, want the windows/amd64 build", asset)
		}
	}
}

func TestFindExecutable(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "bin"), 0o755)
	if runtime.GOOS == "windows" {
		os.WriteFile(filepath.Join(dir, "bin", "tool.exe"), []byte("x"), 0o755)
		os.WriteFile(filepath.Join(dir, "README.txt"), []byte("readme"), 0o644)
	} else {
		os.WriteFile(filepath.Join(dir, "bin", "tool"), []byte("x"), 0o755)
		os.WriteFile(filepath.Join(dir, "README.md"), []byte("readme"), 0o644)
	}
	got, err := findExecutable(dir)
	if err != nil {
		t.Fatalf("findExecutable: %v", err)
	}
	if filepath.Base(got) != "tool" && filepath.Base(got) != "tool.exe" {
		t.Errorf("findExecutable picked %q", got)
	}

	empty := t.TempDir()
	os.WriteFile(filepath.Join(empty, "LICENSE"), []byte("mit"), 0o644)
	if _, err := findExecutable(empty); err == nil {
		t.Error("expected error when no executable is present")
	}
}
