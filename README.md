# Veilium Browser

**Open-source multi-profile privacy browser infrastructure.**

Veilium is a clean-room project inspired by the strongest architectural ideas in Donut Browser, Ant Browser, and VirtualBrowser. It focuses on isolated browser identities, version-aware fingerprint settings, proxy routing, and automation without coupling the application to one browser binary.

> Phase 1 is a tested core service and launch planner. The desktop UI and real browser-process runtime are intentionally developed in later reviewed phases.

## Why Veilium

- One profile = one isolated user-data directory, fingerprint policy, proxy route, and kernel reference.
- Browser-kernel providers expose versioned capability contracts, so the UI cannot silently send obsolete flags.
- Inline proxy credentials are rejected; profiles store only an operating-system vault `credentialRef`, and authenticated proxies use a planned local bridge.
- The local API binds to loopback by default and requires a strong bearer token.
- No source code is copied from the reference projects; implementation is clean-room.

## Current Phase 1 capabilities

- JSON profile persistence with atomic replacement.
- Native and patched Chromium provider contracts.
- Fingerprint consistency validation.
- Chromium launch-plan generation.
- HTTP/HTTPS/SOCKS5 and advanced proxy route classification.
- Local authenticated REST API.
- Linux and Windows CI checks.

## Run the local core

```bash
go run ./cmd/veilium
```

The service listens on `127.0.0.1:51090`. If `VEILIUM_API_TOKEN` is not set, an ephemeral token is generated and printed once.

```bash
VEILIUM_API_TOKEN='replace-with-at-least-24-characters' go run ./cmd/veilium
```

## Validate

```bash
gofmt -w .
go vet ./...
go test ./...
go build ./...
```

## Project status

See [Architecture](docs/ARCHITECTURE.md), [reference-project analysis](docs/REFERENCE_ANALYSIS.md), and [roadmap](docs/ROADMAP.md).

## License

Apache-2.0. Third-party browser kernels and proxy runtimes keep their own licenses and are not bundled until their redistribution terms have been reviewed.
