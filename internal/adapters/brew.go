package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// Brew shells out to a local Homebrew install to install and remove
// formulae, and queries the public formulae.brew.sh JSON API (the same
// data `brew info` reads) to check package existence for search --
// so search works even without brew installed locally.
type Brew struct{}

func (Brew) Name() string { return "brew" }

func brewBinary() (string, error) {
	path, err := exec.LookPath("brew")
	if err != nil {
		return "", fmt.Errorf("no brew binary found on PATH")
	}
	return path, nil
}

func (b Brew) Install(name, version string) (installedVersion string, err error) {
	bin, err := brewBinary()
	if err != nil {
		return "", err
	}

	spec := name
	if version != "" {
		// Homebrew doesn't support arbitrary version pins; only formulae
		// explicitly published under a versioned name (e.g. node@18)
		// accept this. Anything else fails with brew's own clear error.
		spec = name + "@" + version
	}

	// `brew install` is a no-op if any version is already present, so
	// ensuring latest (no version pin, already installed) needs
	// `brew upgrade` instead -- poly upgrade relies on this.
	verb := "install"
	if version == "" {
		if listCmd := exec.Command(bin, "list", "--versions", name); listCmd.Run() == nil {
			verb = "upgrade"
		}
	}

	install := exec.Command(bin, verb, spec)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("brew %s %s failed: %w", verb, spec, err)
	}

	list := exec.Command(bin, "list", "--versions", name)
	out, err := list.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return parseBrewVersion(out), nil
}

func (b Brew) Remove(name string) error {
	bin, err := brewBinary()
	if err != nil {
		return err
	}
	uninstall := exec.Command(bin, "uninstall", name)
	uninstall.Stdout = os.Stdout
	uninstall.Stderr = os.Stderr
	if err := uninstall.Run(); err != nil {
		return fmt.Errorf("brew uninstall %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a formula with this exact name exists via the
// public formulae.brew.sh API, an exact-name lookup.
func (b Brew) Search(name string) (SearchResult, error) {
	resp, err := http.Get("https://formulae.brew.sh/api/formula/" + name + ".json")
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return SearchResult{Found: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SearchResult{}, fmt.Errorf("homebrew lookup failed: %s: %s", resp.Status, body)
	}

	var payload struct {
		Desc     string `json:"desc"`
		Homepage string `json:"homepage"`
		Versions struct {
			Stable string `json:"stable"`
		} `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Found: true, Version: payload.Versions.Stable, Summary: payload.Desc, Homepage: payload.Homepage}, nil
}

// parseBrewVersion reads the last whitespace-separated field of
// `brew list --versions <name>` output, e.g. "ripgrep 15.1.0" -> "15.1.0".
func parseBrewVersion(out []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	if scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			return fields[len(fields)-1]
		}
	}
	return ""
}
