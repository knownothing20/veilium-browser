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

- [x] Supervised Xray provider for constrained VMess, VLESS, Trojan, and Shadowsocks profiles.
- [x] Supervised sing-box provider for constrained Hysteria2, TUIC, and AnyTLS profiles.
- [x] Private per-session Xray and sing-box configurations with vault material resolved only at runtime.
- [x] Loopback SOCKS5 readiness checks and Xray/sing-box process-tree supervision.
- [x] Constrained Xray URL schema with explicit unsupported-field reporting.
- [ ] Broader share-link import, including legacy VMess payloads and ecosystem-specific aliases.
- [ ] Additional reviewed sing-box options, transports, multiplexing, and endpoint formats.
- [ ] XHTTP, HTTPUpgrade, mKCP, Hysteria transport, and additional reviewed Xray options.
- [x] Pinned official Xray and sing-box release manifests with archive and executable SHA-256.
- [x] Native Xray/sing-box configuration checks on Linux and Windows.
- [x] Local Chromium-through-adapter smoke test without a public proxy endpoint.
- [ ] Publisher signatures, transparency evidence, notices, and disabled-by-default optional installers.

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
