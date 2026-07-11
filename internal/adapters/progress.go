package adapters

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const progressBarWidth = 28

// progressWriter renders a live-updating download progress bar to stderr
// as bytes are written through it. It implements io.Writer so it can sit
// alongside the destination file in an io.MultiWriter.
type progressWriter struct {
	label   string
	total   int64
	written int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)
	pw.render()
	return n, nil
}

func (pw *progressWriter) render() {
	if pw.total <= 0 {
		fmt.Fprintf(os.Stderr, "\r%s  %s", pw.label, humanBytes(pw.written))
		return
	}

	frac := float64(pw.written) / float64(pw.total)
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * progressBarWidth)
	bar := strings.Repeat("#", filled) + strings.Repeat(" ", progressBarWidth-filled)

	fmt.Fprintf(os.Stderr, "\r%s  [%s] %3.0f%%  %s/%s",
		pw.label, bar, frac*100, humanBytes(pw.written), humanBytes(pw.total))
}

func (pw *progressWriter) done() {
	pw.render()
	fmt.Fprintln(os.Stderr)
}

func humanBytes(n int64) string {
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

// copyWithProgress copies src into dst while rendering a progress bar to
// stderr, using total as the expected size (-1/0 if unknown).
func copyWithProgress(dst io.Writer, src io.Reader, total int64, label string) error {
	pw := &progressWriter{label: label, total: total}
	mw := io.MultiWriter(dst, pw)
	_, err := io.Copy(mw, src)
	pw.done()
	return err
}
