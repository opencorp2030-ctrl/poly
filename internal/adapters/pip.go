package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
)

// Pip shells out to a local pip3/pip binary to install and remove packages,
// and queries the public PyPI JSON API to check package existence for search.
type Pip struct{}

func (Pip) Name() string { return "pip" }

func pipBinary() (string, error) {
	for _, candidate := range []string{"pip3", "pip"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no pip binary found on PATH (looked for pip3, pip)")
}

func (p Pip) Install(name, version string) (installedVersion string, err error) {
	bin, err := pipBinary()
	if err != nil {
		return "", err
	}

	spec := name
	if version != "" {
		spec = name + "==" + version
	}

	install := exec.Command(bin, "install", spec)
	if out, err := install.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pip install %s failed: %w\n%s", spec, err, out)
	}

	show := exec.Command(bin, "show", name)
	out, err := show.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return parseVersion(out), nil
}

func (p Pip) Remove(name string) error {
	bin, err := pipBinary()
	if err != nil {
		return err
	}
	uninstall := exec.Command(bin, "uninstall", "-y", name)
	if out, err := uninstall.CombinedOutput(); err != nil {
		return fmt.Errorf("pip uninstall %s failed: %w\n%s", name, err, out)
	}
	return nil
}

// Search checks whether a package with this exact name exists on PyPI and
// returns its latest version and summary. PyPI dropped free-text search
// years ago, so this is an exact-name lookup rather than a fuzzy search.
func (p Pip) Search(name string) (SearchResult, error) {
	resp, err := http.Get("https://pypi.org/pypi/" + name + "/json")
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return SearchResult{Found: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SearchResult{}, fmt.Errorf("pypi lookup failed: %s: %s", resp.Status, body)
	}

	var payload struct {
		Info struct {
			Version string `json:"version"`
			Summary string `json:"summary"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Found: true, Version: payload.Info.Version, Summary: payload.Info.Summary}, nil
}

func parseVersion(pipShowOutput []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(pipShowOutput)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
	}
	return ""
}
