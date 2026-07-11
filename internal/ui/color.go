// Package ui provides small ANSI color helpers so poly's own output
// (as opposed to the raw output of pip/npm/downloads it wraps) is
// visually distinct in the terminal, the way Homebrew colors its own
// "==>" status lines.
package ui

import "os"

var enabled = shouldColor()

func shouldColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("CLICOLOR_FORCE") == "1" {
		return true
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

const (
	codeOrange = "\033[38;5;208m"
	codeGreen  = "\033[38;5;114m"
	codeRed    = "\033[38;5;203m"
	codeDim    = "\033[2m"
	codeBold   = "\033[1m"
	codeReset  = "\033[0m"
)

func wrap(code, s string) string {
	if !enabled {
		return s
	}
	return code + s + codeReset
}

// Orange marks poly's own status output -- installs, notes, login state --
// matching the amber accent used across the site and branding.
func Orange(s string) string { return wrap(codeOrange, s) }
func Green(s string) string  { return wrap(codeGreen, s) }
func Red(s string) string    { return wrap(codeRed, s) }
func Bold(s string) string   { return wrap(codeBold, s) }
func Dim(s string) string    { return wrap(codeDim, s) }

// Arrow is poly's "==>" status prefix, in orange bold like Homebrew's own.
func Arrow() string { return wrap(codeOrange+codeBold, "==>") }
