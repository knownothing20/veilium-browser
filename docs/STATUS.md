# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: b5af36ab02ee91f253427ca7c3ba6b768997c485
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.1 — Lifecycle Contract, Inventory, and Operation Journal
Current task: Complete final validation and review of Issue #45 in Draft PR #52
Current branch: `agent/m5-1-lifecycle-foundation`

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`.

Phase 5 is `Active`. Product implementation remains authorized only for Issue #45. M5.2, M5.3, and M5.4 remain blocked until M5.1 is merged and passes a dedicated Closing Review.

`Product implementation allowed: Yes` is not broad authority. No work outside Issue #45 may be added to PR #52.

## Current implementation stage

M5.1 implementation stages 1–5 are complete. Stage 6 is active:

1. versioned lifecycle contract and atomic persistence — complete;
2. operation journal, conflict locks, dependent-operation blocking, and cancellation foundation — complete;
3. read-only managed-storage inventory and startup reconciliation — complete;
4. Desktop Service initialization, compatibility migration, lifecycle gating, and rollback integration — complete;
5. bounded Bootstrap/Wails surface and minimum existing-design UI — complete;
6. final acceptance audit, documentation, protected CI, diff review, and PR handoff — in progress.

No further M5.1 feature stage is planned. After Stage 6, the only legal next step is the M5.1 Closing Review; M5.2 must not begin automatically.

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

## Validation status

The implementation has passed, on current or immediately preceding PR heads:

- Governance scope checks;
- Go formatting, vet, unit/race coverage, and builds;
- lifecycle schema, persistence, rollback, duplicate/conflict, active-session, dependent-operation, cancellation, shutdown, and startup-reconciliation tests;
- Linux and Windows filesystem safety tests, including symlink and reparse handling;
- frontend typecheck, lifecycle policy tests, existing tests, and production build;
- Windows and Linux Wails builds;
- official adapter checks on Windows and Linux;
- Linux real-browser Evidence and both official adapter browser paths;
- exact Windows reviewed-Chromium installation and managed identity/window Evidence in recent runs.

Governance passed on the final code-and-documentation series. The complete protected CI matrix must pass on the current PR head before PR #52 can leave Draft.

### Independent CI reliability observation

A recent Windows Network Evidence attempt failed after the exact identity/window Evidence succeeded. The diagnostic packet reported Chromium Sandbox access denial for the same fixed user-local `chrome.exe` path (`Access is denied (0x5)`), followed by Network Evidence timeout. This matches open Issue #49 and does not call the lifecycle Service or frontend.

No Sandbox, browser binary, Provider identity, Evidence rule, workflow permission, or compatibility claim has been weakened in PR #52. The protected matrix must pass before merge; unrelated experimental ACL work remains outside this PR.

A separate Linux collector shutdown timeout occurred once during implementation and passed on retry without code changes. It remains an observed CI timing fluctuation, not an M5.1 product claim.

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

1. Allow the complete protected CI matrix on the current PR head to finish.
2. Confirm the final diff still contains only Issue #45 scope and no M5.2–M5.4 implementation.
3. Confirm PR comments, reviews, and unresolved threads remain clear.
4. Mark PR #52 ready for review only when required checks pass and no actionable review item remains.
5. Merge only after the owner approves the reviewed PR.
6. Open a dedicated M5.1 Closing Review after merge.
7. Advance STATUS to M5.2 only if that Closing Review passes.
