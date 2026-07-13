package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/manifest"
)

var runCmd = &cobra.Command{
	Use:                "run <package> [args...]",
	Short:              "Run an installed package's binary by name",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		rest := args[1:]

		m, err := manifest.Load()
		if err != nil {
			return err
		}
		entry, ok := m.Get(name)
		if !ok {
			return fmt.Errorf("%s is not installed via poly — run `poly install %s` first, or `poly exec %s` to run it once without installing", name, name, name)
		}

		binPath, err := resolveBinary(name, entry.Adapter)
		if err != nil {
			return err
		}
		return execBinary(binPath, rest)
	},
}

// resolveBinary finds the executable for an installed package: tap and
// community installs always land at a known path in ~/.poly/bin, while
// pip/npm/cargo/go/brew install their own binaries wherever those tools
// put them (usually already on PATH), so those are looked up there.
func resolveBinary(name, adapterName string) (string, error) {
	switch adapterName {
	case "tap", "community":
		binDir, err := adapters.BinDir()
		if err != nil {
			return "", err
		}
		binName := name
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		path := filepath.Join(binDir, binName)
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("%s not found at %s (try reinstalling: poly install %s:%s)", name, path, adapterName, name)
		}
		return path, nil
	default:
		path, err := exec.LookPath(name)
		if err != nil {
			return "", fmt.Errorf("%s was installed via %s but isn't on your PATH: %w", name, adapterName, err)
		}
		return path, nil
	}
}

// execBinary runs path, streaming its stdio straight through, and exits
// poly with the child's own exit code on failure (so `poly run` behaves
// like directly running the binary, not like a poly command that failed).
func execBinary(path string, args []string) error {
	c := exec.Command(path, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
