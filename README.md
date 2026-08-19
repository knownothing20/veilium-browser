# Veilium Browser

Veilium is a local-first, multi-profile privacy browser workspace built with Go, Wails, React, and managed Chromium runtimes.

> Clean-room project: public projects may inform requirements and architecture, while Veilium implementation is developed independently.

## Current baseline

`main` includes isolated browser Profiles, reviewed/custom Chromium Provider contracts, exact package integrity, OS-backed credentials, HTTP/HTTPS/SOCKS5 proxy bridges, supervised Xray/sing-box subsets, browser runtime supervision, identity and Network Evidence, recoverable lifecycle/snapshot/restore, portable Profiles/templates, bounded multi-Profile operations, and a Simplified Chinese desktop workspace.

The current reviewed Chromium Provider is an exact Windows amd64 **stock Chromium Snapshot**. Advanced fingerprint overrides are not claimed for that Provider.

## Development

All current project status, architecture boundaries, priorities, validation debt, and the Fingerprint Chromium plan live in one file:

- [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md)

Do not use old commits, merged PR descriptions, or historical issues as current development authority when they conflict with that file.

```bash
go run ./cmd/veilium
```

Desktop development:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails dev
```

Checks:

```bash
python scripts/check_project_governance.py
make check
```

## Safety and intended use

Veilium is intended for privacy, testing, QA, account separation, and authorized automation. It must not be used for fraud, unauthorized access, evasion of law enforcement, or bypassing platform rules.
