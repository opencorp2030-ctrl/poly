package adapters

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"poly/internal/httpclient"
)

// Gem shells out to a local `gem` to install and remove RubyGems, and
// queries the public rubygems.org API for search. Explicit "gem:" prefix
// only -- like cargo/go it's a single-language ecosystem best reached on
// purpose rather than folded into the general auto-resolve chain.
type Gem struct{}

func (Gem) Name() string { return "gem" }

func gemBinary() (string, error) {
	path, err := exec.LookPath("gem")
	if err != nil {
		return "", fmt.Errorf("no gem binary found on PATH (are you missing a Ruby install?)")
	}
	return path, nil
}

func (g Gem) Install(name, version string) (installedVersion string, err error) {
	bin, err := gemBinary()
	if err != nil {
		return "", err
	}

	args := []string{"install", name}
	if version != "" {
		args = append(args, "--version", version)
	}

	install := exec.Command(bin, args...)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("gem install %s failed: %w", name, err)
	}

	list := exec.Command(bin, "list", "--exact", "--local", name)
	out, err := list.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return parseGemListVersion(out, name), nil
}

func (g Gem) Remove(name string) error {
	bin, err := gemBinary()
	if err != nil {
		return err
	}
	uninstall := exec.Command(bin, "uninstall", name, "--executables", "--all")
	uninstall.Stdout = os.Stdout
	uninstall.Stderr = os.Stderr
	if err := uninstall.Run(); err != nil {
		return fmt.Errorf("gem uninstall %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a gem with this exact name exists on rubygems.org.
func (g Gem) Search(name string) (SearchResult, error) {
	resp, err := httpclient.Get("https://rubygems.org/api/v1/gems/" + name + ".json")
	if err != nil {
		return SearchResult{}, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return SearchResult{Found: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, fmt.Errorf("rubygems.org lookup failed: %w", httpclient.ErrorStatus(resp))
	}
	defer resp.Body.Close()

	var payload struct {
		Version     string `json:"version"`
		Info        string `json:"info"`
		HomepageURI string `json:"homepage_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Found: true, Version: payload.Version, Summary: payload.Info, Homepage: payload.HomepageURI}, nil
}

// parseGemListVersion reads `gem list --local <name>` output, e.g.
// "rails (7.1.3)", and returns the version.
func parseGemListVersion(out []byte, name string) string {
	line := strings.TrimSpace(string(out))
	prefix := name + " ("
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(line, prefix)
	end := strings.IndexAny(rest, ",)")
	if end == -1 {
		return ""
	}
	return rest[:end]
}
