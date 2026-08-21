// Package httpclient is poly's single HTTP configuration point: a shared
// client with a timeout (the Go default client has none, which lets a
// hung network block a command forever) and a descriptive User-Agent.
package httpclient

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const UserAgent = "poly-package-manager (https://github.com/opencorp2030-ctrl/poly)"

// Client is used for all of poly's registry/API calls. 30s is long
// enough for any metadata lookup, short enough that a dead network
// fails a command promptly instead of hanging it.
var Client = &http.Client{Timeout: 30 * time.Second}

// DownloadClient is used for artifact downloads, which can legitimately
// take longer than a metadata lookup on a slow connection; streaming
// still aborts as soon as the connection itself fails.
var DownloadClient = &http.Client{Timeout: 10 * time.Minute}

// Do performs req with the shared client and User-Agent. Existing
// request headers (apikey, Authorization, ...) are left intact.
func Do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", UserAgent)
	return Client.Do(req)
}

// DoUpload performs req with the download client (longer timeout), for
// large body transfers like package uploads.
func DoUpload(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", UserAgent)
	return DownloadClient.Do(req)
}

// Get performs a GET with the shared client and User-Agent.
func Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return Do(req)
}

// GetDownload performs a GET with the download client (longer timeout).
func GetDownload(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	return DownloadClient.Do(req)
}

// Post performs a POST with the shared client and User-Agent.
func Post(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return Do(req)
}

// ReadAll reads a response body to completion and closes it.
func ReadAll(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// ErrorStatus formats a non-2xx response as an error, including the
// (possibly empty) response body for server-provided detail. It reads
// and closes the body, so only call it once per response.
func ErrorStatus(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	return fmt.Errorf("%s: %s", resp.Status, body)
}
