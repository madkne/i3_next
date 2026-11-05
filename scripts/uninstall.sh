#!/usr/bin/env bash
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"

for b in i3-nextd i3-nextctl i3-next-session i3-next-doctor; do
    rm -f "$PREFIX/bin/$b"
done

rm -rf "$PREFIX/share/i3-next/sway"

echo "Uninstalled i3_next binaries and Sway configuration (user configuration preserved)"
