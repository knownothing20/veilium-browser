# Veilium Roadmap

Last updated: 2026-07-19
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current phase status: Closing

## How to use this roadmap

This file controls phase order and phase status. It does not authorize an individual implementation task by itself. Detailed scope and exit criteria live in the active phase document, while the single current task lives in `docs/STATUS.md`.

A later phase must not begin before the current phase closes through a dedicated phase-closure pull request. Changing phase order, user outcome, milestone sequence, or phase goals requires a planning issue and reviewed planning pull request.

## Six-phase plan

| Phase | Outcome | Status | Governing document |
| --- | --- | --- | --- |
| Phase 1 | Clean-room core contracts, local persistence, policy validation, and secure local API foundation | Done | Historical PR and module documents |
| Phase 2 | Wails and React desktop profile workspace with capability-driven configuration | Done | Historical PR and module documents |
| Phase 3 | Verified local kernels, supervised browser runtime, credential vault, proxy bridges, diagnostics, and reviewed Xray/sing-box adapters | Done | Historical PR and module documents |
| Phase 4 | Reviewed browser-provider contracts plus real-browser identity, consistency, and network evidence with truthful compatibility states | Closing | `docs/PHASE_04.md` |
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
- Linux and Windows Go, frontend, desktop-build, and adapter-validation CI;
- repository governance, required pull requests, status checks, phase gates, and SSOT documents.

## Delivered Phase 4 milestone sequence

Phase 4 was implemented in this dependency order:

1. **M4.1 — Kernel Provider Contract v2**
   - reviewed versus custom provider trust;
   - versioned provider, provenance, license, binary identity, and capability contracts;
   - compatibility for existing kernel and profile records;
   - unsupported and unverified states fail closed.
2. **M4.2 — Real-Browser Evidence Harness**
   - observe real browser behavior across controlled contexts;
   - create local, structured, redacted evidence reports;
   - derive capability states from exact provider evidence.
3. **M4.3 — Identity and Window Consistency**
   - enforce coherent OS, browser, language, timezone, CPU, screen, window, viewport, DPR, and supported GPU behavior;
   - derive understandable profile health from evidence.
4. **M4.4 — Live Browser Network Evidence and Compatibility Matrix**
   - verify browser-observed route, exit IP, WebRTC/STUN, and DNS behavior;
   - publish exact provider/version/OS/capability compatibility states without exceeding accepted evidence.

Implementation is complete, but Phase 4 remains `Closing` until Issue #28 reviews every exit gate. The critical question is whether at least one exact reviewed Provider path has applicable identity, consistency/window, and network Evidence. Controlled hosted-browser CI fixtures do not automatically satisfy that production reviewed-Provider gate.

The full scope, non-scope, platform policy, acceptance criteria, rollback rules, and exit gate are defined in `docs/PHASE_04.md`. Logical data and evidence contracts are defined in `docs/PHASE_04_CONTRACTS.md`. The active closing report is `docs/PHASE_04_CLOSING_REVIEW.md`.

## Phase 4 explicit deferrals

The following work is not authorized during Phase 4 closing review:

### Profile lifecycle and operations

- cookie import, export, or editing;
- extension package management;
- complete profile backup, restore, or cross-device migration;
- profile templates and broad batch operations;
- bounded crash-restart policy unless required by evidence-harness reliability.

### Proxy platform expansion

- broader share-link compatibility and ecosystem aliases;
- additional Xray, sing-box, or Mihomo protocols, transports, and options;
- proxy import, tagging, batch testing, rotation, or scheduled health operations;
- unrelated proxy UI expansion.

### Automation, sync, and release

- stable public Launch API and unified CDP gateway;
- MCP server and broad automation-script platform;
- cloud sync;
- application release signing, auto-update, SBOM, and full reproducible application builds except evidence needed for reviewed provider trust;
- broad UI redesign unrelated to provider, capability, evidence, or health states.

These are candidates for Phase 5 or Phase 6 planning and do not gain priority from this backlog entry.

## Phase 4 exit summary

Phase 4 cannot become `Done` until:

- Provider Contract v2 and legacy compatibility are frozen;
- at least one exact reviewed provider path has real-browser evidence;
- custom providers cannot inherit reviewed claims;
- capability states are provider- and evidence-derived;
- window, screen, viewport, and DPR consistency passes the approved matrix;
- browser-observed route, WebRTC, and DNS evidence exists for supported routes;
- unsafe, unsupported, modified, missing, and contradictory states fail closed;
- a reviewed compatibility matrix exists;
- the complete governance and technical validation matrix passes.

The detailed closure requirements live in `docs/PHASE_04.md`. Issue #28 must record every gate as Passed, Blocked, or Not applicable with an explanation.

## Phase status rules

Allowed phase states are:

- `Planning` — scope is being decided; product implementation is blocked;
- `Active` — milestones and exit criteria are approved;
- `Closing` — implementation is complete and closure evidence is being reviewed;
- `Done` — exit criteria passed and the phase is frozen;
- `Planned` — a later phase placeholder without implementation authority.

The active phase document and `docs/STATUS.md` must agree with the metadata at the top of this file. CI validates that relationship.