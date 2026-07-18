# Veilium Roadmap

Last updated: 2026-07-18
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current phase status: Planning

## How to use this roadmap

This file controls phase order and phase status. It does not authorize an individual implementation task by itself. Detailed scope and exit criteria live in the active phase document, while the single current task lives in `docs/STATUS.md`.

A later phase must not begin before the current phase closes through a dedicated phase-closure pull request. Changing phase order or goals requires a planning issue and reviewed roadmap update.

## Six-phase plan

| Phase | Outcome | Status | Governing document |
| --- | --- | --- | --- |
| Phase 1 | Clean-room core contracts, local persistence, policy validation, and secure local API foundation | Done | Historical PR and module documents |
| Phase 2 | Wails and React desktop profile workspace with capability-driven configuration | Done | Historical PR and module documents |
| Phase 3 | Verified local kernels, supervised browser runtime, credential vault, proxy bridges, diagnostics, and reviewed Xray/sing-box adapters | Done | Historical PR and module documents |
| Phase 4 | Next product capability and evidence phase; detailed user outcome and milestone order must be frozen before implementation | Planning | `docs/PHASE_04.md` |
| Phase 5 | Profile lifecycle and scalable day-to-day operations; final scope depends on Phase 4 closure | Planned | To be created during Phase 5 planning |
| Phase 6 | Controlled automation, migration/sync options, and production release hardening; final scope depends on prior phases | Planned | To be created during Phase 6 planning |

## Delivered baseline through Phase 3

The current baseline includes:

- version-aware fingerprint provider capabilities and consistency validation;
- atomic local profile persistence and authenticated loopback REST service;
- desktop profile list, create/edit, groups, tags, search, and clone;
- verified local Chromium kernel imports with SHA-256 re-verification;
- supervised browser lifecycle, Chromium-assigned CDP discovery, readiness checks, logs, and process-tree cleanup;
- operating-system credential storage without plaintext fallback;
- authenticated HTTP, HTTPS, and SOCKS5 loopback bridges;
- proxy connectivity, latency, exit-IP, DNS-route, and WebRTC-policy diagnostics;
- managed Xray and sing-box binaries with provenance, license, integrity, and in-use controls;
- supervised reviewed protocol subsets through private per-session configurations;
- pinned official adapter manifests, native configuration checks, Chromium smoke tests, and explicit verified installation;
- Linux and Windows Go, frontend, desktop-build, and adapter-validation CI.

## Unsequenced candidate backlog

The following items were present in the earlier module-oriented roadmap. They are retained as planning inputs, not approved Phase 4 tasks:

### Browser runtime and identity

- real-window and declared fingerprint dimension consistency;
- extension and cookie management;
- bounded crash-restart policy;
- approved real Chromium test-kernel matrix across supported platforms;
- live browser WebRTC/STUN and delegated-domain DNS leak tests;
- Windows suspended-process creation if evidence shows the remaining launch window is material.

### Proxy platform

- broader share-link compatibility and ecosystem aliases;
- additional reviewed Xray and sing-box transports and options;
- optional local-client authentication for ephemeral bridges;
- proxy import, tagging, batch testing, and rotation policy;
- configurable diagnostic endpoints;
- historical health reports and scheduled retesting;
- publisher signatures, transparency evidence, and reproducible runtime builds.

### Lifecycle, automation, and release

- stable Launch API and unified CDP endpoint;
- MCP server with per-tool authorization;
- export/import with schema migration and optional encrypted sync;
- signed releases, SBOM, provenance, updates, and reproducible application builds.

## Phase status rules

Allowed phase states are:

- `Planning` — scope is being decided; product implementation is blocked;
- `Active` — milestones and exit criteria are approved;
- `Closing` — implementation is complete and closure evidence is being reviewed;
- `Done` — exit criteria passed and the phase is frozen;
- `Planned` — a later phase placeholder without implementation authority.

The active phase document and `docs/STATUS.md` must agree with the metadata at the top of this file. CI validates that relationship.
