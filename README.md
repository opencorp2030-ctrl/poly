# poly

**[English](README.md)** · [Français](README.fr.md)

One command, every package manager. `poly` installs from pip, npm,
Homebrew, and checksum-verified binary releases behind a single unified
command, on macOS, Linux, and Windows.

Site + account management: **[poly.candygate.eu](https://poly.candygate.eu)**

```
$ poly install ripgrep
downloading ripgrep  [############################] 100%  1.7MiB/1.7MiB
installed ripgrep 15.1.0 (via tap)
note: tap binaries are installed to ~/.poly/bin — make sure it's on your PATH

$ poly install requests@2.31.0
installed requests 2.31.0 (via pip)

$ poly install npm:cowsay
installed cowsay 1.6.0 (via npm)

$ poly install brew:jq
installed jq 1.8.2 (via brew)

$ poly list
NAME      VERSION  ADAPTER  INSTALLED
cowsay    1.6.0    npm      2026-07-11 15:52
jq        1.8.2    brew     2026-07-11 17:30
requests  2.31.0   pip      2026-07-11 15:52
ripgrep   15.1.0   tap      2026-07-11 15:52
```

## Install

**macOS** — download [`poly-macos.pkg`](https://github.com/opencorp2030-ctrl/poly/releases/latest/download/poly-macos.pkg),
open it, click through the installer. Installs a universal
(Intel + Apple Silicon) binary to `/usr/local/bin/poly`. It's
unsigned/unnotarized (no Apple Developer account yet), so Gatekeeper
will warn on first open — right-click → Open, or
`System Settings → Privacy & Security → Open Anyway`.

**Windows** — download [`poly-setup.exe`](https://github.com/opencorp2030-ctrl/poly/releases/latest/download/poly-setup.exe),
run it. Installs to `%LOCALAPPDATA%\Poly` and adds it to your user
`PATH` (no admin rights needed). It's unsigned, so SmartScreen will
warn — "More info" → "Run anyway". *(Built cross-platform via NSIS;
not yet verified on a real Windows machine — please report issues.)*

**From source** (any OS, needs Go 1.21+):

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

Rebuild the installers yourself with `VERSION=x.y.z installers/build.sh`
(needs `makensis` for the Windows one: `brew install makensis`).

Cross-compile for another OS/arch with `GOOS`/`GOARCH`, or run
`scripts/build-all.sh` to build all five targets (darwin/linux/windows ×
amd64/arm64) into `dist/`.

## How package resolution works

`poly install <name>` tries each adapter's `Search` in order — **tap →
brew → pip → npm → cargo → go** — and installs through the first one
that reports the package exists. Force a specific backend with a prefix:

| Command | Resolves to |
|---|---|
| `poly install ripgrep` | first match: tap → brew → pip → npm → cargo → go |
| `poly install tap:ripgrep` | forced binary download from the tap formula |
| `poly install brew:wget` | forced `brew install wget` |
| `poly install pip:requests` | forced `pip install requests` |
| `poly install npm:lodash` | forced `npm install -g lodash` |
| `poly install cargo:ripgrep` | forced `cargo install ripgrep` |
| `poly install go:golang.org/x/tools/cmd/goimports` | forced `go install <module>@latest` |

Append `@version` to pin: `poly install requests@2.31.0`. pip, npm, and
cargo pass that straight through; the tap adapter only offers the
version pinned in its formula and errors if you ask for a different
one; `go` treats it as the module's version/tag.

## Commands

| Command | Does |
|---|---|
| `poly install [[adapter:]name[@version] ...]` | install packages, auto-routed or forced; no args installs from `poly.json` |
| `poly remove name` | uninstall, via whichever adapter installed it |
| `poly list` | show everything poly has installed, and through what |
| `poly search [adapter:]name` | check existence and latest version across adapters |
| `poly info [adapter:]name` | version, summary, homepage, and install status, per adapter |
| `poly init` | write `poly.json` from your currently installed packages |
| `poly upgrade` | update every installed package to its adapter's latest version |
| `poly self-update` | update the poly binary itself to the latest GitHub release |
| `poly account` | show your signed-in email/username/bio/plan |
| `poly login` / `poly logout` | sign in/out of your Poly account (unlocks Pro) |
| `poly version` | print the poly build version and, if signed in, your account/plan |

State lives in `~/.poly/manifest.json`. Tap binaries land in `~/.poly/bin`
(add it to your `PATH`). Login credentials live in `~/.poly/credentials.json`
(mode 0600). `poly self-update` and, for Pro accounts, `poly upgrade` also
run automatically in the background (throttled to ~once/day, logged to
`~/.poly/auto-update.log`) so you don't have to remember to run them.

## Reproducible installs with poly.json

`poly init` writes a `poly.json` in the current directory listing every
package poly has installed, pinned to its exact version:

```json
{
  "packages": [
    "tap:ripgrep@15.1.0",
    "npm:eslint@9.2.0",
    "pip:requests@2.31.0"
  ]
}
```

Commit it, and anyone (or any CI machine) can reproduce the same set with
a plain `poly install` — no arguments needed.

## Poly Pro

Poly is and stays 100% free and open source — the whole built-in tap
catalog (`ripgrep`, `fd`, `jq`, ...) is free too, on purpose: Pro isn't
about withholding tools people expect for free. The one real, measurable
perk today:

- `poly install a b c` (multiple packages in one command) installs them
  **sequentially** on the free tier; signed in with an active Pro plan
  (`poly login`), the same command installs them **concurrently**, and
  `poly upgrade` runs automatically in the background instead of only
  on demand.

Manage your account (sign up, check your plan) at
[poly.candygate.eu/account.html](https://poly.candygate.eu/account.html),
backed by Supabase Auth — see `internal/account/account.go` for the
client-side logic and the `public.profiles` table for where plan status
lives.

## Adapters

- **pip** — shells out to `pip3`/`pip`, streaming its output live so you
  see pip's own download progress. Search hits the PyPI JSON API
  (`pypi.org/pypi/<name>/json`), an exact-name lookup — PyPI has no
  public free-text search API anymore.
- **npm** — shells out to `npm install -g`, same live-streamed output.
  Search hits `registry.npmjs.org/<name>/latest`.
- **brew** — shells out to a local `brew install`/`brew uninstall`, same
  live-streamed output. Search hits the public
  `formulae.brew.sh/api/formula/<name>.json`, so it works even without
  brew installed (only actually installing needs it).
- **cargo** — shells out to `cargo install --force`/`cargo uninstall`.
  Search hits the public `crates.io/api/v1/crates/<name>` API.
- **go** — shells out to `go install <module>@version`. Packages here are
  full module import paths, not short names (e.g.
  `golang.org/x/tools/cmd/goimports`), so this one is realistically only
  used via an explicit `go:` prefix. Search walks the path upward against
  the public Go module proxy until it finds the enclosing module, the
  same resolution `go install` does internally.
- **tap** — installs prebuilt binaries directly from a pinned URL, with a
  live byte-progress bar during download, verified against a SHA-256
  checksum, then extracted (`.tar.gz`/`.zip`) or copied into
  `~/.poly/bin`. No Python, Node, Rust, or Homebrew needed.

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
tier: free   # omit for free, or "pro" to gate it behind an active Pro plan
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
entry matching the machine it's running on. Built-in formulas today, all
free: `ripgrep`, `fd`, `jq` — see `internal/registry/embedded/taps`.

## Project layout

```
main.go                          entrypoint
cmd/                              cobra commands (install, upgrade, init, info, account, self-update, ...)
internal/manifest/                ~/.poly/manifest.json read/write
internal/lockfile/                poly.json read/write
internal/adapters/                Adapter interface + pip, npm, brew, cargo, go, tap implementations
internal/registry/embedded/       tap formulas built into the binary
internal/account/                 Supabase Auth client (login/plan/profile)
internal/selfupdate/              downloads + verifies + replaces the poly binary
installers/                       macOS .pkg and Windows .exe installer sources
site/                             marketing site, docs, account, community pages
scripts/build-all.sh              cross-platform release build
```

## Status

Working: install/remove/list/search/info/upgrade/init across pip, npm,
brew, cargo, go, and tap; version pinning; automatic self-update and
(Pro) automatic package upgrades; macOS/Windows installers; a Supabase-
backed account system with a community directory. Not yet built: signed/
notarized installers, a package search UI covering every adapter's full
catalog (vs. exact-name lookups), and Linux distro package managers
(apt/dnf/pacman).
