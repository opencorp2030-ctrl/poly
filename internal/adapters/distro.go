package adapters

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Apt installs packages through the system apt package manager on
// Debian/Ubuntu, the way Brew does for Homebrew. Like brew it is only
// reachable via an explicit "apt:" prefix -- auto-resolving a bare
// package name to a system package would be a surprise on Linux.
type Apt struct{}

func (Apt) Name() string { return "apt" }

func aptBinaries() (aptGet, aptCache string, err error) {
	aptGet, err = exec.LookPath("apt-get")
	if err != nil {
		return "", "", fmt.Errorf("no apt-get binary found on PATH (are you on Debian/Ubuntu?)")
	}
	aptCache, err = exec.LookPath("apt-cache")
	if err != nil {
		return "", "", fmt.Errorf("no apt-cache binary found on PATH (are you on Debian/Ubuntu?)")
	}
	return aptGet, aptCache, nil
}

func (a Apt) Install(name, version string) (installedVersion string, err error) {
	aptGet, _, err := aptBinaries()
	if err != nil {
		return "", err
	}

	spec := name
	if version != "" {
		// apt-get install name=version pins the exact package version.
		spec = name + "=" + version
	}

	install := exec.Command(aptGet, "install", "-y", spec)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("apt-get install %s failed: %w", spec, err)
	}

	query := exec.Command("dpkg-query", "-W", "-f=${Version}", name)
	out, err := query.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (a Apt) Remove(name string) error {
	aptGet, _, err := aptBinaries()
	if err != nil {
		return err
	}
	remove := exec.Command(aptGet, "remove", "-y", name)
	remove.Stdout = os.Stdout
	remove.Stderr = os.Stderr
	if err := remove.Run(); err != nil {
		return fmt.Errorf("apt-get remove %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a package with this exact name exists in the apt
// repositories. apt-cache returns no "Version:" stanza for unknown
// packages, so an empty result means not found.
func (a Apt) Search(name string) (SearchResult, error) {
	_, aptCache, err := aptBinaries()
	if err != nil {
		return SearchResult{}, err
	}
	out, err := exec.Command(aptCache, "show", name).CombinedOutput()
	if err != nil {
		return SearchResult{Found: false}, nil
	}

	var sr SearchResult
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := strings.CutPrefix(line, "Version: "); ok {
			sr.Version = v
			continue
		}
		if sr.Summary == "" {
			if v, ok := strings.CutPrefix(line, "Description: "); ok {
				sr.Summary = v
			} else if v, ok := strings.CutPrefix(line, "Description-en: "); ok {
				sr.Summary = v
			}
		}
	}
	if sr.Version == "" {
		return SearchResult{Found: false}, nil
	}
	sr.Found = true
	return sr, nil
}

// Pacman installs packages through the system pacman package manager on
// Arch-family distros. Explicit "pacman:" prefix only, same reasoning as
// apt/dnf.
type Pacman struct{}

func (Pacman) Name() string { return "pacman" }

func pacmanBinary() (string, error) {
	path, err := exec.LookPath("pacman")
	if err != nil {
		return "", fmt.Errorf("no pacman binary found on PATH (are you on Arch?)")
	}
	return path, nil
}

func (p Pacman) Install(name, version string) (installedVersion string, err error) {
	bin, err := pacmanBinary()
	if err != nil {
		return "", err
	}
	if version != "" {
		// pacman has no built-in way to pin an arbitrary version from
		// the regular repos (that needs the Arch Linux Archive and a
		// direct package file); fail clearly instead of silently
		// installing the wrong version.
		return "", fmt.Errorf("pacman does not support installing a pinned version (%s@%s); omit the version", name, version)
	}

	install := exec.Command(bin, "-S", "--noconfirm", name)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("pacman -S %s failed: %w", name, err)
	}

	query := exec.Command(bin, "-Q", name)
	out, err := query.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return "", fmt.Errorf("installed %s but could not parse its version from %q", name, out)
	}
	return fields[1], nil
}

func (p Pacman) Remove(name string) error {
	bin, err := pacmanBinary()
	if err != nil {
		return err
	}
	remove := exec.Command(bin, "-R", "--noconfirm", name)
	remove.Stdout = os.Stdout
	remove.Stderr = os.Stderr
	if err := remove.Run(); err != nil {
		return fmt.Errorf("pacman -R %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a package with this exact name exists in the
// configured pacman repositories (synced databases, not the AUR).
func (p Pacman) Search(name string) (SearchResult, error) {
	bin, err := pacmanBinary()
	if err != nil {
		return SearchResult{}, err
	}
	out, err := exec.Command(bin, "-Si", name).CombinedOutput()
	if err != nil {
		return SearchResult{Found: false}, nil
	}

	var sr SearchResult
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if v, ok := strings.CutPrefix(line, "Version         : "); ok {
			sr.Version = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "Description     : "); ok {
			sr.Summary = v
			continue
		}
		if v, ok := strings.CutPrefix(line, "URL             : "); ok {
			sr.Homepage = v
		}
	}
	if sr.Version == "" {
		return SearchResult{Found: false}, nil
	}
	sr.Found = true
	return sr, nil
}

// Dnf installs packages through the system dnf package manager on
// Fedora/RHEL-family distros. Explicit "dnf:" prefix only.
type Dnf struct{}

func (Dnf) Name() string { return "dnf" }

func dnfBinary() (string, error) {
	path, err := exec.LookPath("dnf")
	if err != nil {
		return "", fmt.Errorf("no dnf binary found on PATH (are you on Fedora/RHEL?)")
	}
	return path, nil
}

func (d Dnf) Install(name, version string) (installedVersion string, err error) {
	bin, err := dnfBinary()
	if err != nil {
		return "", err
	}

	spec := name
	if version != "" {
		// dnf install name-version pins the exact package version.
		spec = name + "-" + version
	}

	install := exec.Command(bin, "install", "-y", spec)
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return "", fmt.Errorf("dnf install %s failed: %w", spec, err)
	}

	query := exec.Command("rpm", "-q", "--qf", "%{VERSION}", name)
	out, err := query.Output()
	if err != nil {
		return "", fmt.Errorf("installed %s but could not read its version: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (d Dnf) Remove(name string) error {
	bin, err := dnfBinary()
	if err != nil {
		return err
	}
	remove := exec.Command(bin, "remove", "-y", name)
	remove.Stdout = os.Stdout
	remove.Stderr = os.Stderr
	if err := remove.Run(); err != nil {
		return fmt.Errorf("dnf remove %s failed: %w", name, err)
	}
	return nil
}

// Search checks whether a package with this exact name exists in the
// dnf repositories. dnf exits non-zero for unknown packages.
func (d Dnf) Search(name string) (SearchResult, error) {
	bin, err := dnfBinary()
	if err != nil {
		return SearchResult{}, err
	}
	out, err := exec.Command(bin, "info", name).CombinedOutput()
	if err != nil {
		return SearchResult{Found: false}, nil
	}

	var sr SearchResult
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if v, ok := strings.CutPrefix(line, "Version : "); ok {
			sr.Version = v
			continue
		}
		if sr.Summary == "" {
			if v, ok := strings.CutPrefix(line, "Summary : "); ok {
				sr.Summary = v
			}
		}
	}
	if sr.Version == "" {
		return SearchResult{Found: false}, nil
	}
	sr.Found = true
	return sr, nil
}
