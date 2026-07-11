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

// Cargo shells out to a local `cargo` to install and remove crates with
// binaries, and queries the public crates.io API for search.
type Cargo struct{}

func (Cargo) Name() string { return "cargo" }

func cargoBinary() (string, error) {
	path, err := exec.LookPath("cargo")
	if err != nil {
		return "", fmt.Errorf("no cargo binary found on PATH")
	}
	return path, nil
}

func (c Cargo) Install(name, version string) (installedVersion string, err error) {
	bin, err := cargoBinary()
	if err != nil {
		return "", err
	}

	args := []string{"install", "--force", name}
	if version != "" {
		args = append(args, "--version", version)
	}

	install := exec.Command(bin, args...)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("cargo install %s failed: %w", name, err)
	}

	list := exec.Command(bin, "install", "--list")
	out, err := list.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return parseCargoVersion(out, name), nil
}

func (c Cargo) Remove(name string) error {
	bin, err := cargoBinary()
	if err != nil {
		return err
	}
	uninstall := exec.Command(bin, "uninstall", name)
	uninstall.Stdout = os.Stdout
	uninstall.Stderr = os.Stderr
	if err := uninstall.Run(); err != nil {
		return fmt.Errorf("cargo uninstall %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a crate with this exact name exists on crates.io.
// The API requires a descriptive User-Agent or it returns 403.
func (c Cargo) Search(name string) (SearchResult, error) {
	req, err := http.NewRequest("GET", "https://crates.io/api/v1/crates/"+name, nil)
	if err != nil {
		return SearchResult{}, err
	}
	req.Header.Set("User-Agent", "poly-package-manager (https://github.com/opencorp2030-ctrl/poly)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return SearchResult{Found: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return SearchResult{}, fmt.Errorf("crates.io lookup failed: %s: %s", resp.Status, body)
	}

	var payload struct {
		Crate struct {
			NewestVersion string `json:"newest_version"`
			Description   string `json:"description"`
			Homepage      string `json:"homepage"`
			Repository    string `json:"repository"`
		} `json:"crate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SearchResult{}, err
	}

	homepage := payload.Crate.Homepage
	if homepage == "" {
		homepage = payload.Crate.Repository
	}
	return SearchResult{
		Found:    true,
		Version:  payload.Crate.NewestVersion,
		Summary:  strings.TrimSpace(payload.Crate.Description),
		Homepage: homepage,
	}, nil
}

// parseCargoVersion reads `cargo install --list` output, which looks like:
//
//	ripgrep v15.1.0:
//	    rg
//
// and returns the version for the given crate name.
func parseCargoVersion(out []byte, name string) string {
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	prefix := name + " v"
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			rest := strings.TrimPrefix(line, prefix)
			rest = strings.TrimSuffix(rest, ":")
			return strings.TrimSpace(rest)
		}
	}
	return ""
}
