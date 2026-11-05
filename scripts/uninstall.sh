#!/usr/bin/env bash
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"

for b in i3-nextd i3-nextctl i3-next-session i3-next-doctor; do
    rm -f "$PREFIX/bin/$b"
done

echo "Uninstalled i3_next binaries (user configuration preserved)"
