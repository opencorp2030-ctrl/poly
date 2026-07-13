package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"poly/internal/cache"
	"poly/internal/ui"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Inspect or clear poly's download cache (~/.poly/cache)",
}

var cacheSizeCmd = &cobra.Command{
	Use:   "size",
	Short: "Show how much disk space the download cache is using",
	RunE: func(cmd *cobra.Command, args []string) error {
		size, err := cache.Size()
		if err != nil {
			return err
		}
		count, err := cache.Count()
		if err != nil {
			return err
		}
		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange(fmt.Sprintf("%s across %d cached download(s)", humanBytes(size), count)))
		return nil
	},
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Delete every cached download",
	RunE: func(cmd *cobra.Command, args []string) error {
		size, _ := cache.Size()
		if err := cache.Clean(); err != nil {
			return err
		}
		fmt.Printf("%s %s\n", ui.Arrow(), ui.Orange("cache cleared, freed "+humanBytes(size)))
		return nil
	},
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n2 := n / unit; n2 >= unit; n2 /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func init() {
	cacheCmd.AddCommand(cacheSizeCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	rootCmd.AddCommand(cacheCmd)
}
