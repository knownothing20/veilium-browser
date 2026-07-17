# Roadmap

## Phase 1 — Core contracts and CI

- [x] Clean-room architecture and reference analysis.
- [x] Version-aware fingerprint provider capabilities.
- [x] Profile consistency validation.
- [x] Proxy route and bridge selection.
- [x] Atomic local profile persistence.
- [x] Secure local REST API.
- [x] Linux and Windows CI.

## Phase 2 — Desktop profile manager

- [ ] Wails + React desktop shell.
- [ ] Profile list, create/edit, groups, tags, search, clone, and migration.
- [ ] Kernel registry and verified local imports.
- [ ] Capability-driven fingerprint form.
- [ ] Secret storage through the operating-system credential vault.

## Phase 3 — Runtime supervisor

- [ ] Safe browser process lifecycle and crash recovery.
- [ ] Local CDP port allocation and readiness checks.
- [ ] Real-window/fingerprint dimension consistency.
- [ ] Extension and cookie management.
- [ ] Browser runtime integration tests on Windows, Linux, and macOS.

## Phase 4 — Proxy platform

- [ ] Native HTTP/HTTPS/SOCKS routing.
- [ ] Authenticated-proxy local bridge.
- [ ] Xray and sing-box adapters with checksums and pinned provenance.
- [ ] Health, latency, exit-IP, DNS, and WebRTC leak checks.
- [ ] Optional Mihomo adapter after footprint evaluation.

## Phase 5 — Automation

- [ ] Stable Launch API and unified CDP endpoint.
- [ ] MCP server with per-tool authorization.
- [ ] Playwright examples and automation script packages.
- [ ] Rate limits, audit log, and short-lived session tokens.

## Phase 6 — Sync and hardening

- [ ] Export/import with schema migrations.
- [ ] Optional self-hosted sync.
- [ ] End-to-end encrypted profile metadata.
- [ ] Signed releases, SBOM, provenance, and reproducible-build work.
