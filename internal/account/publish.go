package account

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// PublishPackage uploads localPath (a file, or a directory that gets
// tar.gz'd first) to the community packages bucket under the signed-in
// user's own folder, then registers it via the publish_package RPC.
// name must not already be taken by a different account.
func PublishPackage(name, version, localPath, description string) error {
	creds, err := LoadFresh()
	if err != nil {
		return err
	}
	if creds == nil {
		return fmt.Errorf("not signed in — run `poly login`")
	}

	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", localPath, err)
	}

	var data []byte
	ext := ".tar.gz"
	if info.IsDir() {
		data, err = tarGzDir(localPath)
		if err != nil {
			return fmt.Errorf("archiving %s: %w", localPath, err)
		}
	} else {
		data, err = os.ReadFile(localPath)
		if err != nil {
			return err
		}
		ext = filepath.Ext(localPath)
		if ext == "" {
			ext = ".bin"
		}
	}

	sum := sha256.Sum256(data)
	shaHex := hex.EncodeToString(sum[:])
	storagePath := fmt.Sprintf("%s/%s/%s%s", creds.UserID, name, version, ext)

	if err := uploadToStorage(creds.AccessToken, storagePath, data); err != nil {
		return fmt.Errorf("uploading package: %w", err)
	}

	if err := callPublishPackage(creds.AccessToken, name, version, storagePath, shaHex, len(data), description); err != nil {
		return err
	}
	return nil
}

func uploadToStorage(accessToken, storagePath string, data []byte) error {
	url := supabaseURL + "/storage/v1/object/community-packages/" + storagePath
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("x-upsert", "true")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, body)
	}
	return nil
}

func callPublishPackage(accessToken, name, version, storagePath, sha256Hex string, sizeBytes int, description string) error {
	payload := map[string]interface{}{
		"p_name":         name,
		"p_version":      version,
		"p_storage_path": storagePath,
		"p_sha256":       sha256Hex,
		"p_size_bytes":   sizeBytes,
		"p_description":  description,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", supabaseURL+"/rest/v1/rpc/publish_package", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var errPayload struct {
			Message string `json:"message"`
		}
		respBody, _ := io.ReadAll(resp.Body)
		json.Unmarshal(respBody, &errPayload)
		if errPayload.Message != "" {
			return fmt.Errorf("%s", errPayload.Message)
		}
		return fmt.Errorf("publish failed: %s: %s", resp.Status, respBody)
	}
	return nil
}

func tarGzDir(dir string) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		hdr, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		hdr.Name = rel

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
