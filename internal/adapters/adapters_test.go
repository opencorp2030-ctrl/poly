package adapters

import "testing"

func TestParseCargoVersion(t *testing.T) {
	out := `ripgrep v15.1.0:
    rg
cargo-binstall v1.12.0:
    cargo-binstall
`
	if got := parseCargoVersion([]byte(out), "ripgrep"); got != "15.1.0" {
		t.Errorf("ripgrep: got %q, want 15.1.0", got)
	}
	if got := parseCargoVersion([]byte(out), "cargo-binstall"); got != "1.12.0" {
		t.Errorf("cargo-binstall: got %q, want 1.12.0", got)
	}
	if got := parseCargoVersion([]byte(out), "missing"); got != "" {
		t.Errorf("missing: got %q, want empty", got)
	}
}

func TestParseBrewVersion(t *testing.T) {
	if got := parseBrewVersion([]byte("ripgrep 15.1.0\n")); got != "15.1.0" {
		t.Errorf("got %q, want 15.1.0", got)
	}
	if got := parseBrewVersion([]byte("node 18.20.4\n")); got != "18.20.4" {
		t.Errorf("got %q, want 18.20.4", got)
	}
	if got := parseBrewVersion(nil); got != "" {
		t.Errorf("empty input: got %q, want empty", got)
	}
}

func TestParsePipVersion(t *testing.T) {
	out := `Name: requests
Version: 2.31.0
Summary: Python HTTP for Humans.
`
	if got := parseVersion([]byte(out)); got != "2.31.0" {
		t.Errorf("got %q, want 2.31.0", got)
	}
}

func TestModuleVersionFromBuildInfo(t *testing.T) {
	out := `/usr/local/bin/goimports: go1.22.0
	path	golang.org/x/tools/cmd/goimports
	mod	golang.org/x/tools	v0.24.0	h1:...
	build	-buildmode=exe
`
	if got := moduleVersionFromBuildInfo([]byte(out)); got != "v0.24.0" {
		t.Errorf("got %q, want v0.24.0", got)
	}
	if got := moduleVersionFromBuildInfo([]byte("nothing useful\n")); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

func TestEscapeGoModulePath(t *testing.T) {
	cases := map[string]string{
		"github.com/foo/bar":      "github.com/foo/bar",
		"github.com/Foo/bar":      "github.com/!foo/bar",
		"golang.org/x/tools":      "golang.org/x/tools",
		"gopkg.in/yaml.v3":        "gopkg.in/yaml.v3",
		"example.com/UPPER/Case":  "example.com/!u!p!p!e!r/!case",
	}
	for in, want := range cases {
		if got := escapeGoModulePath(in); got != want {
			t.Errorf("escapeGoModulePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseWingetListVersion(t *testing.T) {
	out := `Name    Id                  Version Source
--------------------------------------------
Ripgrep BurntSushi.ripgrep  15.1.0  winget
`
	if got := parseWingetListVersion([]byte(out), "ripgrep"); got != "15.1.0" {
		t.Errorf("got %q, want 15.1.0", got)
	}
	if got := parseWingetListVersion([]byte("Name Id Version Source\n"), "missing"); got != "" {
		t.Errorf("no separator: got %q, want empty", got)
	}
}

func TestParseChocoLimitOutput(t *testing.T) {
	if got := parseChocoLimitOutput([]byte("ripgrep|15.1.0\n")); got != "15.1.0" {
		t.Errorf("got %q, want 15.1.0", got)
	}
	if got := parseChocoLimitOutput(nil); got != "" {
		t.Errorf("empty input: got %q, want empty", got)
	}
}

func TestParseGemListVersion(t *testing.T) {
	if got := parseGemListVersion([]byte("rails (7.1.3)\n"), "rails"); got != "7.1.3" {
		t.Errorf("got %q, want 7.1.3", got)
	}
	if got := parseGemListVersion([]byte("rails (7.1.3, 7.0.8)\n"), "rails"); got != "7.1.3" {
		t.Errorf("multiple versions: got %q, want 7.1.3", got)
	}
	if got := parseGemListVersion([]byte("*** LOCAL GEMS ***\n"), "missing"); got != "" {
		t.Errorf("no match: got %q, want empty", got)
	}
}

func TestSafeJoin(t *testing.T) {
	base := "/tmp/poly-extract"
	good := []string{"rg", "bin/rg", "sub/dir/file", "a/b/../c"}
	for _, name := range good {
		if _, err := safeJoin(base, name); err != nil {
			t.Errorf("safeJoin(%q) rejected a safe path: %v", name, err)
		}
	}
	bad := []string{"../escape", "..", "a/../../etc/passwd", "C:\\Windows\\evil", "C:/Windows/evil", "\\\\server\\share\\evil"}
	for _, name := range bad {
		if _, err := safeJoin(base, name); err == nil {
			t.Errorf("safeJoin(%q) accepted a path escaping the base", name)
		}
	}
}
