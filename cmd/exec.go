package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"poly/internal/manifest"
	"poly/internal/ui"
)

// ephemeralInstaller is implemented by adapters that can install into an
// arbitrary directory instead of always ~/.poly/bin -- currently tap and
// community, the only two where poly itself downloads a single binary
// (pip/npm/cargo/go/brew delegate to tools with their own install
// layouts, so there's no clean "throwaway directory" for them yet).
type ephemeralInstaller interface {
	InstallTo(name, version, destDir string) (string, error)
}

var execCmd = &cobra.Command{
	Use:                "exec [adapter:]package[@version] [args...]",
	Aliases:            []string{"x"},
	Short:              "Run a package once without installing it (tap/community only)",
	Long: `Run a tap or community package once, from a throwaway location,
without adding it to ~/.poly/manifest.json -- like npx/uvx. If the
package is already installed via poly, this just runs it (same as
"poly run"). Not yet supported for pip/npm/cargo/go/brew packages,
which don't have a poly-managed single-binary layout to run ephemerally --
install those normally and use "poly run" instead.`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		spec := args[0]
		rest := args[1:]
		_, name, version := parseSpec(spec)

		m, err := manifest.Load()
		if err != nil {
			return err
		}
		if entry, ok := m.Get(name); ok {
			binPath, err := resolveBinary(name, entry.Adapter)
			if err != nil {
				return err
			}
			return execBinary(binPath, rest)
		}

		adapterPrefix, _, _ := parseSpec(spec)
		a, err := resolveAdapter(adapterPrefix, name)
		if err != nil {
			return err
		}

		ei, ok := a.(ephemeralInstaller)
		if !ok {
			return fmt.Errorf("poly exec doesn't support the %q adapter yet — install it first (poly install %s) then run it with `poly run %s`", a.Name(), spec, name)
		}

		tmpDir, err := os.MkdirTemp("", "poly-exec-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("running %s once (not installed)", name)))
		if _, err := ei.InstallTo(name, version, tmpDir); err != nil {
			return err
		}

		binName := name
		if runtime.GOOS == "windows" {
			binName += ".exe"
		}
		return execBinary(filepath.Join(tmpDir, binName), rest)
	},
}

func init() {
	rootCmd.AddCommand(execCmd)
}
