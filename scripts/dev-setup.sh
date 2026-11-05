#!/usr/bin/env bash
set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
    echo "go is required" >&2
    exit 1
fi

echo "Downloading Go modules..."
go mod download

echo "Building project..."
make build

echo "Development setup complete."
