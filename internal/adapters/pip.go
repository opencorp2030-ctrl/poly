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
	args := []string{"install"}
	if version != "" {
		spec = name + "==" + version
	} else {
		// No version pin means "ensure latest" -- pip's plain `install`
		// is a no-op if any version is already present, so this needs
		// --upgrade to actually fetch newer releases (poly upgrade
		// relies on this).
		args = append(args, "--upgrade")
	}
	args = append(args, spec)

	install := exec.Command(bin, args...)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("pip install %s failed: %w", spec, err)
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
	uninstall.Stdout = os.Stdout
	uninstall.Stderr = os.Stderr
	if err := uninstall.Run(); err != nil {
		return fmt.Errorf("pip uninstall %s failed: %w", name, err)
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
			Version  string `json:"version"`
			Summary  string `json:"summary"`
			HomePage string `json:"home_page"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Found: true, Version: payload.Info.Version, Summary: payload.Info.Summary, Homepage: payload.Info.HomePage}, nil
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
