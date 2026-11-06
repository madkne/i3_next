# Security

## Threat model (v1)

### Assets
- User session and desktop state (workspaces, windows, outputs).
- Configuration files under `~/.config/i3-next/` and `/etc/i3-next/`.
- Daemon control socket `$XDG_RUNTIME_DIR/i3-next/i3-next.sock`.
- Child processes spawned by the supervisor (sway, quickshell, helpers).
- Credentials/secrets that may appear in the environment of children.

### Trust boundaries
- **Daemon socket (JSON-RPC):** trusted to the session owner's UID only; peer credentials (`SO_PEERCRED`) are checked. Socket directory is `0700`, socket `0600`.
- **D-Bus:** session and system buses; inputs validated; privileged operations go through polkit/logind.
- **Configuration:** parsed strictly; invalid config falls back to last valid/defaults without a restart loop.
- **Child processes:** executed with argument lists (no shell); paths validated with `LookPath`; environment allowlisted; sensitive variables redacted in logs.
- **Plugins:** disabled by default in v1; only built-ins run.
- **Filesystem:** paths resolved without traversal/symlink escape; runtime/state/config directories created with `0700`.

### STRIDE summary
- **Spoofing:** socket UID check; D-Bus sender validation.
- **Tampering:** atomic config writes; restrictive permissions; no untrusted plugin code in v1.
- **Repudiation:** structured logs with component/version/session fields.
- **Information disclosure:** redaction of secrets in logs; no telemetry.
- **Denial of service:** bounded queues/backpressure; restart limits; shutdown timeouts.
- **Elevation of privilege:** no `sudo` in-process; polkit for power actions; fail-closed lock screen.

### Assumptions
- The local user account is trusted; the threat model does not defend against a malicious same-UID process beyond socket permissions.
- Sway, Quickshell, and helper binaries are trusted, installed from the package manager.
- No network-facing services are exposed by the daemon.
