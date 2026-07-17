# Veilium Browser

Veilium is an open-source, multi-profile privacy browser workspace focused on isolated identities, explicit kernel capabilities and reviewable network routing.

> Clean-room project: architecture and product lessons are studied from existing open-source browsers, while Veilium implementation code is written independently.

## Current status

### Phase 1 — Core foundation

- version-aware Chromium provider contracts;
- fingerprint consistency validation;
- proxy route classification without inline credentials;
- deterministic launch-plan generation;
- atomic local profile persistence;
- loopback-only bearer-authenticated REST API;
- Linux and Windows CI.

### Phase 2 — Desktop shell

- Wails v2 desktop application;
- React + TypeScript + Vite interface;
- dashboard and searchable profile registry;
- create, edit, clone and delete profile workflows;
- group and tag organization;
- kernel capability registry;
- launch-plan dry-run drawer;
- browser-preview mode with temporary demo data;
- Windows and Linux desktop build CI.

Phase 2 intentionally creates and reviews launch plans but does not execute browser binaries yet.

## Development

### Headless service

```bash
go run ./cmd/veilium
```

The REST service listens on `127.0.0.1:51090` by default. Set `VEILIUM_API_TOKEN` to a strong local token or the command generates an ephemeral one.

### Desktop application

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails dev
```

### Checks

```bash
make check
```

See:

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- [`docs/REFERENCE_ANALYSIS.md`](docs/REFERENCE_ANALYSIS.md)
- [`docs/PHASE2_DESKTOP.md`](docs/PHASE2_DESKTOP.md)
- [`docs/ROADMAP.md`](docs/ROADMAP.md)

## Safety and intended use

Veilium is intended for privacy, testing, QA, account separation and authorized automation. It must not be used to bypass platform rules, commit fraud, evade law enforcement or access systems without authorization.
