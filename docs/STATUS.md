# Current Project Status

Last updated: 2026-07-23
Application version: 0.15.0-dev
Main baseline SHA: ffcf25d94cd821c82f07cc49fc61130d3e02fcdb
Current phase: Phase 5
Current milestone: Consolidated M5.3 and M5.4 product completion
Current task: Complete and validate PR #59 on `agent/handoff-m5-3`
Current implementation stage: M5.3 product scope and the approved M5.4 metadata, bulk export, health, storage inventory, and manual repair-plan surfaces are implemented online and awaiting executable validation

## Operational rule

PR #59 and branch `agent/handoff-m5-3` are the only remaining Phase 5 development path. No additional development branch, handoff PR, Closing Review PR, temporary Issue, or workflow is authorized.

GitHub Actions remain unavailable and must not be created, enabled, modified, manually triggered, or rerun. Online connector development may continue in concentrated commits, but compilation, tests, Wails execution, and Windows packaging must be reported as unverified until they are actually run in a suitable environment.

## Completed baseline

### M5.1 — Lifecycle foundation

M5.1 is merged and frozen. It provides versioned lifecycle records, the authoritative operation journal, per-Profile locks, active-session blockers, cancellation state, per-item results, bounded storage inventory, startup reconciliation, Desktop integration, and minimum lifecycle UI.

### M5.2 — Safe local recovery

M5.2 is merged and frozen. It provides verified same-machine snapshots, restore to a new identity, archive/unarchive, recoverable trash, exact restore-trash, explicit permanent deletion, conservative reconciliation, and the Local recovery workspace.

## Implemented in PR #59

### Portable Profile definitions and templates

- strict versioned portable Profile JSON with canonical SHA-256 tamper detection;
- exclusion of browser data, credentials, binaries, local IDs, local paths, runtime data, logs, and Evidence;
- explicit new-identity and advanced preserve-identity modes;
- exact fail-closed Kernel and adapter remapping;
- explicit local operating-system vault credential selection;
- import preview and creation of a new local Profile without implicit overwrite;
- private template catalog with create, list, apply, and delete;
- import and template results remain `draft` with explicit validation and Evidence limitations;
- template application always creates a new Profile ID, managed directory, and fingerprint seed;
- visible Wails Desktop UI for export, import preview, dependency selection, and templates;
- M5.1 journal, idempotency, source/destination locks, per-item results, and rollback integration.

### Bounded multi-Profile metadata

- fixed, normalized, deterministic selection of up to the lifecycle operation bound;
- available/draft lifecycle and active-session preflight;
- one authoritative `bulk-metadata-update` journal operation and locks for the complete accepted selection;
- bounded group replacement plus case-insensitive tag add/remove;
- source revision conflict detection after preflight;
- cancellation checks before each next Profile;
- truthful succeeded, skipped, cancelled, failed, partial, and idempotently reused item results;
- no bulk start, scheduling, proxy rotation, browser-data mutation, or silent inclusion of newly created Profiles.

### Bounded multi-Profile portable export

- one fixed selection of available, stopped, unlocked Profiles;
- one authoritative export journal operation with complete-selection locks and deterministic idempotency;
- a user-selected existing directory and one collision-resistant JSON filename per Profile;
- existing files, links, aliases, and special destinations are rejected rather than overwritten;
- source revision and lock ownership are revalidated before each artifact is published;
- cancellation prevents the next Profile export from starting;
- per-item success, skip, cancellation, failure, partial aggregate status, and idempotent artifact verification;
- explicit advanced preserve-identity warning while secrets, browser data, binaries, local IDs, paths, trust, and Evidence remain excluded.

### Bulk Profile health refresh

- fixed stopped-Profile selection using the authoritative `bulk-health-refresh` lifecycle operation;
- deterministic idempotency, complete-selection locks, revision revalidation, cancellation between Profiles, and per-item results;
- read-only checks for lifecycle state, managed Kernel integrity, route/adapter/credential validation, fingerprint capability policy, identity/window consistency, and managed browser-data containment;
- explicit `ready`, `limited`, and `blocked` reports with persisted bounded check codes for deterministic retry results;
- a completed health assessment remains distinct from a healthy result, so a blocked Profile is reported truthfully without making the operation itself fail;
- visible Desktop health cards and check-level explanations in the Multi-Profile tools dock;
- service-level tests use real Profile and lifecycle stores to cover ready, blocked, and idempotently reused results.

### Read-only storage management and repair plans

- bounded refresh of the existing managed Profile inventory;
- visible Profile file/byte summaries and missing, incomplete, orphan, or unsafe findings;
- verified snapshot and retained trash totals;
- lifecycle operation history count;
- snapshot-aware suggestions for missing Profile data;
- explicit orphan ownership, unsafe entry, and incomplete-scan review plans;
- every repair plan is manual and observational; no automatic cleanup, deletion, move, restore, orphan repair, or filesystem browser;
- visible Desktop access through the Phase 5 Multi-Profile tools dock.

## Remaining before merge

1. Perform the final complete changed-file, interface, data-contract, and failure-path review.
2. Run Go formatting/vet/tests, frontend typecheck/tests/build, Wails development startup, Windows amd64 build, and manual smoke testing in an executable environment.
3. Fix any issues found by executable validation.
4. Do not claim build, test, package, or manual-test success until those checks are actually completed.

## Frozen security boundaries

- browser contents remain opaque and separate from Profile metadata;
- vault secrets remain local and non-portable;
- local IDs and absolute paths are not portable identities;
- imported metadata cannot manufacture Provider trust, capability support, compatibility, health, or Evidence;
- destructive work remains serialized per Profile and recoverable operations preserve the only healthy copy until verification;
- bulk export never overwrites an existing file and never stores destination paths in lifecycle item results;
- health refresh is observational and cannot silently mutate a Profile, promote trust, or create Evidence;
- storage inventory and repair plans are observational and never authorize automatic mutation;
- unsupported, unsafe, contradictory, missing, modified, or unverifiable state fails closed or remains explicitly limited.

## Validation status

The current online implementation has received static review for Go syntax and formatting, JSON field consistency, Wails method naming, TypeScript interface compatibility, lifecycle lock ownership, deterministic selection, idempotency, cancellation checkpoints, per-item result derivation, portable artifact identity, destination containment, health-check persistence, and non-destructive storage behavior.

No GitHub Actions run was requested. The branch has not yet been fully compiled, executed, packaged, or manually smoke-tested in this connector environment, and PR #59 is not ready to merge until that validation is performed.
