# Veilium Browser

Veilium is an open-source, multi-profile privacy browser workspace focused on isolated identities, explicit kernel capabilities and reviewable network routing.

> Clean-room project: architecture and product lessons are studied from existing open-source browsers, while Veilium implementation code is written independently.

## Current capabilities

- version-aware Chromium provider contracts and fingerprint consistency validation;
- atomic local profile persistence and loopback-only authenticated REST API;
- Wails + React desktop profile workspace;
- verified local kernel registry with managed copies and SHA-256 integrity records;
- operating-system credential vault with no plaintext password fallback;
- per-profile loopback authentication bridges for HTTP, HTTPS, and SOCKS5 upstream proxies;
- one-click proxy connectivity, latency, exit-IP, DNS-route, and WebRTC-policy diagnostics;
- managed Xray and sing-box adapter registry with SHA-256, provenance, license, and profile-reference controls;
- supervised local browser start, stop and runtime-session monitoring;
- Chromium-assigned CDP ports discovered through `DevToolsActivePort`, removing the preselected-port race;
- loopback-only `/json/version` and debugger WebSocket validation;
- Unix process-group ownership and Windows Job Object child-tree cleanup;
- private per-start runtime logs and application-shutdown cleanup;
- Linux and Windows desktop build and runtime tests.

Actual browser execution requires a registered, integrity-verified kernel and a profile using its Veilium-managed user-data directory. Xray and sing-box binaries can be registered and bound to profiles, but advanced protocol configuration providers remain fail-closed until separately implemented and reviewed.

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

See [`docs/PROXY_ADAPTER_RUNTIME.md`](docs/PROXY_ADAPTER_RUNTIME.md), [`docs/PROXY_DIAGNOSTICS.md`](docs/PROXY_DIAGNOSTICS.md), [`docs/AUTHENTICATED_PROXY_BRIDGE.md`](docs/AUTHENTICATED_PROXY_BRIDGE.md), [`docs/CREDENTIAL_VAULT.md`](docs/CREDENTIAL_VAULT.md), [`docs/RUNTIME_SUPERVISOR.md`](docs/RUNTIME_SUPERVISOR.md), [`docs/KERNEL_REGISTRY.md`](docs/KERNEL_REGISTRY.md), and [`docs/ROADMAP.md`](docs/ROADMAP.md).

## Safety and intended use

Veilium is intended for privacy, testing, QA, account separation and authorized automation. It must not be used to bypass platform rules, commit fraud, evade law enforcement or access systems without authorization.
