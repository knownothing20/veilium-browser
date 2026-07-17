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
- [x] Safe browser process lifecycle and shutdown cleanup.
- [x] Chromium-assigned CDP port discovery and readiness checks.
- [x] Unix process groups and Windows Job Object tree cleanup.
- [x] Cross-platform supervised runtime handshake tests.
- [x] In-memory runtime status, logs, exit details and UI controls.
- [x] Operating-system credential vault with transactional metadata.
- [x] Metadata-only credential selection and in-use protection.
- [x] Authenticated HTTP, HTTPS, and SOCKS5 loopback proxy bridge.
- [x] Per-profile bridge health and lifecycle cleanup.

## Runtime follow-up

- [ ] Real-window/fingerprint dimension consistency.
- [ ] Extension and cookie management.
- [ ] Optional explicit crash-restart policy with bounded retries.
- [ ] Approved real Chromium test-kernel matrix on Windows, Linux and macOS.
- [ ] Native suspended-process creation on Windows if the remaining pre-assignment micro-window becomes material.

## Proxy platform follow-up

- [ ] Xray and sing-box adapters with checksums and pinned provenance.
- [ ] Health, latency, exit-IP, DNS and WebRTC leak checks.
- [ ] Optional local-client authentication for the ephemeral loopback bridge.
- [ ] Proxy import, tagging, batch testing and rotation policies.

## Automation and hardening

- [ ] Stable Launch API and unified CDP endpoint.
- [ ] MCP server with per-tool authorization.
- [ ] Export/import with schema migrations and optional encrypted sync.
- [ ] Signed releases, SBOM, provenance and reproducible builds.
