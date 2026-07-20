# Veilium Roadmap

Last updated: 2026-07-20
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current phase status: Active

## How to use this roadmap

This file controls phase order and phase status. It does not authorize an arbitrary implementation task by itself. Detailed scope and exit criteria live in the current phase document, while the single current task lives in `docs/STATUS.md`.

An `Active` phase permits product implementation only for the issue explicitly named in STATUS. `Planning`, `Closing`, `Done`, and `Planned` block ordinary product implementation.

## Six-phase plan

| Phase | Outcome | Status | Governing document |
| --- | --- | --- | --- |
| Phase 1 | Clean-room core contracts, local persistence, policy validation, and secure local API foundation | Done | Historical PR and module documents |
| Phase 2 | Wails and React desktop Profile workspace with capability-driven configuration | Done | Historical PR and module documents |
| Phase 3 | Verified local kernels, supervised browser runtime, credential vault, proxy bridges, diagnostics, and reviewed Xray/sing-box adapters | Done | Historical PR and module documents |
| Phase 4 | Reviewed browser-provider contracts plus real-browser identity, consistency, and Network Evidence with truthful compatibility states | Done | `docs/PHASE_04.md` |
| Phase 5 | Recoverable Profile lifecycle, truthful local snapshots and portable definitions, templates, and bounded day-to-day operations | Active | `docs/PHASE_05.md` |
| Phase 6 | Controlled automation, migration/sync options, and production release hardening; final scope depends on prior phases | Planned | To be created during Phase 6 planning |

## Delivered baseline through Phase 4

The frozen baseline includes:

- atomic Profile metadata persistence and Veilium-managed browser directories;
- searchable desktop Profile workspace with create, edit, clone, groups, tags, and dry-run launch plans;
- operating-system credential vault without plaintext fallback;
- verified Kernel and adapter registries with integrity, provenance, license, and in-use controls;
- supervised browser, authenticated proxy bridge, Xray, and sing-box runtimes;
- Provider Contract v2 with reviewed, custom, legacy, disabled, and invalid trust states;
- real-browser identity, managed-window/consistency, Network Evidence, Profile health, and exact compatibility contracts;
- one exact reviewed official Chromium Snapshot Provider for Windows amd64;
- archive, executable, and complete 261-file Package Tree verification;
- dependency-tamper failure, exact-combination compatibility, and protected Windows/Linux CI;
- Windows reviewed-Chromium CI user-local staging that preserves Sandbox, binary identity, and Evidence assertions;
- repository governance, phase gates, and source-of-truth documents.

Phase 4 is frozen as `Done`. Its support claims and contracts cannot be broadened by Phase 5 lifecycle, import, restore, or migration metadata.

## Phase 5 activation

The project owner approved Phase 5 through Activation Review Issue #40 on 2026-07-20.

The approved planning packet is:

- Issue #37;
- PR #39, merged as `531f56d49cebc79cf6aee7a24d8f972d6275ce6b`;
- `docs/PHASE_05.md`;
- `docs/PHASE_05_CONTRACTS.md`;
- CI Hotfix #43, merged as `6d4b04a9668c87cc110a4c0d423909d45649b529`;
- the documentation-only activation PR.

Exactly one implementation issue is authorized at activation:

- Issue #45 — **M5.1 Lifecycle Contract, Inventory, and Operation Journal**.

## Phase 5 outcome

A user can preserve, recover, move, template, archive, and manage Veilium Profiles without silently copying secrets, weakening Provider trust, reusing identities unintentionally, or losing browser data during interrupted operations.

## Phase 5 milestone sequence

1. **M5.1 — Lifecycle Contract, Inventory, and Operation Journal**
   - versioned lifecycle states separate from runtime and health;
   - durable operation records, per-item results, locks, cancellation, inventory, and startup reconciliation;
   - conservative compatibility for existing Profiles;
   - current and only authorized implementation milestone: Issue #45.
