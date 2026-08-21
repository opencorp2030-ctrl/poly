// Package selfupdate lets poly update its own binary from GitHub
// releases -- checked (throttled) on every command, available on the
// free tier, matching the "poly stays current on its own" behavior
// described in the docs.
package selfupdate

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"poly/internal/httpclient"
)

const releasesAPI = "https://api.github.com/repos/opencorp2030-ctrl/poly/releases/latest"

type release struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func latestRelease() (*release, error) {
	req, err := http.NewRequest("GET", releasesAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpclient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checking latest release failed: %w", httpclient.ErrorStatus(resp))
	}
	defer resp.Body.Close()

	var r release
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

func assetName() string {
	name := fmt.Sprintf("poly-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// Check reports the latest published version (without a leading "v")
// and whether it's different from currentVersion. A "dev" build never
// reports an update, since it isn't tied to a release tag.
func Check(currentVersion string) (latest string, hasUpdate bool, err error) {
	if currentVersion == "dev" {
		return "", false, nil
	}
	r, err := latestRelease()
	if err != nil {
		return "", false, err
	}
	latest = strings.TrimPrefix(r.TagName, "v")
	return latest, latest != "" && latest != currentVersion, nil
}

// Apply downloads the release asset matching this OS/arch, verifies it
// against the release's checksums.txt when present, and atomically
// replaces the currently running executable. Returns the new version.
func Apply() (newVersion string, err error) {
	r, err := latestRelease()
	if err != nil {
		return "", err
	}

	want := assetName()
	var assetURL, checksumURL string
	for _, a := range r.Assets {
		if a.Name == want {
			assetURL = a.BrowserDownloadURL
		}
		if a.Name == "checksums.txt" {
			checksumURL = a.BrowserDownloadURL
		}
	}
	if assetURL == "" {
		return "", fmt.Errorf("no release binary published for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", err
	}

	// Download into the same directory as the running binary so the
	// final rename is same-filesystem (atomic), not a cross-device copy.
	tmpPath := execPath + ".update"
	if err := download(assetURL, tmpPath); err != nil {
		return "", err
	}
	defer os.Remove(tmpPath)

	if checksumURL != "" {
		expected, err := fetchChecksum(checksumURL, want)
		if err != nil {
			return "", fmt.Errorf("fetching checksums.txt: %w", err)
		}
		if expected == "" {
			return "", fmt.Errorf("no checksum entry for %s in checksums.txt", want)
		}
		if err := verifySHA256(tmpPath, expected); err != nil {
			return "", err
		}
	}

	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return "", err
	}
	if err := replaceExecutable(tmpPath, execPath); err != nil {
		return "", err
	}

	return strings.TrimPrefix(r.TagName, "v"), nil
}

// replaceExecutable atomically swaps tmpPath in over execPath. On Unix a
// plain rename suffices. Windows keeps the running executable locked
// against overwrite, so it renames the running exe aside to
// "poly.exe.old" first (allowed even for a running image), moves the new
// binary into place, and leaves the stale .old for CleanupStale to
// remove on the next invocation.
func replaceExecutable(tmpPath, execPath string) error {
	if err := os.Rename(tmpPath, execPath); err == nil {
		return nil
	} else if runtime.GOOS != "windows" {
		return fmt.Errorf("replacing %s: %w", execPath, err)
	}

	oldPath := execPath + ".old"
	if err := os.Rename(execPath, oldPath); err != nil {
		return fmt.Errorf("replacing %s (tried moving the running binary aside): %w", execPath, err)
	}
	if err := os.Rename(tmpPath, execPath); err != nil {
		os.Rename(oldPath, execPath) // best-effort rollback
		return fmt.Errorf("replacing %s: %w", execPath, err)
	}
	return nil
}

// CleanupStale removes the .old binary a Windows self-update leaves
// behind once the new one is in place. Best-effort; called on every
// command start via cmd.maybeAutoUpdate.
func CleanupStale() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	os.Remove(execPath + ".old")
}

func download(url, dest string) error {
	resp, err := httpclient.GetDownload(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func fetchChecksum(checksumURL, assetName string) (string, error) {
	resp, err := httpclient.Get(checksumURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching checksums.txt failed: %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[1] == assetName {
			return fields[0], nil
		}
	}
	return "", nil
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
		return fmt.Errorf("checksum mismatch for downloaded binary: got %s, want %s", got, expected)
	}
	return nil
}
