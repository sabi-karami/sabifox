#!/usr/bin/env bash
# Builds the Linux core into dist/: sabifox-core-linux-amd64.tar.gz,
# sabifox-core-linux-arm64.tar.gz, and sabifox-core-linux-armv7.tar.gz
# (32-bit ARM routers; docs/OPENWRT.md), each an sabifox-core-linux-<arch>
# folder with the core, core/scripts/install-systemd.sh, and the license;
# install.sh (at the repository root) installs one. The version is
# "major.minor.patch" from app/pubspec.yaml.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="$(sed -n 's/^version:[[:space:]]*\([0-9]*\.[0-9]*\.[0-9]*\).*/\1/p' "$root/app/pubspec.yaml")"
[ -n "$version" ] || { echo "no version in app/pubspec.yaml" >&2; exit 1; }

mkdir -p "$root/dist"
for arch in amd64 arm64 armv7; do
  name="sabifox-core-linux-$arch"
  stage="$(mktemp -d)"
  mkdir "$stage/$name"
  (cd "$root/core" && CGO_ENABLED=0 GOOS=linux GOARCH="${arch/armv7/arm}" GOARM=7 go build -trimpath \
    -ldflags "-s -w -X github.com/sabi-karami/sabifox/core/internal/version.Current=$version" \
    -o "$stage/$name/sabifox-core" ./cmd/sabifox-core)
  install -m 0755 "$root/core/scripts/install-systemd.sh" "$stage/$name/install-systemd.sh"
  install -m 0644 "$root/LICENSE" "$stage/$name/LICENSE"
  tar -czf "$root/dist/$name.tar.gz" -C "$stage" --owner=0 --group=0 "$name"
  rm -rf "$stage"
  echo "Built dist/$name.tar.gz ($version)"
done
