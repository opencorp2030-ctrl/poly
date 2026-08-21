package adapters

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Winget installs packages through the Windows Package Manager. Explicit
// "winget:" prefix only, same reasoning as apt/dnf/pacman: auto-resolving
// a bare name to a system-wide installer would be a surprise.
type Winget struct{}

func (Winget) Name() string { return "winget" }

func wingetBinary() (string, error) {
	path, err := exec.LookPath("winget")
	if err != nil {
		return "", fmt.Errorf("no winget binary found on PATH (are you on Windows 10 1809+/11?)")
	}
	return path, nil
}

func (w Winget) Install(name, version string) (installedVersion string, err error) {
	bin, err := wingetBinary()
	if err != nil {
		return "", err
	}

	args := []string{"install", "--exact", "--silent", "--accept-package-agreements", "--accept-source-agreements", "--id", name}
	if version != "" {
		args = append(args, "--version", version)
	}

	install := exec.Command(bin, args...)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("winget install %s failed: %w", name, err)
	}

	list := exec.Command(bin, "list", "--exact", "--id", name)
	out, err := list.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return parseWingetListVersion(out, name), nil
}

func (w Winget) Remove(name string) error {
	bin, err := wingetBinary()
	if err != nil {
		return err
	}
	uninstall := exec.Command(bin, "uninstall", "--exact", "--silent", "--id", name)
	uninstall.Stdout = os.Stdout
	uninstall.Stderr = os.Stderr
	if err := uninstall.Run(); err != nil {
		return fmt.Errorf("winget uninstall %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a package with this exact id/name exists via
// `winget show`, which queries the configured sources (winget's own
// community repo by default) with no separate API to call.
func (w Winget) Search(name string) (SearchResult, error) {
	bin, err := wingetBinary()
	if err != nil {
		return SearchResult{}, err
	}
	out, err := exec.Command(bin, "show", "--exact", "--id", name, "--source", "winget").CombinedOutput()
	if err != nil {
		return SearchResult{Found: false}, nil
	}

	var sr SearchResult
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if v, ok := strings.CutPrefix(line, "Version: "); ok {
			sr.Version = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "Description: "); ok {
			sr.Summary = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "Homepage: "); ok {
			sr.Homepage = v
		}
	}
	if sr.Version == "" {
		return SearchResult{Found: false}, nil
	}
	sr.Found = true
	return sr, nil
}

// parseWingetListVersion reads `winget list --id <name>` output, a table
// like:
//
//	Name    Id       Version  Source
//	--------------------------------
//	Ripgrep BurntSushi.ripgrep  15.1.0  winget
//
// and returns the last column-but-one (Version) of the first data row
// (the line right after the "----" separator).
func parseWingetListVersion(out []byte, name string) string {
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	seenSeparator := false
	for scanner.Scan() {
		line := scanner.Text()
		if !seenSeparator {
			if strings.HasPrefix(strings.TrimSpace(line), "---") {
				seenSeparator = true
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			return fields[len(fields)-2]
		}
	}
	return ""
}
