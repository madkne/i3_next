#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "======================================"
echo " i3_next - Pre-push validation"
echo "======================================"

# 1. Synchronize Go dependencies
echo ""
echo "[1/5] Synchronizing dependencies..."
go mod tidy
go mod download
go mod verify

# 2. Format Go source files
echo ""
echo "[2/5] Formatting Go source..."
make fmt

# 3. Run validation
echo ""
echo "[3/5] Running checks..."
make check

# 4. Build binaries
echo ""
echo "[4/5] Building project..."
make build

# 5. Push
echo ""
echo "[5/5] Pushing to remote..."

git push "$@"

echo ""
echo "======================================"
echo " Push completed successfully!"
echo "======================================"