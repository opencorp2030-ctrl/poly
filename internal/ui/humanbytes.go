package ui

import "fmt"

// HumanBytes formats a byte count the way download tools do, e.g.
// "512B", "1.7MiB", "2.4GiB". Shared by the download progress bar and
// `poly cache size` so they render sizes identically.
func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%dB", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
