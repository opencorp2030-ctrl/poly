# poly

One command, every package manager. `poly` installs from pip, npm, and
checksum-verified binary releases behind a single unified command, on
macOS, Linux, and Windows.

```
$ poly install ripgrep
installed ripgrep 15.1.0 (via tap)
note: tap binaries are installed to ~/.poly/bin — make sure it's on your PATH

$ poly install requests@2.31.0
installed requests 2.31.0 (via pip)

$ poly install npm:cowsay
installed cowsay 1.6.0 (via npm)

$ poly list
NAME      VERSION  ADAPTER  INSTALLED
cowsay    1.6.0    npm      2026-07-11 15:52
requests  2.31.0   pip      2026-07-11 15:52
ripgrep   15.1.0   tap      2026-07-11 15:52
```

## Install

Requires Go 1.21+. There's no hosted installer yet, so build it from source
and put the binary somewhere on your `PATH`:

```
git clone https://github.com/opencorp2030-ctrl/poly.git
cd poly
go build -o poly .
./poly version           # sanity check: runs the binary in place

# put it on PATH, e.g.:
cp poly /opt/homebrew/bin/poly   # macOS (Homebrew's bin, no sudo needed)
# or: sudo cp poly /usr/local/bin/poly       # macOS/Linux, needs sudo
# or: add this folder to PATH yourself

poly version              # now works from anywhere
```

Cross-compile for another OS/arch with `GOOS`/`GOARCH`, or run
`scripts/build-all.sh` to build all five targets (darwin/linux/windows ×
amd64/arm64) into `dist/`.

## How package resolution works

`poly install <name>` tries each adapter's `Search` in order — **tap →
pip → npm** — and installs through the first one that reports the
package exists. Force a specific backend with a prefix:

| Command | Resolves to |
|---|---|
| `poly install ripgrep` | first match: tap → pip → npm |
| `poly install tap:ripgrep` | forced binary download from the tap formula |
| `poly install pip:requests` | forced `pip install requests` |
| `poly install npm:lodash` | forced `npm install -g lodash` |

Append `@version` to pin: `poly install requests@2.31.0`. The pip and npm
adapters pass that straight through; the tap adapter only offers the
version pinned in its formula and errors if you ask for a different one.

## Commands

| Command | Does |
|---|---|
| `poly install [adapter:]name[@version]` | install, auto-routed or forced |
| `poly remove name` | uninstall, via whichever adapter installed it |
| `poly list` | show everything poly has installed, and through what |
| `poly search [adapter:]name` | check existence and latest version across adapters |
| `poly version` | print the poly build version |

State lives in `~/.poly/manifest.json`. Tap binaries land in `~/.poly/bin`
(add it to your `PATH`).

## Adapters

- **pip** — shells out to `pip3`/`pip`. Search hits the PyPI JSON API
  (`pypi.org/pypi/<name>/json`), an exact-name lookup — PyPI has no
  public free-text search API anymore.
- **npm** — shells out to `npm install -g`. Search hits
  `registry.npmjs.org/<name>/latest`.
- **tap** — installs prebuilt binaries directly from a pinned URL,
  verified against a SHA-256 checksum, then extracted (`.tar.gz`/`.zip`)
  or copied into `~/.poly/bin`. No Python or Node runtime needed.

## Adding a tap formula

Drop a YAML file into `~/.poly/taps/<name>.yaml` (this overrides any
built-in formula of the same name — see `internal/registry/embedded/taps`
for the ones that ship with poly):

```yaml
name: ripgrep
description: "Line-oriented search tool that recursively searches directories for a regex pattern"
homepage: "https://github.com/BurntSushi/ripgrep"
version: "15.1.0"
binary: rg
artifacts:
  darwin_arm64:
    url: "https://github.com/BurntSushi/ripgrep/releases/download/15.1.0/ripgrep-15.1.0-aarch64-apple-darwin.tar.gz"
    sha256: "378e973289176ca0c6054054ee7f631a065874a352bf43f0fa60ef079b6ba715"
  darwin_amd64: { url: "...", sha256: "..." }
  linux_amd64: { url: "...", sha256: "..." }
  linux_arm64: { url: "...", sha256: "..." }
  windows_amd64: { url: "...", sha256: "..." }
```

The artifact key is `<GOOS>_<GOARCH>`. `poly install <name>` picks the
entry matching the machine it's running on.

## Project layout

```
main.go                          entrypoint
cmd/                              cobra commands (install, remove, list, search, version)
internal/manifest/                ~/.poly/manifest.json read/write
internal/adapters/                Adapter interface + pip, npm, tap implementations
internal/registry/embedded/       tap formulas built into the binary
site/                             marketing/docs site (site/index.html)
scripts/build-all.sh              cross-platform release build
```

## Status

Early. Working: install/remove/list/search across pip, npm, and tap;
version pinning; cross-platform builds. Not yet built: a hosted
installer/registry, more built-in tap formulas, and a package search UI
on the website.
