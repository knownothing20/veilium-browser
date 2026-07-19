# Veilium Roadmap

Last updated: 2026-07-20
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current phase status: Closing

## How to use this roadmap

This file controls phase order and phase status. It does not authorize an individual implementation task by itself. Detailed scope and exit criteria live in the current phase document, while the single current task lives in `docs/STATUS.md`.

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

## Phase 4 Closing Review

Issue #35 is the only authorized task. It reruns every Phase 4 exit gate against merged baseline `49ae2de6cb652d789c97aa961c0007513362bb6f`.

During `Closing`:

- product implementation is blocked;
- no Phase 5 planning implementation or product implementation is authorized;
- the review must record a Pass or Blocked decision;
- a Pass result requires a separate closure PR that marks Phase 4 `Done`, records unresolved risks and deferrals, updates `docs/STATUS.md`, and identifies the first Phase 5 planning task;
- a Blocked result requires one narrow corrective issue without expanding Phase 4 scope.

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

## Phase 4 exit summary

The final review must confirm:

- Provider Contract v2 and legacy compatibility are frozen;
- at least one exact reviewed Provider path has real-browser Evidence;
- custom Providers cannot inherit reviewed claims;
- capability states are Provider- and Evidence-derived;
- window, screen, viewport, and DPR consistency passes the approved matrix;
- browser-observed route and controlled Network Evidence exists for the exact reviewed binary;
- unsafe, unsupported, modified, missing, stale, contradictory, and incompatible states fail closed or remain explicitly limited;
- a reviewed exact-combination compatibility contract exists;
- privacy, licensing, rollback, platform, governance, and technical validation gates pass;
- unresolved risks and deferred work are recorded.

The detailed closure requirements live in `docs/PHASE_04.md` and the pending review record in `docs/PHASE_04_CLOSING_REVIEW.md`.

## Phase status rules

Allowed phase states are:

- `Planning` — scope is being decided; product implementation is blocked;
- `Active` — milestones and exit criteria are approved;
- `Closing` — implementation is complete and closure evidence is being reviewed;
- `Done` — exit criteria passed and the phase is frozen;
- `Planned` — a later phase placeholder without implementation authority.

The current phase document and `docs/STATUS.md` must agree with the metadata at the top of this file. CI validates that relationship.