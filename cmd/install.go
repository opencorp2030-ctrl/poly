package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"poly/internal/adapters"
	"poly/internal/manifest"
)

var installCmd = &cobra.Command{
	Use:   "install [adapter:]package[@version]",
	Short: "Install a package (auto-detected across tap, pip, npm, or forced via a prefix)",
	Long: `Install a package.

Examples:
  poly install ripgrep          # auto-detected: tap, then pip, then npm
  poly install requests@2.31.0  # pinned version
  poly install pip:requests     # force the pip adapter
  poly install npm:lodash       # force the npm adapter
  poly install tap:ripgrep      # force a direct binary download`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		adapterPrefix, name, version := parseSpec(args[0])

		a, err := resolveAdapter(adapterPrefix, name)
		if err != nil {
			return err
		}

		installedVersion, err := a.Install(name, version)
		if err != nil {
			return err
		}

		m, err := manifest.Load()
		if err != nil {
			return err
		}
		m.Add(manifest.Entry{
			Name:        name,
			Adapter:     a.Name(),
			Version:     installedVersion,
			InstalledAt: time.Now(),
		})
		if err := m.Save(); err != nil {
			return err
		}

		fmt.Printf("installed %s %s (via %s)\n", name, installedVersion, a.Name())

		if a.Name() == "tap" {
			binDir, err := adapters.BinDir()
			if err == nil {
				fmt.Printf("note: tap binaries are installed to %s — make sure it's on your PATH\n", binDir)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
