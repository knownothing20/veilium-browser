# Veilium Browser

Veilium is an open-source, multi-profile privacy browser workspace focused on isolated identities, explicit kernel capabilities and reviewable network routing.

> Clean-room project: architecture and product lessons are studied from existing open-source browsers, while Veilium implementation code is written independently.

## Current capabilities

- version-aware Chromium Provider contracts and fingerprint consistency validation;
- one exact reviewed official Chromium Snapshot Provider for Windows amd64, installed only after explicit license acknowledgement;
- immutable archive, executable, and complete 261-file Package Tree verification with no moving-latest resolution or silent update;
- atomic local profile persistence and loopback-only authenticated REST API;
- Wails + React desktop profile workspace;
- verified local kernel registry with managed custom executables and complete-package records;
- operating-system credential vault with no plaintext password fallback;
- per-profile loopback authentication bridges for HTTP, HTTPS, and SOCKS5 upstream proxies;
- one-click proxy connectivity, latency, exit-IP, DNS-route, and WebRTC-policy diagnostics;
- managed Xray and sing-box adapter registry with SHA-256, provenance, license, and profile-reference controls;
- embedded official Xray/sing-box release pins with archive and executable digests plus native configuration checks;
- explicit, disabled-by-default desktop installation of exact pinned adapter assets with license acknowledgement and full archive/executable verification;
- supervised Xray execution for constrained VLESS, VMess, Trojan, and Shadowsocks profiles through a private loopback SOCKS5 endpoint;
- supervised sing-box execution for constrained Hysteria2, TUIC, and AnyTLS profiles through a private loopback SOCKS5 endpoint;
- supervised local browser start, stop and runtime-session monitoring;
- Chromium-assigned CDP ports discovered through `DevToolsActivePort`, removing the preselected-port race;
- loopback-only `/json/version` and debugger WebSocket validation;
- real-browser identity, managed-window/consistency, and Network Evidence with exact-combination compatibility records;
- Unix process-group ownership and Windows Job Object child-tree cleanup;
- private per-start runtime logs and application-shutdown cleanup;
- Linux and Windows desktop build and runtime tests.

Actual browser execution requires a registered, integrity-verified kernel and a profile using its Veilium-managed user-data directory. The reviewed Chromium claim applies only to the exact Windows amd64 Snapshot package documented in [`docs/OFFICIAL_CHROMIUM_PROVIDER.md`](docs/OFFICIAL_CHROMIUM_PROVIDER.md). Custom and legacy kernels remain outside that trust boundary, and unsupported stock Chromium fingerprint controls remain fail-closed.

The reviewed Xray and sing-box subsets run through private per-session configurations and supervised loopback SOCKS5 endpoints. Unsupported options remain fail-closed.

## Project direction and current work

Veilium uses a small set of sources of truth so that different developers and AI sessions follow the same plan:

1. [`docs/PRODUCT.md`](docs/PRODUCT.md) — product purpose, principles, and non-goals;
2. [`docs/ROADMAP.md`](docs/ROADMAP.md) — six-phase sequence and phase status;
3. [`docs/STATUS.md`](docs/STATUS.md) — the current milestone, task, blockers, and handoff;
4. the active phase document named by `docs/STATUS.md`;
5. [`docs/DEVELOPMENT_PROCESS.md`](docs/DEVELOPMENT_PROCESS.md) — issue, PR, validation, activation, and closure rules.

Contributors and development agents must also follow [`AGENTS.md`](AGENTS.md). Product-code changes are blocked by governance CI while the active phase remains in planning, and product-code PRs must update `docs/STATUS.md`.

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
python scripts/check_project_governance.py
make check
```

Architecture and implementation references:

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md)
- [`docs/OFFICIAL_CHROMIUM_PROVIDER.md`](docs/OFFICIAL_CHROMIUM_PROVIDER.md)
- [`docs/KERNEL_REGISTRY.md`](docs/KERNEL_REGISTRY.md)
- [`docs/COMPATIBILITY_MATRIX.md`](docs/COMPATIBILITY_MATRIX.md)
- [`docs/OFFICIAL_ADAPTER_INSTALLER.md`](docs/OFFICIAL_ADAPTER_INSTALLER.md)
- [`docs/OFFICIAL_ADAPTER_VALIDATION.md`](docs/OFFICIAL_ADAPTER_VALIDATION.md)
- [`docs/SING_BOX_PROVIDER.md`](docs/SING_BOX_PROVIDER.md)
- [`docs/XRAY_PROVIDER.md`](docs/XRAY_PROVIDER.md)
- [`docs/PROXY_ADAPTER_RUNTIME.md`](docs/PROXY_ADAPTER_RUNTIME.md)
- [`docs/PROXY_DIAGNOSTICS.md`](docs/PROXY_DIAGNOSTICS.md)
- [`docs/AUTHENTICATED_PROXY_BRIDGE.md`](docs/AUTHENTICATED_PROXY_BRIDGE.md)
- [`docs/CREDENTIAL_VAULT.md`](docs/CREDENTIAL_VAULT.md)
- [`docs/RUNTIME_SUPERVISOR.md`](docs/RUNTIME_SUPERVISOR.md)

## Safety and intended use

Veilium is intended for privacy, testing, QA, account separation and authorized automation. It must not be used to bypass platform rules, commit fraud, evade law enforcement or access systems without authorization.
