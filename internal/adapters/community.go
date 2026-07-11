// Community installs packages other Poly users have published with
// `poly send`.
//
// It is deliberately NOT in the auto-detect chain (see All()) -- only
// reachable via an explicit "community:" prefix. Package names here are
// first-come-first-served and unrelated to pip/npm/crates.io/etc, so
// without that guard someone could publish a "requests" or "ripgrep"
// under this adapter and have it silently shadow the real thing for
// anyone who ran a plain `poly install requests`. Requiring the prefix
// means installing a community package is always a deliberate choice.
//
// There is no malware scanning. The SHA-256 recorded at publish time is
// verified at install time, which protects against corruption or
// tampering in transit -- not against a malicious uploader.
package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

// Same public Supabase project/anon key as internal/account -- kept as
// its own copy here (like account.go does) rather than a shared config
// package, since the anon key is meant to be embedded in clients anyway.
const (
	supabaseURL     = "https://iuymslcbbrbahxbfuzrr.supabase.co"
	supabaseAnonKey = "sb_publishable_ocTnDk4vyMMGcyGBeB3bOg_0CMGkdaD"
)

func binaryFileName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

type Community struct{}

func (Community) Name() string { return "community" }

type communityPackageRow struct {
	Version     string `json:"version"`
	Description string `json:"description"`
	StoragePath string `json:"storage_path"`
	SHA256      string `json:"sha256"`
	IsOfficial  bool   `json:"is_official"`
}

func fetchCommunityPackage(name string) (row communityPackageRow, found bool, err error) {
	url := supabaseURL + "/rest/v1/community_packages?name=eq." + name + "&select=version,description,storage_path,sha256,is_official"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return row, false, err
	}
	req.Header.Set("apikey", supabaseAnonKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return row, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return row, false, fmt.Errorf("community registry lookup failed: %s: %s", resp.Status, body)
	}

	var rows []communityPackageRow
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return row, false, err
	}
	if len(rows) == 0 {
		return row, false, nil
	}
	return rows[0], true, nil
}

func (c Community) Search(name string) (SearchResult, error) {
	row, found, err := fetchCommunityPackage(name)
	if err != nil {
		return SearchResult{}, err
	}
	if !found {
		return SearchResult{Found: false}, nil
	}
	summary := row.Description
	if row.IsOfficial {
		summary += " [official ✓]"
	}
	return SearchResult{Found: true, Version: row.Version, Summary: summary}, nil
}

func (c Community) Install(name, version string) (installedVersion string, err error) {
	row, found, err := fetchCommunityPackage(name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("no community package named %q (published packages are found only via poly search/install community:<name>)", name)
	}
	if version != "" && version != row.Version {
		return "", fmt.Errorf("community package %s is published at version %s (requested %s) -- only the latest published version is available", name, row.Version, version)
	}

	downloadURL := supabaseURL + "/storage/v1/object/public/community-packages/" + row.StoragePath

	archivePath, err := downloadToTemp(downloadURL, name)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", name, err)
	}
	defer os.Remove(archivePath)

	if err := verifySHA256(archivePath, row.SHA256); err != nil {
		return "", fmt.Errorf("checksum verification failed for %s: %w", name, err)
	}

	extractDir, err := os.MkdirTemp("", "poly-community-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(extractDir)

	binName := binaryFileName(name)
	if err := extractArchive(archivePath, row.StoragePath, extractDir, binName); err != nil {
		return "", fmt.Errorf("extracting %s: %w", name, err)
	}

	srcPath, err := findFile(extractDir, binName)
	if err != nil {
		return "", fmt.Errorf("published package for %s didn't contain a file named %q: %w", name, binName, err)
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

	recordCommunityDownload(name) // best-effort; a failed counter bump shouldn't fail the install

	return row.Version, nil
}

func recordCommunityDownload(name string) {
	payload, err := json.Marshal(map[string]string{"p_name": name})
	if err != nil {
		return
	}
	req, err := http.NewRequest("POST", supabaseURL+"/rest/v1/rpc/record_download", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (c Community) Remove(name string) error {
	binDir, err := polyBinDir()
	if err != nil {
		return err
	}
	path := filepath.Join(binDir, binaryFileName(name))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
