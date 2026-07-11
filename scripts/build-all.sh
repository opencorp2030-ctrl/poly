#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist

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
  echo "building ${out} (GOOS=${os} GOARCH=${arch})"
  GOOS="$os" GOARCH="$arch" go build -ldflags "-X poly/cmd.Version=0.1.0" -o "$out" .
done
