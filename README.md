# Veilium Browser

Veilium is a local-first, multi-profile privacy browser workspace built with Go, Wails, React, and managed Chromium runtimes.

> Clean-room project: public projects may inform requirements and architecture, while Veilium implementation is developed independently.

## Current baseline

The consolidated `main` branch includes:

- isolated browser Profiles with managed user-data directories;
- reviewed/custom Chromium Provider contracts and exact package-integrity records;
- OS-backed credential storage with no plaintext password fallback;
- HTTP/HTTPS/SOCKS5 proxy bridges plus supervised Xray and sing-box subsets;
- browser runtime supervision, CDP readiness, process-tree cleanup, and private runtime logs;
- proxy diagnostics, browser identity Evidence, managed-window consistency, and Network Evidence;
- lifecycle journal, locks, cancellation, storage inventory, snapshots, restore, archive, recoverable trash, and rollback;
- portable non-secret Profile definitions, dependency remapping, templates, bounded multi-Profile operations, storage review, and operation-report export;
- a Simplified Chinese task-oriented desktop workspace for browser environments, network, recovery, batch management, and settings.

The current reviewed Chromium Provider remains an exact Windows amd64 stock Chromium Snapshot. Advanced fingerprint overrides are not claimed for that Provider.

## Current development direction

The next major work is to add a separate, evidence-backed Veilium Fingerprint Chromium Provider without weakening the existing stock Provider, security boundaries, or lifecycle model.

Current sources of truth:

- [`docs/PRODUCT.md`](docs/PRODUCT.md)
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- [`docs/ROADMAP.md`](docs/ROADMAP.md)
- [`docs/STATUS.md`](docs/STATUS.md)
- [`docs/PRISM_COMPARISON_OPTIMIZATION_PLAN.md`](docs/PRISM_COMPARISON_OPTIMIZATION_PLAN.md)
- [`docs/DEVELOPMENT_PROCESS.md`](docs/DEVELOPMENT_PROCESS.md)

Completed implementation and historical planning documents live under [`docs/archive/`](docs/archive/README.md) and are not active development authority.

## Development

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
