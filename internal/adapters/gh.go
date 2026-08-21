package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"poly/internal/httpclient"
)

// Gh installs prebuilt binaries straight from a GitHub project's
// "latest" release, in the spirit of the many curl | bash install
// scripts -- but with the release asset picked for the current OS/arch
// and, when the release ships a sha256 checksum asset, verified against
// it. It's reachable only via an explicit "gh:" prefix
// (gh:owner/repo): auto-resolving a bare package name to someone's
// GitHub release would be too unpredictable.
type Gh struct{}

func (Gh) Name() string { return "gh" }

func ghRepoParts(name string) (owner, repo string, err error) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("gh packages must be owner/repo (e.g. gh:cli/cli), got %q", name)
	}
	return parts[0], parts[1], nil
}

type ghReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type ghRelease struct {
	TagName string           `json:"tag_name"`
	Assets  []ghReleaseAsset `json:"assets"`
}

func fetchGHReleaseOnce(owner, repo, tag string) (rel ghRelease, found bool, err error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	if tag != "" {
		url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, tag)
	}
	resp, err := httpclient.Get(url)
	if err != nil {
		return rel, false, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return rel, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return rel, false, fmt.Errorf("github lookup failed: %w", httpclient.ErrorStatus(resp))
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return rel, false, err
	}
	return rel, true, nil
}

// fetchGHRelease resolves the release to use: the latest one, or the
// one tagged version -- retrying with a "v" prefix since most projects
// tag releases vX.Y.Z while users naturally type X.Y.Z.
func fetchGHRelease(owner, repo, tag string) (rel ghRelease, found bool, err error) {
	rel, found, err = fetchGHReleaseOnce(owner, repo, tag)
	if err != nil || found {
		return rel, found, err
	}
	if tag != "" && !strings.HasPrefix(strings.ToLower(tag), "v") {
		return fetchGHReleaseOnce(owner, repo, "v"+tag)
	}
	return rel, false, nil
}

var ghOSKeywords = map[string][]string{
	"windows": {"windows", "win32", "win64", "win"},
	"darwin":  {"darwin", "macos", "osx", "mac"},
	"linux":   {"linux"},
}

var ghArchKeywords = map[string][]string{
	"amd64": {"amd64", "x86_64", "x64"},
	"arm64": {"arm64", "aarch64"},
	"386":   {"386", "i386", "i686", "x86"},
	"arm":   {"armv6", "armv7", "arm"},
}

func isSourceCodeAsset(name, url string) bool {
	if strings.Contains(url, "/archive/refs/") {
		return true
	}
	n := strings.ToLower(name)
	return n == "source code (zip)" || n == "source code (tar.gz)"
}

func isChecksumAsset(name string) bool {
	n := strings.ToLower(name)
	return strings.Contains(n, "sha256") || strings.Contains(n, "sha512") || strings.Contains(n, "checksum")
}

func isArchiveURL(url string) bool {
	u := strings.ToLower(url)
	return strings.HasSuffix(u, ".tar.gz") || strings.HasSuffix(u, ".tgz") ||
		strings.HasSuffix(u, ".tar.xz") || strings.HasSuffix(u, ".tar") ||
		strings.HasSuffix(u, ".zip")
}

// ghOSMatch returns the OS group whose longest keyword appears in name.
// Substring overlap between groups (e.g. "win" inside "darwin") makes
// simple counting wrong, so the longest matching keyword decides.
func ghOSMatch(name string) string {
	bestGroup, bestLen := "", 0
	for group, words := range ghOSKeywords {
		for _, w := range words {
			if strings.Contains(name, w) && len(w) > bestLen {
				bestGroup, bestLen = group, len(w)
			}
		}
	}
	return bestGroup
}

// ghArchMatch returns the arch group whose longest keyword appears in
// name, e.g. "x86_64" beats "x86", "amd64" beats "64", "arm64" beats
// "arm".
func ghArchMatch(name string) string {
	bestGroup, bestLen := "", 0
	for group, words := range ghArchKeywords {
		for _, w := range words {
			if strings.Contains(name, w) && len(w) > bestLen {
				bestGroup, bestLen = group, len(w)
			}
		}
	}
	return bestGroup
}

