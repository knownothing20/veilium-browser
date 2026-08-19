# Profile Lifecycle Foundation

Status: M5.1 implementation ready for final validation
Phase: Phase 5
Milestone: M5.1 — Lifecycle Contract, Inventory, and Operation Journal
Implementation issue: #45
Draft pull request: #52

## Purpose

The `internal/lifecycle` package and its bounded desktop integration provide one truthful local source of Profile lifecycle, operation, storage-inventory, and recovery state. Lifecycle state remains separate from:

- legacy `profiles.json` Profile definitions;
- browser runtime sessions;
- derived Profile health and compatibility;
- Provider trust and capability claims;
- browser and Network Evidence;
- operating-system credential-vault secrets;
- opaque browser user-data contents.

Lifecycle state can restrict product behavior. It cannot grant launchability, health, reviewed Provider trust, capability support, compatibility, or Evidence validity.

## Persisted files and bounds

M5.1 defines two independent JSON stores under the private application data root:

- `lifecycle.json` — versioned Profile lifecycle records;
- `lifecycle-operations.json` — versioned lifecycle operation records.

Both stores:

- use a versioned envelope and versioned records;
- are bounded to 8 MiB and 4,096 records;
- reject unknown required structure, trailing data, duplicate IDs, unsupported versions, symlinks, and non-regular authoritative files;
- use private temporary files, flush before rename, and replace authoritative state only after successful encoding and validation;
- retain the previous in-memory and on-disk state when persistence fails;
- use optimistic revisions to reject stale updates.

Operation records are additionally bounded to 256 selected Profiles and 256 per-item results. Text, identifiers, code lists, relative paths, and reports have explicit limits.

## Lifecycle records

A lifecycle record contains:

- schema version and Profile ID;
- lifecycle state;
- canonical relative managed-directory identity;
- created and updated timestamps;
- optional archive, trash, retention, and source references reserved by the contract;
- optional operation lock;
- recovery and limitation codes;
- optimistic revision.

Allowed states are:

- `available`;
- `draft`;
- `archived`;
- `trashed`;
- `invalid`.

`available` means only that lifecycle policy does not independently block the Profile. Kernel, adapter, route, fingerprint, managed-path, consistency, health, and Evidence checks remain independently required.

## Conservative compatibility and migration

Existing `profiles.json` records remain readable. Desktop startup performs an explicit, idempotent compatibility reconciliation:

- an existing Profile with a non-empty name and its exact expected managed user-data path receives an `available` lifecycle record;
- an unmanaged, contradictory, or incomplete legacy Profile receives `invalid` plus bounded limitation codes;
- compatibility creation is written as one atomic batch and reported in startup reconciliation;
- no browser user data, Provider identity, dependency record, Evidence, or credential is rewritten;
- an existing lifecycle record is never optimistically replaced by compatibility mapping;
- managed-directory identity drift fails closed rather than silently rewriting the record.

A newly saved Profile that intentionally retains a legacy unmanaged path remains readable but receives `invalid`; launch and launch-plan creation remain blocked. This preserves the previous metadata compatibility contract without treating unsafe storage as launchable.

Downgrade behavior is fail-closed. Older builds reject unsupported future envelopes or records and do not rewrite them.

## Operation journal

The journal reserves the approved Phase 5 operation vocabulary, but M5.1 implements only the foundation. Each record contains:

- schema version, stable operation ID, and operation type;
- deterministic selected Profile IDs;
- optional idempotency and predecessor identity;
- status, stage, and timestamps;
- cancellation request and safe-cancellation stage;
- bounded per-item results and reasons;
- limitations and recovery actions;
- managed relative staging or quarantine references;
- application version, platform, and optimistic revision.

Terminal statuses require a completion timestamp and item results. `completed` is valid only when every required item succeeded. Cancelled, interrupted, partial, failed, and recovery-required work cannot become completed implicitly.

## Conflict, dependent-operation, and cancellation policy

The coordinator provides the M5.1 policy foundation:

- active browser sessions and protected dependent operations are checked before an operation is journaled;
- Profile locks are acquired for the complete selected set in one persistence transaction;
- one conflicting lock prevents partial acquisition;
- repeated requests with the same idempotency identity return the existing operation;
- a newly locked accepted operation moves to `running` and stage `locked`;
- a journal transition failure releases newly acquired locks and leaves durable accepted state for reconciliation;
- cancellation is a durable request, checked only by later operation implementations at declared safe checkpoints;
- terminal state is written before locks are released, so unlock failure remains visible and recoverable.

M5.1 does not expose a cancellation action because it does not execute later lifecycle data operations. The desktop bootstrap and UI expose whether cancellation was requested and whether a safe cancellation stage is declared.

## Read-only managed-storage inventory

The inventory scanner reports filesystem facts without opening browser files for content inspection and without repairing or deleting anything. It reports:

- expected managed Profile directories that are present or missing;
- bounded regular-file counts and byte totals;
- unexpected managed Profile directories as orphan candidates;
- non-directory orphan entries;
- unsafe roots, symlinks, Windows junction/reparse paths, and special files;
- cancelled, bounded, or otherwise incomplete scans as incomplete rather than successful.

Reports use relative managed paths. Duplicate lifecycle records pointing to one managed directory fail closed. Inventory never parses Cookies, LocalStorage, IndexedDB, history, extension data, tokens, page data, or Evidence payloads.

