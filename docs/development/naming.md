# Naming

i3_next uses two spellings of its name. They are not interchangeable.

## Rule

| Form | Use for | Examples |
|---|---|---|
| `i3_next` | Go module path, QML module name, application/product name, Go identifiers | `module i3_next`, `module i3_next` in `qmldir`, `AppName = "i3_next"`, `i3_next session manager` |
| `i3-next` | Directories on disk, binary names, installed paths, XDG directory names, socket/database/lock file names | `cmd/i3-nextd`, `/usr/share/i3-next`, `~/.config/i3-next`, `i3-next.sock`, `i3-next.lock` |
| `I3NEXT_*` | Environment variables (uppercase, no separator) | `I3NEXT_UI_DEBUG=1` |

## Guidance

- Go code and imports always use the module name `i3_next` (for example `i3_next/internal/version`).
- Filesystem locations and executable names always use the hyphenated form `i3-next`.
- The application/product name shown to users is `i3_next`.
- Environment variables use `I3NEXT_` followed by the setting name in uppercase.
- Do not create new spellings such as `i3 next`, `i3_nextd`, `i3-next_next`, or `I3_NEXT_*`.

## Enforcement

The rule is checked by `scripts/check-naming.sh` (also available as `make naming` and `make check-naming`). Run it before committing.
