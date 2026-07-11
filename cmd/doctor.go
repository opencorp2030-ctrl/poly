package cmd

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"poly/internal/account"
	"poly/internal/manifest"
	"poly/internal/ui"
)

// severity controls how a failed check is reported and whether it flips
// the command's exit code -- a missing pip is informational (you may
// just not use Python), but a broken ~/.poly is a real problem.
type severity int

const (
	sevInfo severity = iota
	sevError
)

type doctorCheck struct {
	label string
	run   func() (ok bool, detail string)
	sev   severity
}

func toolVersion(bin string, args ...string) (bool, string) {
	if _, err := exec.LookPath(bin); err != nil {
		return false, "not found on PATH"
	}
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		return true, "found, but --version failed"
	}
	line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
	return true, line
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your machine for common poly problems",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		polyDir := filepath.Join(home, ".poly")
		binDir := filepath.Join(polyDir, "bin")

		checks := []doctorCheck{
			{"go", func() (bool, string) { return toolVersion("go", "version") }, sevInfo},
			{"node/npm", func() (bool, string) { return toolVersion("npm", "--version") }, sevInfo},
			{"python/pip", func() (bool, string) {
				if ok, detail := toolVersion("pip3", "--version"); ok {
					return ok, detail
				}
				return toolVersion("pip", "--version")
			}, sevInfo},
			{"cargo", func() (bool, string) { return toolVersion("cargo", "--version") }, sevInfo},
			{"brew", func() (bool, string) {
				if runtime.GOOS == "windows" {
					return true, "not applicable on Windows"
				}
				return toolVersion("brew", "--version")
			}, sevInfo},
			{"~/.poly accessible", func() (bool, string) {
				if err := os.MkdirAll(polyDir, 0o755); err != nil {
					return false, err.Error()
				}
				probe := filepath.Join(polyDir, ".doctor-write-test")
				if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
					return false, "not writable: " + err.Error()
				}
				os.Remove(probe)
				return true, polyDir
			}, sevError},
			{"~/.poly/bin on PATH", func() (bool, string) {
				pathEnv := os.Getenv("PATH")
				sep := string(os.PathListSeparator)
				for _, p := range strings.Split(pathEnv, sep) {
					if filepath.Clean(p) == binDir {
						return true, binDir
					}
				}
				return false, binDir + " isn't on PATH -- tap/community installs won't be runnable by name"
			}, sevError},
			{"manifest readable", func() (bool, string) {
				m, err := manifest.Load()
				if err != nil {
					return false, err.Error()
				}
				return true, pluralCount(len(m.Packages), "package")
			}, sevError},
			{"account", func() (bool, string) {
				email := account.Email()
				if email == "" {
					return true, "not signed in (run `poly login` to publish packages or get Pro)"
				}
				if account.IsPro() {
					return true, email + " — pro ✓"
				}
				return true, email
			}, sevInfo},
			{"github.com reachable", func() (bool, string) {
				client := http.Client{Timeout: 3 * time.Second}
				resp, err := client.Head("https://api.github.com")
				if err != nil {
					return false, "offline or blocked -- self-update and tap downloads need this"
				}
				resp.Body.Close()
				return true, "reachable"
			}, sevInfo},
		}

		anyError := false
		for _, c := range checks {
			ok, detail := c.run()
			mark := ui.Green("✓")
			if !ok {
				if c.sev == sevError {
					mark = ui.Red("✗")
					anyError = true
				} else {
					mark = ui.Orange("!")
				}
			}
			cmd.Printf("%s %-20s %s\n", mark, c.label, ui.Dim(detail))
		}

		cmd.Println()
		if anyError {
			cmd.Println(ui.Red("some checks failed -- see above"))
			os.Exit(1)
		}
		cmd.Println(ui.Green("poly looks healthy"))
		return nil
	},
}

func pluralCount(n int, noun string) string {
	if n == 1 {
		return "1 " + noun + " tracked"
	}
	return strconv.Itoa(n) + " " + noun + "s tracked"
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
