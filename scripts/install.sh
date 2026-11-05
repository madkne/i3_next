#!/usr/bin/env bash
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"

for cmd in sway go make; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "missing dependency: $cmd" >&2
        exit 1
    fi
done

make build

install -d "$PREFIX/bin"
install -m 0755 build/i3-nextd build/i3-nextctl build/i3-next-session build/i3-next-doctor "$PREFIX/bin/"

install -d "$PREFIX/share/i3-next/sway"
install -m 0644 configs/sway/config configs/sway/outputs configs/sway/keybindings configs/sway/window-rules "$PREFIX/share/i3-next/sway/"

echo "Installed i3_next binaries to $PREFIX/bin"
echo "Installed i3_next Sway configuration to $PREFIX/share/i3-next/sway"
