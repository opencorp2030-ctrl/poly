package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
)

func TestParseSpec(t *testing.T) {
	cases := []struct {
		spec             string
		prefix, name, ver string
	}{
		{"ripgrep", "", "ripgrep", ""},
		{"ripgrep@15.1.0", "", "ripgrep", "15.1.0"},
		{"npm:lodash", "npm", "lodash", ""},
		{"npm:lodash@4.17.21", "npm", "lodash", "4.17.21"},
		{"pip:requests@2.31.0", "pip", "requests", "2.31.0"},
		{"tap:ripgrep@15.1.0", "tap", "ripgrep", "15.1.0"},
		{"brew:jq", "brew", "jq", ""},
		{"cargo:ripgrep", "cargo", "ripgrep", ""},
		{"go:golang.org/x/tools/cmd/goimports@latest", "go", "golang.org/x/tools/cmd/goimports", "latest"},
		{"go:golang.org/x/tools/cmd/goimports", "go", "golang.org/x/tools/cmd/goimports", ""},
		// npm scoped packages: the leading "@scope/" belongs to the name.
		{"npm:@types/node", "npm", "@types/node", ""},
		{"npm:@types/node@20.0.0", "npm", "@types/node", "20.0.0"},
		{"@types/node", "", "@types/node", ""},
		{"@types/node@20.0.0", "", "@types/node", "20.0.0"},
		{"npm:@types/node@", "npm", "@types/node", ""},
		// A prefix that isn't a known adapter stays part of the name.
		{"unknown:foo", "", "unknown:foo", ""},
		// Version-like suffix with no adapter.
		{"jq@1.8.2", "", "jq", "1.8.2"},
	}
	for _, c := range cases {
		prefix, name, ver := parseSpec(c.spec)
		if prefix != c.prefix || name != c.name || ver != c.ver {
			t.Errorf("parseSpec(%q) = (%q, %q, %q), want (%q, %q, %q)",
				c.spec, prefix, name, ver, c.prefix, c.name, c.ver)
		}
	}
}

// stubAdapter is a scripted Adapter for routing tests.
type stubAdapter struct {
	name    string
	found   bool
	version string
	// limited simulates a VersionLimited adapter (tap) with a fixed version.
	limited  bool
	fixedVer string
}

func (s stubAdapter) Name() string                       { return s.name }
func (s stubAdapter) Install(name, version string) (string, error) { return s.version, nil }
func (s stubAdapter) Remove(name string) error           { return nil }
func (s stubAdapter) Search(name string) (adapters.SearchResult, error) {
	if !s.found {
		return adapters.SearchResult{}, nil
	}
	return adapters.SearchResult{Found: true, Version: s.version}, nil
}

func (s stubAdapter) AvailableVersion(name string) (string, bool) {
	if !s.limited {
		return "", false
	}
	return s.fixedVer, true
}

func withAdapters(ads []adapters.Adapter, fn func()) {
	orig := autoAdapters
	autoAdapters = func() []adapters.Adapter { return ads }
	defer func() { autoAdapters = orig }()
	fn()
}

func TestResolveAdapterPrefix(t *testing.T) {
	a, err := resolveAdapter("npm", "lodash", "")
	if err != nil {
		t.Fatalf("resolveAdapter(npm) error: %v", err)
	}
	if a.Name() != "npm" {
		t.Errorf("got adapter %q, want npm", a.Name())
	}

	if _, err := resolveAdapter("nope", "x", ""); err == nil {
		t.Error("expected error for unknown adapter prefix")
	}
}

func TestResolveAdapterAutoDetect(t *testing.T) {
	tap := stubAdapter{name: "tap", found: true, version: "15.1.0", limited: true, fixedVer: "15.1.0"}
	pip := stubAdapter{name: "pip", found: true, version: "16.0.0"}

	withAdapters([]adapters.Adapter{tap, pip}, func() {
		// No version: first match wins.
		a, err := resolveAdapter("", "ripgrep", "")
		if err != nil {
			t.Fatalf("resolveAdapter: %v", err)
		}
		if a.Name() != "tap" {
			t.Errorf("no version: got %q, want tap", a.Name())
		}

		// Version matching the tap's pinned one: tap still wins.
		a, err = resolveAdapter("", "ripgrep", "15.1.0")
		if err != nil {
			t.Fatalf("resolveAdapter: %v", err)
		}
		if a.Name() != "tap" {
			t.Errorf("matching version: got %q, want tap", a.Name())
		}

		// Version the tap can't offer: skipped, next adapter wins.
		a, err = resolveAdapter("", "ripgrep", "16.0.0")
		if err != nil {
			t.Fatalf("resolveAdapter: %v", err)
		}
		if a.Name() != "pip" {
			t.Errorf("unavailable version: got %q, want pip", a.Name())
		}
	})
}

func TestResolveAdapterNotFound(t *testing.T) {
	withAdapters([]adapters.Adapter{stubAdapter{name: "tap", found: false}}, func() {
		_, err := resolveAdapter("", "missing", "")
		if err == nil {
			t.Fatal("expected error when no adapter finds the package")
		}
	})
}

func TestSpecArgCompleter(t *testing.T) {
	comps, directive := specArgCompleter(installCmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("directive = %d, want NoFileComp", directive)
	}
	if len(comps) == 0 {
		t.Fatal("expected adapter prefix completions for empty prefix")
	}
	want := map[string]bool{}
	for _, a := range adapters.Names() {
		want[a+":"] = true
	}
	for _, c := range comps {
		if !want[c] {
			t.Errorf("unexpected completion %q", c)
		}
	}

	// Partial prefix filters, e.g. "pip:" only.
	comps, _ = specArgCompleter(installCmd, nil, "pi")
	if len(comps) != 1 || comps[0] != "pip:" {
		t.Errorf(`completions for "pi" = %v, want ["pip:"]`, comps)
	}

	// Once a colon is typed, no more adapter suggestions.
	comps, _ = specArgCompleter(installCmd, nil, "pip:req")
	if len(comps) != 0 {
		t.Errorf(`completions for "pip:req" = %v, want none`, comps)
	}
}
