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
- [x] On-demand proxy connectivity, latency, and exit-IP measurement.
- [x] DNS-route analysis and WebRTC-policy risk reporting.
- [x] Managed Xray and sing-box executable imports.
- [x] Adapter SHA-256, provenance, license, capability, and in-use controls.
- [x] Profile-to-adapter binding with runtime integrity re-verification.

## Runtime follow-up

- [ ] Real-window/fingerprint dimension consistency.
- [ ] Extension and cookie management.
- [ ] Optional explicit crash-restart policy with bounded retries.
- [ ] Approved real Chromium test-kernel matrix on Windows, Linux and macOS.
- [ ] Native suspended-process creation on Windows if the remaining pre-assignment micro-window becomes material.
- [ ] Live browser WebRTC/STUN and delegated-domain DNS leak tests.

## Proxy adapter providers

- [ ] Xray provider for VMess, VLESS, Trojan, Shadowsocks, and compatible transports.
- [ ] sing-box provider for Hysteria2, TUIC, AnyTLS, and compatible transports.
- [ ] Private per-session adapter configuration with vault material resolved only at runtime.
- [ ] Loopback inbound readiness checks and adapter process-tree supervision.
- [ ] Share-link parser schemas with explicit capability and unsupported-field reporting.
- [ ] Pinned optional download manifests with signatures, checksums, notices, and disabled-by-default policy.

## Proxy platform follow-up

- [ ] Optional local-client authentication for the ephemeral loopback bridge.
- [ ] Proxy import, tagging, batch testing and rotation policies.
- [ ] Configurable or self-hosted diagnostic probe endpoints.
- [ ] Historical health reports and scheduled retesting with explicit retention controls.

## Automation and hardening

- [ ] Stable Launch API and unified CDP endpoint.
- [ ] MCP server with per-tool authorization.
- [ ] Export/import with schema migrations and optional encrypted sync.
- [ ] Signed releases, SBOM, provenance and reproducible builds.
