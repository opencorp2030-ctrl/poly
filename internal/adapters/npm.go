package adapters

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

// Npm shells out to a local npm binary to install and remove global
// packages, and queries the public npm registry API for search.
type Npm struct{}

func (Npm) Name() string { return "npm" }

func npmBinary() (string, error) {
	path, err := exec.LookPath("npm")
	if err != nil {
		return "", fmt.Errorf("no npm binary found on PATH")
	}
	return path, nil
}

func (n Npm) Install(name, version string) (installedVersion string, err error) {
	bin, err := npmBinary()
	if err != nil {
		return "", err
	}

	spec := name
	if version != "" {
		spec = name + "@" + version
	}

	install := exec.Command(bin, "install", "-g", spec)
	if out, err := install.CombinedOutput(); err != nil {
		return "", fmt.Errorf("npm install -g %s failed: %w\n%s", spec, err, out)
	}

	list := exec.Command(bin, "ls", "-g", name, "--depth=0", "--json")
	out, err := list.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}

	var payload struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", fmt.Errorf("installed %s but could not parse npm ls output: %w", name, err)
	}
	return payload.Dependencies[name].Version, nil
}

func (n Npm) Remove(name string) error {
	bin, err := npmBinary()
	if err != nil {
		return err
	}
	uninstall := exec.Command(bin, "uninstall", "-g", name)
	if out, err := uninstall.CombinedOutput(); err != nil {
		return fmt.Errorf("npm uninstall -g %s failed: %w\n%s", name, err, out)
	}
	return nil
}

// Search checks whether a package with this exact name exists on the npm
// registry and returns its latest version and description.
func (n Npm) Search(name string) (SearchResult, error) {
	resp, err := http.Get("https://registry.npmjs.org/" + name + "/latest")
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return SearchResult{Found: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SearchResult{}, fmt.Errorf("npm registry lookup failed: %s: %s", resp.Status, body)
	}

	var payload struct {
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Found: true, Version: payload.Version, Summary: payload.Description}, nil
}
