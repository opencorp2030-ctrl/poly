// Package account talks to Poly's Supabase project to sign in a CLI user
// and check whether they're on the Pro plan. The anon key below is a
// public, RLS-scoped key (Supabase's equivalent of a browser-safe
// publishable key) -- it is meant to be embedded in distributed clients.
package account

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	supabaseURL     = "https://iuymslcbbrbahxbfuzrr.supabase.co"
	supabaseAnonKey = "sb_publishable_ocTnDk4vyMMGcyGBeB3bOg_0CMGkdaD"
)

type Credentials struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func credentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".poly", "credentials.json"), nil
}

// Login signs in against Supabase's password grant and persists the
// session to ~/.poly/credentials.json (mode 0600).
func Login(email, password string) (*Credentials, error) {
	body, err := json.Marshal(map[string]string{"email": email, "password": password})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", supabaseURL+"/auth/v1/token?grant_type=password", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var errPayload struct {
			ErrorDescription string `json:"error_description"`
			Msg              string `json:"msg"`
		}
		json.Unmarshal(data, &errPayload)
		msg := errPayload.ErrorDescription
		if msg == "" {
			msg = errPayload.Msg
		}
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("login failed: %s", msg)
	}

	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	creds := &Credentials{
		UserID:       payload.User.ID,
		Email:        payload.User.Email,
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
	}
	if err := save(creds); err != nil {
		return nil, err
	}
	return creds, nil
}

func Logout() error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func Load() (*Credentials, error) {
	path, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func save(c *Credentials) error {
	path, err := credentialsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// IsPro reports whether the signed-in user has an active Pro plan. If
// nobody is signed in, or the check can't complete (offline, expired
// session, ...), it fails open to false -- Pro perks silently fall back
// to free behavior rather than erroring the install.
func IsPro() bool {
	creds, err := Load()
	if err != nil || creds == nil {
		return false
	}

	url := supabaseURL + "/rest/v1/profiles?id=eq." + creds.UserID + "&select=plan"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var rows []struct {
		Plan string `json:"plan"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil || len(rows) == 0 {
		return false
	}
	return rows[0].Plan == "pro"
}

// Email returns the signed-in user's email, or "" if nobody is signed in.
func Email() string {
	creds, err := Load()
	if err != nil || creds == nil {
		return ""
	}
	return creds.Email
}

// Profile is the signed-in user's public account info, as shown on
// site/account.html and by `poly account`.
type Profile struct {
	Email     string `json:"email"`
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	Plan      string `json:"plan"`
	CreatedAt string `json:"created_at"`
}

// GetProfile fetches the signed-in user's profile row. Returns an error
// if nobody is signed in or the request fails.
func GetProfile() (*Profile, error) {
	creds, err := Load()
	if err != nil {
		return nil, err
	}
	if creds == nil {
		return nil, fmt.Errorf("not signed in — run `poly login`")
	}

	url := supabaseURL + "/rest/v1/profiles?id=eq." + creds.UserID + "&select=email,username,bio,plan,created_at"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("apikey", supabaseAnonKey)
	req.Header.Set("Authorization", "Bearer "+creds.AccessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetching profile failed: %s: %s", resp.Status, body)
	}

	var rows []Profile
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no profile found for this account")
	}
	return &rows[0], nil
}
