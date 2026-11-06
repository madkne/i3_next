#!/usr/bin/env bash
set -euo pipefail

# Check the i3_next naming rule:
#   i3_next  -> Go module / QML module / application name
#   i3-next  -> directories, binaries, installed paths, sockets, files
#   I3NEXT_* -> environment variables

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

fail=0
report() {
    echo "naming: $*" >&2
    fail=1
}

# Naming document must exist.
if [[ ! -f docs/development/naming.md ]]; then
    report "missing docs/development/naming.md"
fi

# Go module name must be i3_next.
module_name="$(sed -n 's/^module[[:space:]]\+//p' go.mod | head -n 1)"
if [[ "$module_name" != "i3_next" ]]; then
    report "go.mod module must be 'i3_next' (got '$module_name')"
fi

# QML module name must be i3_next.
qml_module="$(grep -m 1 -E '^[[:space:]]*module[[:space:]]+' ui/quickshell/qmldir | awk '{print $2}')"
if [[ "$qml_module" != "i3_next" ]]; then
    report "ui/quickshell/qmldir module must be 'i3_next' (got '$qml_module')"
fi

# Command directories must use the hyphenated form.
if [[ -d cmd ]]; then
    for dir in cmd/*/; do
        [[ -d "$dir" ]] || continue
        base="$(basename "$dir")"
        case "$base" in
            i3-next*) ;;
            *) report "cmd/ directory must be named 'i3-next*' (got '$base')" ;;
        esac
    done
fi

# Go import paths must use the module name, never the hyphenated form.
if grep -RIn '"i3-next/' --include='*.go' . >/tmp/i3-next-imports 2>/dev/null; then
    report "Go imports must use 'i3_next', not 'i3-next':"
    cat /tmp/i3-next-imports >&2
fi
rm -f /tmp/i3-next-imports

# Environment variables must use I3NEXT_, not I3_NEXT_ or I3-NEXT_.
if grep -RInE 'I3[_-]NEXT_' --include='*.go' --include='*.sh' --include='*.yaml' --include='*.yml' --include='*.qml' --include='*.md' . 2>/dev/null \
    | grep -v 'scripts/check-naming.sh' | grep -v 'docs/development/naming.md' >/tmp/i3-next-env; then
    report "environment variables must use 'I3NEXT_', not 'I3_NEXT_' or 'I3-NEXT_':"
    cat /tmp/i3-next-env >&2
fi
rm -f /tmp/i3-next-env

# paths.go must keep the two forms in their proper places.
paths_file="internal/core/config/paths.go"
if [[ -f "$paths_file" ]]; then
    if ! grep -q 'AppName = "i3_next"' "$paths_file"; then
        report "$paths_file: AppName must be \"i3_next\""
    fi
    for name in RuntimeDirName ConfigDirName DataDirName StateDirName CacheDirName; do
        if ! grep -qE "${name}[[:space:]]*=[[:space:]]*\"i3-next\"" "$paths_file"; then
            report "$paths_file: $name must be \"i3-next\""
        fi
    done
    for name in BinaryDaemon BinaryControl BinarySession BinaryDoctor BinaryPluginHost; do
        if ! grep -qE "${name}[[:space:]]*=[[:space:]]*\"i3-next" "$paths_file"; then
            report "$paths_file: $name must start with \"i3-next\""
        fi
    done
fi

# QML imports must use the module name, never the hyphenated form.
if grep -RInE 'import[[:space:]]+i3-next' ui >/tmp/i3-next-qml 2>/dev/null; then
    report "QML imports must use 'i3_next', not 'i3-next':"
    cat /tmp/i3-next-qml >&2
fi
rm -f /tmp/i3-next-qml

if [[ "$fail" -ne 0 ]]; then
    echo "naming check failed" >&2
    exit 1
fi

echo "naming check passed"
