# Reference project analysis

This document records architectural lessons, not copied implementation details.

## Donut Browser

Upstream: <https://github.com/zhom/donutbrowser>

Strong ideas:

- Product-level profile isolation, extensions, proxies, VPN, sync, and MCP.
- Careful consistency between proxy exit, timezone, locale, geolocation, and real window dimensions.
- Token-protected local automation endpoints and encrypted-profile concepts.

Risks to avoid:

- The browser-management application depends on the external Wayfern binary for core fingerprint behavior.
- Some automation and cross-OS behavior is coupled to cloud capability checks.
- AGPL obligations require a clean-room approach for a differently licensed project.

## Ant Browser

Upstream: <https://github.com/black-ant/Ant-Browser>

Strong ideas:

- Go/Wails architecture is straightforward to extend.
- Kernel management, profile migration, rich proxy bridging, local Launch API, unified CDP, plugins, and automation scripts.
- Local-first data management.

Risks to avoid:

- UI parameters can drift from the command-line contract of the selected Chromium version.
- The main repository previously lacked an explicit license, so Veilium does not copy its source.
- Optional API authentication would become dangerous if listening beyond loopback.

## VirtualBrowser

Upstream: <https://github.com/Virtual-Browser/VirtualBrowser>

Strong ideas:

- Clear taxonomy for profile and fingerprint configuration.
- Simple Playwright-over-CDP automation concept.
- Familiar multi-profile management UI.

Risks to avoid:

- Public source does not make the complete fingerprint engine and runtime behavior equally auditable.
- A large number of manually editable fields can create internally inconsistent identities.
- UI claims must never be treated as proof that a browser binary applies a setting.

## Veilium synthesis

Veilium combines the useful ideas through independent implementation:

1. Ant-style local extensibility and provider separation.
2. Donut-style identity consistency and secure local automation.
3. VirtualBrowser-style understandable configuration, restricted by real provider capabilities.
4. A kernel capability matrix keyed by provider and Chromium major version.
5. Clean separation between profile data, launch planning, browser runtime, desktop UI, and automation API.