2. **M5.2 — Safe Local Snapshot, Restore, Archive, and Trash**
   - same-machine full snapshots of Profile metadata and opaque managed browser data;
   - deterministic tree identity, bounds, staging, atomic activation, rollback, and disk-space checks;
   - archive, recoverable trash, retention, restore, and explicit permanent deletion;
   - blocked until M5.1 closes and STATUS advances.
3. **M5.3 — Portable Profile Definitions and Templates**
   - configuration-only packages without secrets, browser data, local paths, binaries, runtime state, or Evidence;
   - dependency requirements and explicit local remapping;
   - new-identity default, explicit preserve-identity mode, and stale Evidence handling;
   - templates that always create a new Profile ID, directory, and fingerprint seed;
   - blocked until M5.2 closes and STATUS advances.
4. **M5.4 — Bounded Multi-profile Operations and Storage Management**
   - bounded tags/groups, lifecycle actions, definition export, health/integrity refresh, stop operations, and storage inspection;
   - preflight, cancellation, bounded concurrency, per-item results, and non-destructive repair plans;
   - no bulk start, scheduling, proxy rotation, or general automation;
   - blocked until M5.3 closes and STATUS advances.

The full active scope and exit criteria are in `docs/PHASE_05.md`. Logical state, artifact, migration, recovery, and platform contracts remain authoritative in `docs/PHASE_05_CONTRACTS.md`.

## Phase 5 boundaries

### Data classes

Phase 5 keeps these classes separate:

- portable Profile metadata and validated non-secret configuration;
- local/machine-bound browser user data and operating-system vault secrets;
- managed Kernel and adapter binaries with independent identities;
- Provider-, binary-, platform-, route-, and Evidence-bound claims;
- operational logs, staging, quarantine, and runtime state.

A local full snapshot is not automatically a cross-machine migration package. A portable definition is not a copy of browser state.

### Platform policy

- Windows and Linux are the implementation and CI targets for claimed Phase 5 lifecycle behavior;
- full local snapshot/restore requires real filesystem integration tests on each claimed platform;
- the reviewed browser Provider remains Windows amd64 only;
- Linux lifecycle support does not create a reviewed Linux browser Provider;
- macOS and cross-platform full browser-state migration remain unclaimed.

### Explicit Phase 5 non-goals

- Cookie import, export, or editing as a separate product surface;
- extension package management;
- export or synchronization of operating-system vault secrets;
- bundling or updating Chromium, Xray, or sing-box inside Profile artifacts;
- a second reviewed browser Provider, revision, platform, or architecture;
- new fingerprint controls or optimistic compatibility claims;
- proxy expansion, pools, rotation, scheduled health actions, or account farming;
- bulk Profile start, general browser automation, public API, MCP, or cloud sync;
- release signing, auto-update, SBOM, or reproducible application release work;
- unrelated broad UI redesign.

## Milestone control

- implementation must follow the milestone order;
- STATUS identifies exactly one current implementation issue;
- product-code PRs must update STATUS in the same change set;
- each milestone requires a dedicated closing review before the next milestone starts;
- later milestone scope cannot enter an earlier milestone PR;
- Phase 4 Provider, capability, Evidence, privacy, compatibility, and fail-closed boundaries remain mandatory regressions.

## Phase 6 remains deferred

Phase 6 may later plan:

- controlled local automation and audited session control;
- public Launch API, bounded CDP gateway, or MCP;
- cloud or cross-device synchronization;
- release signing, auto-update, SBOM, and reproducible builds;
- other production-hardening work.

It gains no implementation authority from Phase 5 activation.

## Phase status rules

Allowed phase states are:

- `Planning` — scope is being decided; product implementation is blocked;
- `Active` — milestones and exit criteria are approved, but STATUS still limits work to one current issue;
- `Closing` — implementation is complete and closure Evidence is being reviewed;
- `Done` — exit criteria passed and the phase is frozen;
- `Planned` — a later phase placeholder without implementation authority.

The current phase document and `docs/STATUS.md` must agree with the metadata at the top of this file. Governance CI validates that relationship.