// selectGHAsset picks the release asset that best matches the current
// platform: it must carry an OS keyword, and the asset whose arch
// matches wins. GitHub release assets have no standard naming scheme,
// so the matching is heuristic. It also returns the best matching
// checksum asset if the release ships one.
func selectGHAsset(rel ghRelease) (asset, checksumURL string, err error) {
	best := -1
	var bestAsset ghReleaseAsset
	for _, a := range rel.Assets {
		if isSourceCodeAsset(a.Name, a.BrowserDownloadURL) || isChecksumAsset(a.Name) {
			continue
		}
		n := strings.ToLower(a.Name)
		if ghOSMatch(n) != runtime.GOOS {
			continue
		}
		score := 100
		if ghArchMatch(n) == runtime.GOARCH {
			score += 10
		}
		if isArchiveURL(a.BrowserDownloadURL) {
			score++ // prefer archives over raw single-file binaries
		}
		if score > best {
			best, bestAsset = score, a
		}
	}
	if best < 0 {
		return "", "", fmt.Errorf("no release asset matches %s/%s (assets: %s)", runtime.GOOS, runtime.GOARCH, ghAssetNames(rel))
	}

	csBest := -1
	var csAsset ghReleaseAsset
	for _, a := range rel.Assets {
		if !isChecksumAsset(a.Name) {
			continue
		}
		n := strings.ToLower(a.Name)
		score := 0
		if ghOSMatch(n) == runtime.GOOS {
			score += 100
		}
		if ghArchMatch(n) == runtime.GOARCH {
			score += 10
		}
		if score > csBest {
			csBest, csAsset = score, a
		}
	}
	if csBest >= 0 {
		checksumURL = csAsset.BrowserDownloadURL
	}
	return bestAsset.Name, checksumURL, nil
}

