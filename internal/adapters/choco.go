package adapters

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Choco installs packages through Chocolatey on Windows. Explicit
// "choco:" prefix only, same reasoning as apt/dnf/pacman/winget.
type Choco struct{}

func (Choco) Name() string { return "choco" }

func chocoBinary() (string, error) {
	path, err := exec.LookPath("choco")
	if err != nil {
		return "", fmt.Errorf("no choco binary found on PATH (install Chocolatey first: https://chocolatey.org/install)")
	}
	return path, nil
}

func (c Choco) Install(name, version string) (installedVersion string, err error) {
	bin, err := chocoBinary()
	if err != nil {
		return "", err
	}

	args := []string{"install", name, "-y", "--no-progress"}
	if version != "" {
		args = append(args, "--version="+version)
	}

	install := exec.Command(bin, args...)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("choco install %s failed: %w", name, err)
	}

	list := exec.Command(bin, "list", "--local-only", "--exact", name, "--limit-output")
	out, err := list.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return parseChocoLimitOutput(out), nil
}

func (c Choco) Remove(name string) error {
	bin, err := chocoBinary()
	if err != nil {
		return err
	}
	uninstall := exec.Command(bin, "uninstall", name, "-y")
	uninstall.Stdout = os.Stdout
	uninstall.Stderr = os.Stderr
	if err := uninstall.Run(); err != nil {
		return fmt.Errorf("choco uninstall %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a package with this exact name exists in the
// configured Chocolatey sources (community repo by default) via `choco
// search --exact`, which prints "name|version" with --limit-output.
func (c Choco) Search(name string) (SearchResult, error) {
	bin, err := chocoBinary()
	if err != nil {
		return SearchResult{}, err
	}
	out, err := exec.Command(bin, "search", name, "--exact", "--limit-output").CombinedOutput()
	if err != nil {
		return SearchResult{Found: false}, nil
	}
	version := parseChocoLimitOutput(out)
	if version == "" {
		return SearchResult{Found: false}, nil
	}
	return SearchResult{Found: true, Version: version}, nil
}

// parseChocoLimitOutput reads choco's --limit-output format, "name|version"
// (one line), and returns the version half.
func parseChocoLimitOutput(out []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	if scanner.Scan() {
		parts := strings.SplitN(strings.TrimSpace(scanner.Text()), "|", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return ""
}
