# Roadmap

## Completed foundation

- [x] Clean-room architecture and reference analysis.
- [x] Version-aware fingerprint provider capabilities.
- [x] Profile consistency validation and proxy route planning.
- [x] Atomic local profile persistence and secure local REST API.
- [x] Wails + React desktop profile manager.
- [x] Profile list, create/edit, groups, tags, search and clone.
- [x] Capability-driven fingerprint form.
- [x] Verified local kernel registry and managed imports.
- [x] SHA-256 re-verification and in-use deletion protection.

## Runtime supervisor

- [ ] Safe browser process lifecycle and crash recovery.
- [ ] Local CDP port allocation and readiness checks.
- [ ] Real-window/fingerprint dimension consistency.
- [ ] Extension and cookie management.
- [ ] Browser runtime integration tests on Windows, Linux and macOS.

## Proxy and credential platform

- [ ] Secret storage through the operating-system credential vault.
- [ ] Authenticated-proxy local bridge.
- [ ] Xray and sing-box adapters with checksums and pinned provenance.
- [ ] Health, latency, exit-IP, DNS and WebRTC leak checks.

## Automation and hardening

- [ ] Stable Launch API and unified CDP endpoint.
- [ ] MCP server with per-tool authorization.
- [ ] Export/import with schema migrations and optional encrypted sync.
- [ ] Signed releases, SBOM, provenance and reproducible builds.
