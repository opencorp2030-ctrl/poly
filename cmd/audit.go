package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"poly/internal/manifest"
	"poly/internal/ui"
)

// osvEcosystem maps poly's adapter names to OSV.dev's ecosystem
// identifiers. Only pip and npm packages have a well-known public
// vulnerability database poly can query this way; tap/community/cargo/
// go/brew are skipped (cargo and go do have OSV ecosystems too, but
// poly doesn't yet track cargo/go packages by their registry name in a
// way that's safe to query -- left for later rather than guessing).
var osvEcosystem = map[string]string{
	"pip": "PyPI",
	"npm": "npm",
}

type osvQuery struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Version string `json:"version"`
}

type osvVuln struct {
	ID string `json:"id"`
}

type osvResult struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvBatchResponse struct {
	Results []osvResult `json:"results"`
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Check installed pip/npm packages against known vulnerabilities (via osv.dev)",
	RunE: func(cmd *cobra.Command, args []string) error {
		m, err := manifest.Load()
		if err != nil {
			return err
		}

		type candidate struct {
			name, version, ecosystem string
		}
		var candidates []candidate
		for name, e := range m.Packages {
			eco, ok := osvEcosystem[e.Adapter]
			if !ok {
				continue
			}
			candidates = append(candidates, candidate{name, e.Version, eco})
		}

		if len(candidates) == 0 {
			fmt.Println(ui.Dim("no pip/npm packages installed via poly to audit"))
			return nil
		}

		queries := make([]osvQuery, len(candidates))
		for i, c := range candidates {
			queries[i].Package.Name = c.name
			queries[i].Package.Ecosystem = c.ecosystem
			queries[i].Version = c.version
		}

		body, err := json.Marshal(map[string]any{"queries": queries})
		if err != nil {
			return err
		}

		resp, err := http.Post("https://api.osv.dev/v1/querybatch", "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("checking osv.dev: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("osv.dev returned %s", resp.Status)
		}

		var batch osvBatchResponse
		if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
			return err
		}

		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("audited %d package(s)", len(candidates))))

		found := 0
		for i, result := range batch.Results {
			if len(result.Vulns) == 0 {
				continue
			}
			c := candidates[i]
			found++
			fmt.Println(ui.Red(fmt.Sprintf("%s %s (%s): %d known advisory(ies)", c.name, c.version, c.ecosystem, len(result.Vulns))))
			for _, v := range result.Vulns {
				fmt.Printf("  - %s (https://osv.dev/vulnerability/%s)\n", v.ID, v.ID)
			}
		}

		if found == 0 {
			fmt.Println(ui.Orange("no known vulnerabilities found"))
		} else {
			fmt.Println(ui.Dim("advisory details fetched from osv.dev; upgrade with `poly upgrade` where a fix is available"))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(auditCmd)
}
