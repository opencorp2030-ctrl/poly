#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist

VERSION="${VERSION:-0.1.0}"

targets=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
)

for target in "${targets[@]}"; do
  os="${target%% *}"
  arch="${target##* }"
  out="dist/poly-${os}-${arch}"
  if [ "$os" = "windows" ]; then
    out="${out}.exe"
  fi
  echo "building ${out} (GOOS=${os} GOARCH=${arch}, version ${VERSION})"
  GOOS="$os" GOARCH="$arch" go build -ldflags "-X poly/cmd.Version=${VERSION}" -o "$out" .
done

echo "== checksums =="
if command -v sha256sum >/dev/null 2>&1; then
  (cd dist && sha256sum poly-* > checksums.txt)
else
  (cd dist && shasum -a 256 poly-* > checksums.txt)
fi
cat dist/checksums.txt
