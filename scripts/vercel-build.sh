#!/bin/sh
# Vercel build: the build image has no Go toolchain, so download one,
# then pre-render the site into public/ (see cmd/export).
set -eu

GO_VERSION="1.25.5"

echo "Installing Go ${GO_VERSION}..."
curl -fsSL "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz" | tar -xz -C /tmp
export PATH="/tmp/go/bin:${PATH}"
go version

go run ./cmd/export