## Startup reconciliation and shutdown handoff

Desktop startup:

1. creates a missing application data root with private permissions;
2. rejects a symlink, reparse-unsafe, non-directory, malformed, duplicate, contradictory, or unsupported lifecycle baseline;
3. opens lifecycle and operation stores;
4. creates conservative compatibility records;
5. converts interrupted non-terminal operations to `recovery-required`, never `completed`;
6. creates per-Profile recovery-required item results;
7. clears stale locks only after the owning operation is terminal or confirmed missing and records a recovery code;
8. reports recognized staging and quarantine paths without deleting or resuming them;
9. runs the read-only bounded inventory.

Application shutdown does not invent terminal lifecycle results. A durable running operation and lock remain available for the next startup, where reconciliation converts them into explicit recovery state and safely reconciles the stale lock.

## Desktop service behavior

The desktop Service owns the lifecycle record store, journal, coordinator, scanner, and latest startup report.

- malformed lifecycle state prevents application initialization;
- Profile creation first writes Profile metadata, then lifecycle metadata; lifecycle failure rolls Profile metadata back;
- the metadata-only lifecycle rollback helper never touches managed browser files;
- cloning requires the source Profile to be `available` and unlocked and creates a new lifecycle record for the new identity;
- launch and launch-plan creation require `available` and unlocked lifecycle state in addition to all existing Phase 4 checks;
- editing is blocked while locked, archived, or trashed; invalid and draft Profiles remain inspectable/editable for repair;
- direct Profile deletion fails closed until the M5.2 trash transaction exists.

No M5.1 Service path moves, copies, archives, restores, quarantines, trashes, or permanently deletes browser data.

## Desktop bootstrap and UI

`Bootstrap()` exposes the bounded read-only lifecycle surface:

- lifecycle records;
- operation records;
- startup reconciliation actions;
- startup storage inventory;
- operation cancellation-request and safe-stage fields.

The existing desktop design is extended without adding a new application area or unrelated redesign:

- Profile rows display runtime/health status separately from lifecycle state;
- limitation, recovery, and lock reasons are visible in bounded form;
- start, launch-plan, diagnostics, Evidence, clone, and edit controls are disabled according to lifecycle policy;
- delete remains disabled and identifies M5.2 trash as deferred;
- the dashboard shows available, limited, locked, recovery, missing, orphan, and unsafe summaries;
- recent operation records show type, status, stage, selected Profile count, and cancellation availability;
- browser-preview mode creates no fake lifecycle records or support claims.

The UI contains no archive, trash, restore, permanent-delete, cancellation, batch, import/export, template, filesystem-browser, or automation action.

## Security and privacy

Lifecycle files, bootstrap, reports, and UI may contain only bounded IDs, states, stages, timestamps, relative managed references, counts, redacted reason codes, and recovery instructions. They must not contain:

- resolved credentials or operating-system vault contents;
- proxy passwords or generated private adapter configuration;
- Cookies, LocalStorage, IndexedDB, history, tokens, extension data, or arbitrary browser contents;
- executable bytes or copied Provider trust;
- browser or Network Evidence payloads;
- unauthenticated remote-control, public API, MCP, cloud-sync, or telemetry state.

No plaintext-secret fallback is introduced.

## Platform contract

M5.1 lifecycle persistence, Service integration, UI, and real-filesystem inventory are claimed for Windows and Linux only:

- Linux rejects symlink and special-file escapes;
- Windows rejects junction/reparse unsafe paths using platform-specific inspection;
- both platforms build the Wails desktop application and run lifecycle tests;
- macOS lifecycle support remains unclaimed.

Reviewed Chromium support remains the exact Windows amd64 Phase 4 Provider contract. Lifecycle metadata cannot create or expand reviewed Provider, adapter, protocol, fingerprint, compatibility, or Evidence claims.

## Validation coverage

The M5.1 implementation includes tests for:

- schema validation, future-version rejection, unknown fields, trailing data, duplicate IDs, duplicate idempotency, and oversized records;
- private atomic persistence, reopen, optimistic conflicts, and simulated write-failure rollback;
- active-session and protected dependent-operation blocking;
- atomic Profile lock acquisition/release and stale-lock recovery;
- cancellation request, terminal truthfulness, application shutdown handoff, and startup interruption recovery;
- compatibility migration and managed-path contradiction;
- missing, orphaned, unsafe, symlink, junction/reparse, traversal, special-file, cancellation, and inventory-bound filesystem cases;
- Profile creation/clone lifecycle synchronization and rollback boundaries;
- lifecycle launch/edit/clone/delete policy;
- frontend lifecycle policy, cancellation availability, typecheck, tests, and production build;
- Windows and Linux Wails builds;
- retained Phase 4 adapter and browser Evidence matrices.

The final Draft PR still requires Governance, the complete protected CI matrix, final diff review, and a dedicated M5.1 Closing Review before M5.2 can be considered.

## Explicit non-scope retained

M5.1 does not implement:

- snapshot container creation or snapshot/restore execution;
- archive, trash, restore-trash, retention, or permanent-delete data movement;
- portable Profile export/import;
- templates;
- Cookie or extension management;
- secret export;
- multi-Profile batch UI or operations;
- scheduling, bulk start, proxy rotation, account farming, general automation, MCP, sync, or release work;
- Provider, Kernel, adapter, fingerprint, proxy-protocol, compatibility, or Evidence claim expansion.
