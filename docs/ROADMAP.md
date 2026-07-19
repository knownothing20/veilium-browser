# Veilium Roadmap

Last updated: 2026-07-20
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current phase status: Done

## How to use this roadmap

This file controls phase order and phase status. It does not authorize an individual implementation task by itself. Detailed scope and exit criteria live in the current phase document, while the single current task lives in `docs/STATUS.md`.

A later phase must not begin before the current phase closes through a dedicated phase-closure pull request. Changing phase order, user outcome, milestone sequence, or phase goals requires a planning issue and reviewed planning pull request.

## Six-phase plan

| Phase | Outcome | Status | Governing document |
| --- | --- | --- | --- |
| Phase 1 | Clean-room core contracts, local persistence, policy validation, and secure local API foundation | Done | Historical PR and module documents |
| Phase 2 | Wails and React desktop profile workspace with capability-driven configuration | Done | Historical PR and module documents |
| Phase 3 | Verified local kernels, supervised browser runtime, credential vault, proxy bridges, diagnostics, and reviewed Xray/sing-box adapters | Done | Historical PR and module documents |
| Phase 4 | Reviewed browser-provider contracts plus real-browser identity, consistency, and network evidence with truthful compatibility states | Done | `docs/PHASE_04.md` |
| Phase 5 | Profile lifecycle and scalable day-to-day operations; final scope must be defined by Issue #37 | Planned | To be created during Phase 5 planning |
| Phase 6 | Controlled automation, migration/sync options, and production release hardening; final scope depends on prior phases | Planned | To be created during Phase 6 planning |

## Delivered baseline through Phase 3

The baseline includes:

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
- Linux and Windows Go, frontend, desktop-build, and adapter-validation CI;
- repository governance, required pull requests, status checks, phase gates, and SSOT documents.

## Delivered Phase 4 sequence

Phase 4 was delivered in this dependency order:

1. **M4.1 — Kernel Provider Contract v2**
   - reviewed versus custom provider trust;
   - versioned provider, provenance, license, binary identity, and capability contracts;
   - compatibility for existing kernel and profile records;
   - unsupported and unverified states fail closed.
2. **M4.2 — Real-Browser Evidence Harness**
   - observe real browser behavior across controlled contexts;
   - create local, structured, redacted Evidence reports;
   - derive capability states from exact Provider Evidence.
3. **M4.3 — Identity and Window Consistency**
   - enforce coherent OS, browser, language, timezone, CPU, screen, window, viewport, DPR, and supported GPU behavior;
   - derive understandable Profile health from Evidence.
4. **M4.4 — Live Browser Network Evidence and Compatibility Matrix**
   - verify browser-observed route, Exit IP, WebRTC/STUN, and DNS behavior;
   - generate reviewed Provider/version/OS/capability compatibility states.
5. **Corrective — First Exact Reviewed Provider Path**
   - add one pinned official Windows amd64 Chromium Snapshot Provider;
   - freeze archive, executable, and complete Package Tree identity, source, license/provenance, limitations, and rollback rules;
   - run the same exact managed binary through M4.2, M4.3, and M4.4 Evidence;
   - restrict compatibility to the exact reviewed combination.

The full scope, non-scope, platform policy, acceptance criteria, rollback rules, and exit gates are defined in `docs/PHASE_04.md`. Logical contracts are defined in `docs/PHASE_04_CONTRACTS.md`. The reviewed Provider boundary is documented in `docs/OFFICIAL_CHROMIUM_PROVIDER.md` and `docs/PHASE_04_CORRECTIVE_PLAN.md`.

## Phase 4 closure result

Issue #35 recorded **Pass** after reviewing merged implementation baseline `49ae2de6cb652d789c97aa961c0007513362bb6f` and Closing-state baseline `759dd7ab6689c244e28ce9d09b63e9f2bac1878c`.

The final review confirmed:

- Provider Contract v2 and legacy compatibility are frozen;
- one exact reviewed Windows amd64 Chromium Snapshot Provider exists;
- custom and legacy Providers cannot inherit reviewed claims;
- archive, executable, and complete-package identities are immutable and fail closed;
- the same exact managed binary passes identity, managed-window/consistency, and controlled Network Evidence;
- capability and compatibility states remain Provider-, platform-, binary-, and Evidence-derived;
- unsafe, unsupported, modified, missing, stale, contradictory, and incompatible states remain explicit;
- privacy, licensing, rollback, platform, governance, and technical validation gates pass;
- unresolved risks and deferred work are recorded in `docs/PHASE_04_CLOSING_REVIEW.md`.

A first Closing-state Windows Evidence attempt hit a GitHub Runner temporary-directory Chromium Sandbox access denial after successful package installation. The identical Job passed on a fresh Runner without changing code, binary, assertions, or flags. This is retained as a non-blocking CI-environment reliability risk.

Phase 4 is now frozen as `Done`. Later changes to its contracts or support claims require a dedicated issue and reviewed planning change.

## Phase 4 explicit deferrals

The following work is not part of Phase 4 and gains no implementation authority from closure:

### Profile lifecycle and operations

- cookie import, export, or editing;
- extension package management;
- complete profile backup, restore, or cross-device migration;
- profile templates and broad batch operations;
- bounded crash-restart policy unless separately planned.

### Proxy platform expansion

- broader share-link compatibility and ecosystem aliases;
- additional Xray, sing-box, or Mihomo protocols, transports, and options;
- proxy import, tagging, batch testing, rotation, or scheduled health operations;
- unrelated proxy UI expansion.

### Automation, sync, and release

- stable public Launch API and unified CDP gateway;
- MCP server and broad automation-script platform;
- cloud sync;
- application release signing, auto-update, SBOM, and full reproducible application builds;
- broad UI redesign unrelated to Provider, capability, Evidence, or health states.

These are candidates for Phase 5 or Phase 6 planning only. They must be prioritized through the next phase planning process rather than inferred from this backlog.

## Next planning task

Issue #37, **define Phase 5 profile lifecycle and operations scope**, is the only next authorized planning task.

It must create a dedicated `docs/PHASE_05.md` in `Planning`, define one explicit user outcome, non-goals, dependency-ordered milestones, data and migration contracts, security/privacy/platform boundaries, validation, and exit gates, and update ROADMAP and STATUS in one reviewed planning pull request.

Issue #37 does not activate Phase 5. Phase 5 remains `Planned`, and all product implementation is blocked until the planning pull request is reviewed and a separate activation decision explicitly permits implementation.

## Phase status rules

Allowed phase states are:

- `Planning` — scope is being decided; product implementation is blocked;
- `Active` — milestones and exit criteria are approved;
- `Closing` — implementation is complete and closure evidence is being reviewed;
- `Done` — exit criteria passed and the phase is frozen;
- `Planned` — a later phase placeholder without implementation authority.

The current phase document and `docs/STATUS.md` must agree with the metadata at the top of this file. CI validates that relationship.