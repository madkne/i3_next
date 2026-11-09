#!/usr/bin/env bash
set -euo pipefail

if ! command -v go >/dev/null 2>&1; then
    echo "go is required" >&2
    exit 1
fi

echo "Downloading Go modules..."
go mod download

echo "Configuring git hooks..."
git config core.hooksPath .githooks 2>/dev/null || true

echo "Installing development tools..."
make install-lint-tools

echo "Building project..."
make build

echo "Development setup complete."
echo "Run 'make check' before committing."
