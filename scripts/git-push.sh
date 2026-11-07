#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "======================================"
echo " i3_next - Pre-push validation"
echo "======================================"

# 0. Refuse to push with uncommitted changes (push would not match the tree).
if ! git diff --quiet || ! git diff --cached --quiet; then
    echo "error: working tree has uncommitted changes; commit or stash first" >&2
    exit 1
fi

# 1. Synchronize Go dependencies
echo ""
echo "[1/6] Synchronizing dependencies..."
go mod tidy
go mod download
go mod verify

# 2. gofmt check (do not silently rewrite; fail and tell the user)
echo ""
echo "[2/6] Checking formatting..."
dirty="$(gofmt -l .)"
if [ -n "$dirty" ]; then
    echo "gofmt: files need formatting:" >&2
    echo "$dirty" >&2
    echo "run: make fmt" >&2
    exit 1
fi

# 3. Static analysis (vet, lint, staticcheck)
echo ""
echo "[3/6] Running vet / lint / staticcheck..."

# Ensure GOPATH/bin is on PATH so locally installed tools
# (golangci-lint, staticcheck, ...) are found by `command -v`.
if command -v go >/dev/null 2>&1; then
    GOPATH_BIN="$(go env GOPATH)/bin"
    case ":$PATH:" in
        *":$GOPATH_BIN:"*) ;;
        *) export PATH="$PATH:$GOPATH_BIN" ;;
    esac
fi

go vet ./...
if command -v golangci-lint >/dev/null 2>&1; then
    golangci-lint run ./...
else
    echo "golangci-lint not installed; skipping (CI still runs it)" >&2
fi
if command -v staticcheck >/dev/null 2>&1; then
    staticcheck ./...
else
    echo "staticcheck not installed; skipping (CI still runs it)" >&2
fi

# 4. Tests
echo ""
echo "[4/6] Running tests (race)..."
go test -race ./...

# 5. Build (also proves ldflags/toolchain are consistent)
echo ""
echo "[5/6] Building project..."
make build

# 6. Push
echo ""
echo "[6/6] Pushing to remote..."
git push "$@"

echo ""
echo "======================================"
echo " Push completed successfully!"
echo "======================================"