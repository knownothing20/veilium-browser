# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: b5af36ab02ee91f253427ca7c3ba6b768997c485
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.1 — Lifecycle Contract, Inventory, and Operation Journal
Current task: Owner review and merge decision for PR #52
Current branch: `agent/m5-1-lifecycle-foundation`

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`.

Phase 5 is `Active`. Product implementation remains authorized only for Issue #45. M5.2, M5.3, and M5.4 remain blocked until M5.1 is merged and passes a dedicated Closing Review.

`Product implementation allowed: Yes` is not broad authority. No work outside Issue #45 may be added to PR #52.

## M5.1 implementation stage

All six implementation and delivery stages are complete:

1. versioned lifecycle contract and atomic persistence — complete;
2. operation journal, conflict locks, dependent-operation blocking, and cancellation foundation — complete;
3. read-only managed-storage inventory and startup reconciliation — complete;
4. Desktop Service initialization, compatibility migration, lifecycle gating, and rollback integration — complete;
5. bounded Bootstrap/Wails surface and minimum existing-design UI — complete;
6. acceptance audit, implementation documentation, full protected CI, diff review, and PR handoff — complete.

No further M5.1 feature stage is authorized. The next action is review and merge of PR #52, followed by the dedicated M5.1 Closing Review. M5.2 must not begin automatically.

## Implemented M5.1 behavior

### Lifecycle and persistence

- independent versioned `lifecycle.json` records with `available`, `draft`, `archived`, `trashed`, and `invalid`;
- lifecycle state remains separate from runtime, health, compatibility, Provider trust, capability support, and Evidence;
- independent versioned `lifecycle-operations.json` journal with operation state, stage, timestamps, selected Profiles, per-item results, idempotency, cancellation state, limitations, and recovery actions;
- private, bounded, strict, atomic stores with optimistic revisions and rollback behavior;
- duplicate, malformed, oversized, future-version, unknown-field, symlink, non-regular-file, and unsafe-path input fails closed.

### Conflict, interruption, and recovery

- one conflicting lifecycle operation lock per Profile;
- selected Profile locks are acquired atomically;
- active browser sessions and protected dependent operations block before journal execution;
- idempotent duplicates return the existing operation;
- cancellation remains a durable request checked only at safe stages;
- interrupted, cancelled, partial, or failed work cannot become successful implicitly;
- application shutdown preserves durable non-terminal state;
- startup converts interrupted operations to `recovery-required`, creates per-item recovery results, and reconciles stale locks without guessing success.

### Storage inventory

- read-only expected-directory present/missing reporting;
- orphaned managed-directory reporting;
- unsafe root, symlink, Windows reparse/junction, special-file, non-directory, traversal, and uninspectable entry reporting;
- bounded regular-file counts and byte totals without opening browser content;
- cancelled or bounded scans remain explicitly incomplete;
- no automatic repair or deletion.

### Desktop Service and compatibility

- application startup securely creates a missing private data root and opens/reconciles lifecycle state;
- malformed or unsupported lifecycle state prevents Service initialization;
- existing safe managed Profiles receive explicit compatibility records;
- contradictory or unmanaged legacy Profiles remain readable but are lifecycle `invalid` and cannot launch;
- Profile creation synchronizes lifecycle metadata and rolls Profile metadata back if lifecycle persistence fails;
- cloning creates a new identity and lifecycle record and requires an available, unlocked source;
- launch and launch-plan creation require lifecycle `available` and unlocked in addition to all frozen Phase 4 checks;
- editing is blocked while locked, archived, or trashed; invalid/draft metadata remains inspectable for repair;
- direct Profile deletion fails closed until the M5.2 trash transaction exists.

### Desktop/UI boundary

- Bootstrap exposes lifecycle records, operation records, startup reconciliation, inventory findings, cancellation-request state, and safe-cancellation stage;
- Profile rows show lifecycle separately from runtime and health;
- lifecycle limitation, recovery, and lock reasons are visible in bounded form;
- controls are disabled according to lifecycle policy;
- dashboard shows lifecycle, operation, cancellation-availability, inventory, orphan, unsafe, and recovery state;
- no archive, trash, restore, permanent-delete, cancellation, batch, import/export, template, filesystem-browser, or automation action is exposed;
- browser preview creates no fake lifecycle records or support claims.

## Validation completed

The complete protected matrix passed on the completed implementation and documentation series:

- Governance scope checks;
- Go formatting, vet, unit/race coverage, and builds;
- lifecycle schema, persistence, rollback, duplicate/conflict, active-session, dependent-operation, cancellation, shutdown, and startup-reconciliation tests;
- Linux and Windows filesystem safety tests, including symlink and reparse handling;
- frontend typecheck, lifecycle policy tests, existing tests, and production build;
- Windows and Linux Wails builds;
- official adapter checks on Windows and Linux;
- Linux real-browser Evidence and both official adapter browser paths;
- exact Windows reviewed-Chromium installation, identity/window Evidence, Network Evidence, dependency-tamper fail-closed, build, artifact, and safe cleanup.

The final diff contains only Issue #45 lifecycle, Desktop integration, minimum UI, tests, and required documentation. It changes no workflow, Provider, Kernel, adapter protocol, fingerprint contract, Evidence rule, or M5.2–M5.4 data operation.

PR #52 has no review comments, review submissions, or unresolved inline threads at handoff.

### Independent CI reliability observation

During development, one Windows Network Evidence attempt reproduced open Issue #49: Chromium Sandbox reported `Access is denied (0x5)` for the fixed user-local executable. A later complete protected run passed the exact identity/window, Network Evidence, tamper, build, artifact, and cleanup chain without lifecycle or workflow changes.

A Linux collector shutdown timeout also occurred once and passed on retry without code changes. These remain recorded CI reliability observations; no Sandbox or Evidence requirement was weakened.

## Security, privacy, and compatibility boundaries

- operating-system vault secrets remain non-portable and are never copied into lifecycle state;
- no Cookies, LocalStorage, IndexedDB, history, tokens, page data, extension contents, or Evidence payloads are inspected;
- lifecycle reports expose bounded relative managed identities, not arbitrary filesystem browsing;
- no remote binding, public API, MCP, cloud sync, telemetry, automatic download, or workflow write permission is added;
- Phase 4 Provider trust, exact binary identity, Sandbox, compatibility, and Evidence requirements remain frozen;
- lifecycle metadata cannot create reviewed trust or Evidence applicability;
- macOS lifecycle support remains unclaimed.

## Explicit non-scope retained

PR #52 does not implement:

- snapshot container creation or snapshot/restore execution;
- archive, trash, restore-trash, retention, or permanent-delete data movement;
- portable Profile export/import;
- templates;
- Cookie or extension management;
- secret export;
- multi-Profile batch UI or operations;
- bulk start, scheduling, proxy rotation, account farming, general automation, public Launch API, MCP, sync, or release work;
- Provider, Kernel, adapter, fingerprint, proxy-protocol, compatibility, or Evidence claim expansion.

## Exact next task

1. Confirm the docs-only handoff commit retains green Governance and protected CI.
2. Mark PR #52 ready for review.
3. Obtain the owner merge decision and use squash merge if approved.
4. After merge, open the dedicated M5.1 Closing Review.
5. Advance STATUS to M5.2 only if that Closing Review passes.
