# Veilium Roadmap

Last updated: 2026-07-23
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current phase status: Active

## How to use this roadmap

This file controls phase order and phase status. Detailed scope and exit criteria live in the active phase document, while the exact current task lives in `docs/STATUS.md`.

An `Active` phase permits product implementation only within the authority explicitly recorded by `docs/PHASE_05.md` and `docs/STATUS.md`. Lower-level documents and pull-request descriptions may add detail but may not weaken product, security, privacy, licensing, Provider, lifecycle, or Evidence boundaries.

## Six-phase plan

| Phase | Outcome | Status | Governing document |
| --- | --- | --- | --- |
| Phase 1 | Clean-room core contracts, local persistence, policy validation, and secure local API foundation | Done | Historical PR and module documents |
| Phase 2 | Wails and React desktop Profile workspace with capability-driven configuration | Done | Historical PR and module documents |
| Phase 3 | Verified local kernels, supervised browser runtime, credential vault, proxy bridges, diagnostics, and reviewed Xray/sing-box adapters | Done | Historical PR and module documents |
| Phase 4 | Reviewed browser-provider contracts plus real-browser identity, consistency, and Network Evidence with truthful compatibility states | Done | `docs/PHASE_04.md` |
| Phase 5 | Recoverable Profile lifecycle, truthful local snapshots and portable definitions, templates, bounded day-to-day operations, and a usable Chinese desktop browser workspace | Active | `docs/PHASE_05.md` |
| Phase 6 | Controlled automation, migration/sync options, and production release hardening; final scope depends on Phase 5 closure | Planned | To be created during Phase 6 planning |

## Frozen baseline through Phase 4

The frozen baseline includes:

- atomic Profile metadata persistence and Veilium-managed browser directories;
- searchable desktop Profile workspace with create, edit, clone, groups, tags, and dry-run launch plans;
- operating-system credential vault without plaintext fallback;
- verified Kernel and adapter registries with integrity, provenance, license, and in-use controls;
- supervised browser, authenticated proxy bridge, Xray, and sing-box runtimes;
- Provider Contract v2 with reviewed, custom, legacy, disabled, and invalid trust states;
- real-browser identity, managed-window/consistency, Network Evidence, Profile health, and exact compatibility contracts;
- one exact reviewed official Chromium Snapshot Provider for Windows amd64;
- archive, executable, and complete Package Tree verification;
- dependency-tamper failure, exact-combination compatibility, and protected Windows/Linux evidence.

Phase 4 remains frozen. Phase 5 UI and lifecycle work cannot broaden Provider support, fingerprint capability claims, compatibility, health, or Evidence applicability.

## Phase 5 outcome

A user can create, start, inspect, preserve, recover, move, template, archive, and manage Veilium browser environments without silently copying secrets, weakening Provider trust, reusing identities unintentionally, or losing browser data during interrupted operations.

The desktop product must also present those capabilities through a task-oriented Simplified Chinese workspace in which the primary object is the browser environment and the primary action is opening the browser. Technical state remains truthful and reviewable through progressive disclosure.

## Phase 5 milestone sequence and record

1. **M5.1 — Lifecycle Contract, Inventory, and Operation Journal — Done**
   - versioned lifecycle states separate from runtime and health;
   - durable operation records, per-item results, locks, cancellation, inventory, and startup reconciliation;
   - implementation PR #52 and Closing Review #53 passed.
2. **M5.2 — Safe Local Snapshot, Restore, Archive, and Trash — Done**
   - same-machine verified snapshots, restore to a new identity, archive, recoverable trash, explicit permanent cleanup, rollback, and recovery state;
   - implementation PR #56 merged and Closing Review #57 passed.
3. **M5.3 — Portable Profile Definitions and Templates — In PR #59**
   - configuration-only packages without secrets, browser data, local paths, binaries, runtime state, or Evidence;
   - dependency remapping, import preview, new-identity default, advanced preserve-identity mode, and private templates.
4. **M5.4 — Bounded Multi-profile Operations and Storage Management — In PR #59**
   - bounded metadata, recoverable lifecycle, definition export, health refresh, storage inspection, operation reports, and non-destructive repair plans;
   - no bulk start, scheduling, proxy rotation, or general automation.
5. **M5.5 — Chinese Browser Workspace Productization — Authorized in PR #59**
   - governed by `docs/CHINESE_BROWSER_WORKSPACE_DEVELOPMENT_PLAN.md`;
   - typed `zh-CN` localization with an English fallback dictionary;
   - browser environments become the default workspace;
   - task-oriented primary navigation for environments, network, recovery, batch management, and settings;
   - Multi-Profile tools move from a floating dock into the normal page hierarchy;
   - create/edit and common environment actions become understandable Chinese flows;
   - technical Provider, Kernel, runtime, lifecycle, compatibility, and Evidence states remain accessible without dominating the default experience;
   - existing persisted contracts and backend authority remain unchanged unless separately documented and tested.

M5.3, M5.4, and M5.5 use PR #59 and branch `agent/handoff-m5-3` as the single remaining Phase 5 development path, as recorded by `docs/STATUS.md`.

## Phase 5 boundaries

### Data classes remain separate

- portable Profile metadata and validated non-secret configuration;
- local/machine-bound browser user data and operating-system vault secrets;
- managed Kernel and adapter binaries with independent identities;
- Provider-, binary-, platform-, route-, compatibility-, health-, and Evidence-bound claims;
- operational logs, staging, quarantine, lifecycle, and runtime state;
- application-language preferences, which must never mutate Profile identity settings.

A local snapshot is not a portable migration package. A portable definition is not a copy of browser state. A Chinese management interface does not imply a Chinese browser language, timezone, Accept-Language, or fingerprint configuration for every Profile.

### Platform policy

- Windows and Linux remain the implementation and validation targets for claimed Phase 5 behavior;
- reviewed browser Provider trust remains Windows amd64 only;
- Linux support does not create a reviewed Linux browser Provider;
- macOS and cross-platform full browser-state migration remain unclaimed.

### Explicit Phase 5 non-goals

- Cookie import, export, or editing as a separate product surface;
- extension package management;
- export or synchronization of operating-system vault secrets;
- bundling or silently updating Chromium, Xray, or sing-box;
- a second reviewed browser Provider, revision, platform, or architecture;
- new fingerprint controls or optimistic compatibility claims;
- proxy pools, rotation, scheduled health actions, or account farming;
- bulk Profile start, general browser automation, public API, MCP, or cloud sync;
- release signing, auto-update, SBOM, or reproducible application release work;
- embedding a Chrome-style web renderer, address bar, tab strip, extension bar, or download manager inside the Wails management window;
- UI work unrelated to the approved Chinese workspace plan.

## Validation and closure

Before Phase 5 may close, PR #59 must receive actual executable validation, not static review alone:

- Go formatting, vet, unit/race tests, and builds;
- frontend typecheck, tests, and production build;
- Wails development startup and Windows amd64 build;
- real Chromium start, readiness, stop, and cleanup;
- proxy, lifecycle, recovery, portability, template, batch, storage, and report smoke tests;
- Simplified Chinese primary-flow review at 1366×768 and 1920×1080;
- confirmation that management-language changes do not alter Profile identity settings;
- final documentation and changed-file review.

GitHub Actions remain paused under the current owner instruction. No success may be claimed until the applicable checks are actually run in a suitable environment.

## Phase 6 remains deferred

Phase 6 may later plan controlled local automation, audited session control, public Launch API or bounded CDP/MCP surfaces, cloud or cross-device synchronization, and production release hardening. It gains no implementation authority from Phase 5.
