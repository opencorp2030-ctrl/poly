package adapters

import (
	"fmt"
	"io"
	"os"
	"strings"

	"poly/internal/ui"
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
	label := ui.Orange(pw.label)

	if pw.total <= 0 {
		fmt.Fprintf(os.Stderr, "\r%s  %s", label, ui.HumanBytes(pw.written))
		return
	}

	frac := float64(pw.written) / float64(pw.total)
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * progressBarWidth)
	bar := ui.Orange(strings.Repeat("#", filled)) + strings.Repeat(" ", progressBarWidth-filled)

	fmt.Fprintf(os.Stderr, "\r%s  [%s] %3.0f%%  %s/%s",
		label, bar, frac*100, ui.HumanBytes(pw.written), ui.HumanBytes(pw.total))
}

func (pw *progressWriter) done() {
	pw.render()
	fmt.Fprintln(os.Stderr)
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
