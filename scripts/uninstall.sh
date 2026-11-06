#!/usr/bin/env bash
set -euo pipefail

PREFIX="${PREFIX:-/usr/local}"
DESTDIR="${DESTDIR:-}"

for b in i3-nextd i3-nextctl i3-next-session i3-next-doctor; do
    rm -f "${DESTDIR}${PREFIX}/bin/$b"
done

rm -rf "${DESTDIR}${PREFIX}/share/i3-next"
rm -f "${DESTDIR}${PREFIX}/share/wayland-sessions/i3-next.desktop"
rm -rf "${DESTDIR}${PREFIX}/share/doc/i3-next"

echo "Uninstalled i3_next (user configuration preserved)"
