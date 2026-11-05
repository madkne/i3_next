# Security Policy

## Supported versions

Only the latest development state on the default branch is supported. Once
versioned releases are published, this table will list the supported release
lines.

| Version | Supported |
| ------- | --------- |
| main    | Yes       |

## Reporting a vulnerability

Please do **not** open a public issue for security vulnerabilities.

Report security issues privately by contacting the maintainers through the
repository's private vulnerability reporting channel, or by email to the
address listed in the project metadata. Include:

- A description of the issue and its impact
- Steps to reproduce or a proof of concept
- The affected version or commit
- Any suggested mitigation

You will receive an acknowledgment within a reasonable time, and we will
coordinate a fix and disclosure timeline with you.

## Security practices

i3_next follows these practices in development:

- Least privilege: runtime files and sockets are created with restrictive
  permissions.
- Input validation: IPC, D-Bus, configuration, and plugin inputs are
  validated.
- No shell interpolation: external processes are executed with argument
  lists, never via shell strings.
- Secrets are never logged.
- Dependencies are reviewed and vulnerability-scanned in CI.
- Privileged operations require explicit, OS-standard authorization.

## Threat model

The v1 threat model is documented in `docs/SECURITY.md` (when available).
Until that document is published, treat the daemon socket, configuration
files, and plugin inputs as trusted-user-only surfaces.

## Disclosure

We follow coordinated disclosure. Please allow maintainers reasonable time to
fix reported issues before public disclosure.
