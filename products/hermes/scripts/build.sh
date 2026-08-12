#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
VERSION="$(tr -d '\n' < VERSION)"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"
BUILD_TIME="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
LDFLAGS="-s -w -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.Version=$VERSION -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.Commit=$COMMIT -X github.com/mca-rolando/HermesDDNS/internal/buildinfo.BuildTime=$BUILD_TIME"
mkdir -p bin
go test ./...
go build -trimpath -ldflags "$LDFLAGS" -o bin/hermesddns ./cmd/hermesddns
go build -trimpath -ldflags "$LDFLAGS" -o bin/hermesctl ./cmd/hermesctl
printf 'Built HermesDDNS %s (%s)\n' "$VERSION" "$COMMIT"
