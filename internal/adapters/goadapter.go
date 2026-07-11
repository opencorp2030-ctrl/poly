package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Go shells out to a local `go install` to fetch and build binaries
// straight from their module source, and queries the public Go module
// proxy for search. Unlike the other adapters, packages here are full
// module import paths (e.g. github.com/user/repo/cmd/tool), not short
// names, so this is realistically only used via an explicit "go:" prefix.
type Go struct{}

func (Go) Name() string { return "go" }

func goBinary() (string, error) {
	path, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("no go binary found on PATH")
	}
	return path, nil
}

func (g Go) Install(module, version string) (installedVersion string, err error) {
	bin, err := goBinary()
	if err != nil {
		return "", err
	}

	ver := version
	if ver == "" {
		ver = "latest"
	}
	spec := module + "@" + ver

	install := exec.Command(bin, "install", spec)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("go install %s failed: %w", spec, err)
	}

	binPath, err := g.binaryPath(module)
	if err != nil {
		return ver, nil // installed fine; just couldn't resolve the exact version
	}
	if resolved := installedModuleVersion(bin, binPath, module); resolved != "" {
		return resolved, nil
	}
	return ver, nil
}

func (g Go) Remove(module string) error {
	binPath, err := g.binaryPath(module)
	if err != nil {
		return err
	}
	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (g Go) binaryPath(module string) (string, error) {
	name := filepath.Base(module)
	dir, err := goBinDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name), nil
}

func goBinDir() (string, error) {
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		return gobin, nil
	}
	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return "", err
	}
	return filepath.Join(strings.TrimSpace(string(out)), "bin"), nil
}

// installedModuleVersion reads the module version embedded in a Go
// binary's build info (via `go version -m`) so poly can report exactly
// what got installed rather than just echoing back "latest".
func installedModuleVersion(goBin, binPath, module string) string {
	out, err := exec.Command(goBin, "version", "-m", binPath).Output()
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && (fields[0] == "mod" || fields[0] == "path") && fields[0] == "mod" {
			return fields[2]
		}
	}
	return ""
}

// Search checks whether a module exists via the public Go module proxy
// (the same one `go install` itself resolves through). The proxy's
// @latest endpoint only answers for a module's root (where go.mod
// lives), not arbitrary package paths inside it -- but most real
// `go install` targets are a cmd subpackage of a larger module (e.g.
// golang.org/x/tools/cmd/goimports lives in the golang.org/x/tools
// module). So this walks the path upward, the same resolution `go
// install` itself does, until it finds the enclosing module.
func (g Go) Search(module string) (SearchResult, error) {
	segments := strings.Split(module, "/")
	for end := len(segments); end >= 2; end-- {
		candidate := strings.Join(segments[:end], "/")
		version, found, err := latestModuleVersion(candidate)
		if err != nil {
			return SearchResult{}, err
		}
		if found {
			return SearchResult{Found: true, Version: version, Homepage: "https://pkg.go.dev/" + module}, nil
		}
	}
	return SearchResult{Found: false}, nil
}

func latestModuleVersion(module string) (version string, found bool, err error) {
	resp, err := http.Get("https://proxy.golang.org/" + escapeGoModulePath(module) + "/@latest")
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("go module proxy lookup failed: %s: %s", resp.Status, body)
	}

	var payload struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", false, err
	}
	return payload.Version, true, nil
}

// escapeGoModulePath implements Go's module path escaping (uppercase
// letters become "!" + the lowercase letter) required by the module
// proxy protocol.
func escapeGoModulePath(p string) string {
	var b strings.Builder
	for _, r := range p {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