func ghAssetNames(rel ghRelease) string {
	names := make([]string, 0, len(rel.Assets))
	for _, a := range rel.Assets {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}

// verifyGHChecksum verifies archivePath against the sha256 line for
// assetName inside the release's checksum file. It's best-effort: if
// the checksum file doesn't cover the chosen asset, verification is
// skipped rather than failing the install.
func verifyGHChecksum(archivePath, assetName, checksumURL string) error {
	resp, err := httpclient.GetDownload(checksumURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	assetName = strings.ToLower(assetName)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(strings.ToLower(line), assetName) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		return verifySHA256(archivePath, fields[0])
	}
	return nil
}

func (g Gh) Install(name, version string) (installedVersion string, err error) {
	binDir, err := polyBinDir()
	if err != nil {
		return "", err
	}
	v, _, err := g.installTo(name, version, binDir, true)
	return v, err
}

// InstallTo is like Install but writes the binary into destDir instead
// of ~/.poly/bin, returning the installed version and the executable
// file name (sans extension). Used by `poly exec`/`poly x`.
func (g Gh) InstallTo(name, version, destDir string) (installedVersion, binaryName string, err error) {
	return g.installTo(name, version, destDir, false)
}

func (g Gh) installTo(name, version, destDir string, persist bool) (installedVersion, binaryName string, err error) {
	owner, repo, err := ghRepoParts(name)
	if err != nil {
		return "", "", err
	}
	rel, found, err := fetchGHRelease(owner, repo, version)
	if err != nil {
		return "", "", err
	}
	if !found {
		if version != "" {
			return "", "", fmt.Errorf("%s/%s has no release tagged %s", owner, repo, version)
		}
		return "", "", fmt.Errorf("%s/%s has no releases yet", owner, repo)
	}
	tagVersion := strings.TrimPrefix(rel.TagName, "v")
	if tagVersion == "" {
		tagVersion = rel.TagName
	}

	asset, checksumURL, err := selectGHAsset(rel)
	if err != nil {
		return "", "", fmt.Errorf("%s/%s %s: %w", owner, repo, tagVersion, err)
	}

	archivePath, err := downloadToTemp(asset, name)
	if err != nil {
		return "", "", fmt.Errorf("downloading %s: %w", name, err)
	}

	if checksumURL != "" {
		if err := verifyGHChecksum(archivePath, asset, checksumURL); err != nil {
			return "", "", fmt.Errorf("checksum verification failed for %s: %w", name, err)
		}
	}

	extractDir, err := os.MkdirTemp("", "poly-gh-*")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(extractDir)

	srcPath := archivePath
	if isArchiveURL(asset) {
		if err := extractArchive(archivePath, asset, extractDir, repo); err != nil {
			return "", "", fmt.Errorf("extracting %s: %w", name, err)
		}
		srcPath, err = findExecutable(extractDir)
		if err != nil {
			return "", "", fmt.Errorf("could not find an executable inside %s: %w", name, err)
		}
	}

	// Binary name without a .exe suffix on Windows -- callers add it.
	binaryName = filepath.Base(srcPath)
	if runtime.GOOS == "windows" && strings.EqualFold(filepath.Ext(binaryName), ".exe") {
		binaryName = strings.TrimSuffix(binaryName, filepath.Ext(binaryName))
	}
	destFile := binaryName
	if runtime.GOOS == "windows" {
		destFile += ".exe"
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", "", err
	}
	if err := copyFile(srcPath, filepath.Join(destDir, destFile), 0o755); err != nil {
		return "", "", err
	}

	if persist {
		saveGHState(owner, repo, binaryName)
	}
	return tagVersion, binaryName, nil
}

func (g Gh) Remove(name string) error {
	owner, repo, err := ghRepoParts(name)
	if err != nil {
		return err
	}
	binDir, err := polyBinDir()
	if err != nil {
		return err
	}

	binName := repo
	if s, ok := loadGHState(owner, repo); ok {
		binName = s.Binary
	}
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	path := filepath.Join(binDir, binName)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	clearGHState(owner, repo)
	return nil
}

func (g Gh) Search(name string) (SearchResult, error) {
	owner, repo, err := ghRepoParts(name)
	if err != nil {
		return SearchResult{}, err
	}

	resp, err := httpclient.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo))
	if err != nil {
		return SearchResult{}, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return SearchResult{Found: false}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, fmt.Errorf("github lookup failed: %w", httpclient.ErrorStatus(resp))
	}
	var payload struct {
		Description string `json:"description"`
		HTMLURL     string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		resp.Body.Close()
		return SearchResult{}, err
	}
	resp.Body.Close()

	sr := SearchResult{Found: true, Summary: payload.Description, Homepage: payload.HTMLURL}
	if rel, found, err := fetchGHRelease(owner, repo, ""); err == nil && found {
		sr.Version = strings.TrimPrefix(rel.TagName, "v")
	}
	return sr, nil
}

// findExecutable locates the runnable binary inside an extracted
// release archive: the first .exe on Windows, the first file with the
// executable bit set on Unix. Returns an error if the archive holds no
// executable, since picking an arbitrary file (a README, a license...)
// would install the wrong thing.
func findExecutable(root string) (string, error) {
	found := ""
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return nil
		}
		if runtime.GOOS == "windows" {
			if strings.EqualFold(filepath.Ext(d.Name()), ".exe") {
				found = path
				return fs.SkipAll
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if info.Mode()&0o111 != 0 {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	if found == "" {
		return "", fmt.Errorf("no executable found in archive")
	}
	return found, nil
}

// ghState remembers the installed binary name for a repo so `poly
// remove gh:owner/repo` knows which file to delete -- the release's
// executable name isn't predictable from the repo name (cli/cli's
// binary is "gh").
type ghState struct {
	Binary string `json:"binary"`
}

func ghStatePath(owner, repo string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".poly", "gh-state", owner, repo+".json"), nil
}

func loadGHState(owner, repo string) (ghState, bool) {
	p, err := ghStatePath(owner, repo)
	if err != nil {
		return ghState{}, false
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ghState{}, false
	}
	var s ghState
	if err := json.Unmarshal(data, &s); err != nil {
		return ghState{}, false
	}
	return s, s.Binary != ""
}

func saveGHState(owner, repo, binary string) {
	p, err := ghStatePath(owner, repo)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(ghState{Binary: binary})
	if err != nil {
		return
	}
	_ = os.WriteFile(p, data, 0o644)
}

func clearGHState(owner, repo string) {
	if p, err := ghStatePath(owner, repo); err == nil {
		_ = os.Remove(p)
	}
}
