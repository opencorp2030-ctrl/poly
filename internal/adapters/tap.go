package adapters

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"poly/internal/account"
	"poly/internal/registry/embedded"
)

// Tap installs prebuilt binaries directly from upstream release URLs,
// verified against a pinned sha256 checksum, the way Homebrew casks
// install binaries rather than building from source. Formulas are plain
// YAML: a handful ship embedded in the poly binary, and users can drop
// more into ~/.poly/taps.
type Tap struct{}

func (Tap) Name() string { return "tap" }

type formulaArtifact struct {
	URL    string `yaml:"url"`
	SHA256 string `yaml:"sha256"`
}

type formula struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Homepage    string `yaml:"homepage"`
	Version     string `yaml:"version"`
	Binary      string `yaml:"binary"`
	// Tier is "free" (default, if omitted) or "pro". Pro formulas are
	// part of poly's expanded formula catalog and require an active
	// Pro plan to install -- they still show up in search results so
	// free users can see what they're missing.
	Tier      string                     `yaml:"tier"`
	Artifacts map[string]formulaArtifact `yaml:"artifacts"`
}

func (f formula) isPro() bool {
	return strings.EqualFold(f.Tier, "pro")
}

func userTapsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".poly", "taps"), nil
}

func polyBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".poly", "bin"), nil
}

// loadFormula looks for name's formula, first among user-added taps
// (~/.poly/taps/<name>.yaml) so they can override built-ins, then among
// the formulas embedded in the binary.
func loadFormula(name string) (f formula, found bool, err error) {
	if dir, derr := userTapsDir(); derr == nil {
		data, rerr := os.ReadFile(filepath.Join(dir, name+".yaml"))
		if rerr == nil {
			if err := yaml.Unmarshal(data, &f); err != nil {
				return formula{}, false, fmt.Errorf("parsing tap %s: %w", name, err)
			}
			return f, true, nil
		}
	}

	data, err := fs.ReadFile(embedded.Taps, "taps/"+name+".yaml")
	if err != nil {
		return formula{}, false, nil
	}
	if err := yaml.Unmarshal(data, &f); err != nil {
		return formula{}, false, fmt.Errorf("parsing built-in tap %s: %w", name, err)
	}
	return f, true, nil
}

func (t Tap) Install(name, version string) (installedVersion string, err error) {
	f, found, err := loadFormula(name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("no tap formula for %s", name)
	}
	if f.isPro() && !account.IsPro() {
		return "", fmt.Errorf("%s is part of poly's Pro formula catalog — run `poly login` with an active Pro plan to install it", name)
	}
	if version != "" && version != f.Version {
		return "", fmt.Errorf("tap %s only offers version %s (requested %s)", name, f.Version, version)
	}

	key := runtime.GOOS + "_" + runtime.GOARCH
	artifact, ok := f.Artifacts[key]
	if !ok {
		return "", fmt.Errorf("%s has no prebuilt binary for %s", name, key)
	}

	archivePath, err := downloadToTemp(artifact.URL, name)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", name, err)
	}
	defer os.Remove(archivePath)

	if err := verifySHA256(archivePath, artifact.SHA256); err != nil {
		return "", fmt.Errorf("checksum verification failed for %s: %w", name, err)
	}

	extractDir, err := os.MkdirTemp("", "poly-tap-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(extractDir)

	binName := f.Binary
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	if err := extractArchive(archivePath, artifact.URL, extractDir, binName); err != nil {
		return "", fmt.Errorf("extracting %s: %w", name, err)
	}

	srcPath, err := findFile(extractDir, binName)
	if err != nil {
		return "", fmt.Errorf("could not find %s inside downloaded archive: %w", binName, err)
	}

	binDir, err := polyBinDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}

	destPath := filepath.Join(binDir, binName)
	if err := copyFile(srcPath, destPath, 0o755); err != nil {
		return "", err
	}

	return f.Version, nil
}

func (t Tap) Remove(name string) error {
	f, found, err := loadFormula(name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no tap formula for %s", name)
	}

	binName := f.Binary
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binDir, err := polyBinDir()
	if err != nil {
		return err
	}
	path := filepath.Join(binDir, binName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (t Tap) Search(name string) (SearchResult, error) {
	f, found, err := loadFormula(name)
	if err != nil {
		return SearchResult{}, err
	}
	if !found {
		return SearchResult{Found: false}, nil
	}
	summary := f.Description
	if f.isPro() {
		summary += " [pro]"
	}
	return SearchResult{Found: true, Version: f.Version, Summary: summary, Homepage: f.Homepage}, nil
}

// BinDir exposes ~/.poly/bin so the CLI layer can tell the user to add it
// to PATH after a tap install.
func BinDir() (string, error) {
	return polyBinDir()
}

func downloadToTemp(url, label string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "poly-download-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	if err := copyWithProgress(tmp, resp.Body, resp.ContentLength, "downloading "+label); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, expected)
	}
	return nil
}

func extractArchive(archivePath, sourceURL, destDir, binName string) error {
	switch {
	case strings.HasSuffix(sourceURL, ".tar.gz") || strings.HasSuffix(sourceURL, ".tgz"):
		return extractTarGz(archivePath, destDir)
	case strings.HasSuffix(sourceURL, ".zip"):
		return extractZip(archivePath, destDir)
	default:
		// Raw binary, no archive -- write it directly under the expected
		// binary name rather than the download's own filename (release
		// assets are often named e.g. "jq-macos-arm64", not "jq").
		return copyFile(archivePath, filepath.Join(destDir, binName), 0o755)
	}
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
		}
	}
}

func extractZip(archivePath, destDir string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, zf := range r.File {
		target, err := safeJoin(destDir, zf.Name)
		if err != nil {
			return err
		}

		if zf.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := zf.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, zf.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			rc.Close()
			return err
		}
		out.Close()
		rc.Close()
	}
	return nil
}

// safeJoin joins base and name, rejecting archive entries that would
// escape base via ".." (zip-slip).
func safeJoin(base, name string) (string, error) {
	target := filepath.Join(base, name)
	if !strings.HasPrefix(target, filepath.Clean(base)+string(os.PathSeparator)) && target != filepath.Clean(base) {
		return "", fmt.Errorf("illegal archive entry path: %s", name)
	}
	return target, nil
}

func findFile(root, name string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == name {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("%s not found", name)
	}
	return found, nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
