#!/usr/bin/env bash
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
DESTDIR="${DESTDIR:-}"

for cmd in make; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "missing dependency: $cmd" >&2
        exit 1
    fi
done

make build
make install PREFIX="$PREFIX" DESTDIR="$DESTDIR"

echo "Installed i3_next to ${DESTDIR}${PREFIX}"
