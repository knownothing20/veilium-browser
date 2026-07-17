# Veilium Browser

Veilium is an open-source, multi-profile privacy browser workspace focused on isolated identities, explicit kernel capabilities and reviewable network routing.

> Clean-room project: architecture and product lessons are studied from existing open-source browsers, while Veilium implementation code is written independently.

## Current capabilities

- version-aware Chromium provider contracts and fingerprint consistency validation;
- atomic local profile persistence and loopback-only authenticated REST API;
- Wails + React desktop profile workspace;
- verified local kernel registry with managed copies and SHA-256 integrity records;
- symlink and non-regular-file rejection during kernel import;
- profile references to registered kernels and in-use deletion protection;
- kernel re-verification before launch-plan generation;
- Linux and Windows desktop build CI.

Veilium currently creates and reviews launch plans but does not execute browser binaries. Process supervision remains a separate reviewed feature.

## Development

### Headless service

```bash
go run ./cmd/veilium
```

### Desktop application

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails dev
```

### Checks

```bash
make check
```

See [`docs/KERNEL_REGISTRY.md`](docs/KERNEL_REGISTRY.md), [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), and [`docs/ROADMAP.md`](docs/ROADMAP.md).

## Safety and intended use

Veilium is intended for privacy, testing, QA, account separation and authorized automation. It must not be used to bypass platform rules, commit fraud, evade law enforcement or access systems without authorization.
