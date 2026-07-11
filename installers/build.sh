#!/usr/bin/env bash
# Builds the macOS .pkg and Windows .exe installers.
# Requires: go, lipo/pkgbuild (ships with macOS), makensis (brew install makensis).
set -euo pipefail
cd "$(dirname "$0")/.."

VERSION="${VERSION:-0.1.0}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "== macOS .pkg =="
mkdir -p "$WORK/pkgroot/usr/local/bin"
GOOS=darwin GOARCH=amd64 go build -ldflags "-X poly/cmd.Version=${VERSION}" -o "$WORK/poly-amd64" .
GOOS=darwin GOARCH=arm64 go build -ldflags "-X poly/cmd.Version=${VERSION}" -o "$WORK/poly-arm64" .
lipo -create "$WORK/poly-amd64" "$WORK/poly-arm64" -output "$WORK/pkgroot/usr/local/bin/poly"
chmod 755 "$WORK/pkgroot/usr/local/bin/poly"

mkdir -p installers/macos
pkgbuild --root "$WORK/pkgroot" \
  --identifier eu.candygate.poly \
  --version "$VERSION" \
  --install-location / \
  "installers/macos/poly-macos-${VERSION}.pkg"
cp "installers/macos/poly-macos-${VERSION}.pkg" installers/macos/poly-macos.pkg
echo "-> installers/macos/poly-macos-${VERSION}.pkg (+ stable alias poly-macos.pkg)"

echo "== Windows .exe =="
GOOS=windows GOARCH=amd64 go build -ldflags "-X poly/cmd.Version=${VERSION}" -o "$WORK/poly-windows-amd64.exe" .
(cd installers/windows && makensis -DVERSION="$VERSION" "-DSOURCE_EXE=$WORK/poly-windows-amd64.exe" poly-installer.nsi)
cp "installers/windows/poly-setup-${VERSION}.exe" installers/windows/poly-setup.exe
echo "-> installers/windows/poly-setup-${VERSION}.exe (+ stable alias poly-setup.exe)"
