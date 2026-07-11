// Package embedded ships poly's small built-in set of tap formulas
// (binary download definitions) directly inside the compiled binary, the
// way Homebrew ships its "core" tap. Users can add more by dropping
// additional YAML files into ~/.poly/taps.
package embedded

import "embed"

//go:embed taps/*.yaml
var Taps embed.FS
