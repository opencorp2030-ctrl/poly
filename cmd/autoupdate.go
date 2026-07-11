package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"poly/internal/account"
)

const autoCheckInterval = 24 * time.Hour

// maybeAutoUpdate runs (throttled) on every command: it self-updates poly
// on the free tier, and additionally auto-upgrades installed packages for
// Pro accounts. Both run as detached background processes so they never
// add latency to the command the user actually typed.
func maybeAutoUpdate() {
	maybeSpawn("last-selfupdate-check", "self-update")
	if account.IsPro() {
		maybeSpawn("last-autoupgrade-check", "upgrade")
	}
}

func maybeSpawn(marker string, args ...string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	polyDir := filepath.Join(home, ".poly")
	markerPath := filepath.Join(polyDir, marker)

	if info, err := os.Stat(markerPath); err == nil && time.Since(info.ModTime()) < autoCheckInterval {
		return
	}
	if err := os.MkdirAll(polyDir, 0o755); err != nil {
		return
	}
	// Touch the marker before spawning so a burst of near-simultaneous
	// commands (or the spawned child itself, which also goes through
	// this same check) doesn't fire it more than once.
	if err := os.WriteFile(markerPath, []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		return
	}
	logFile, err := os.OpenFile(filepath.Join(polyDir, "auto-update.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}

	c := exec.Command(execPath, args...)
	c.Stdout = logFile
	c.Stderr = logFile
	_ = c.Start() // detached: intentionally not Wait()-ing
}
