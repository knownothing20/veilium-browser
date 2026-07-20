# Profile Lifecycle Foundation

Status: M5.1 implementation in progress
Phase: Phase 5
Milestone: M5.1 — Lifecycle Contract, Inventory, and Operation Journal

## Purpose

The `internal/lifecycle` package owns versioned Profile lifecycle records and the durable lifecycle operation journal. These records are intentionally separate from:

- legacy `profiles.json` Profile definitions;
- browser runtime sessions;
- derived Profile health and compatibility;
- Provider trust and capability claims;
- browser and Network Evidence;
- operating-system credential-vault secrets;
- browser user-data contents.

A lifecycle state can restrict product behavior, but it cannot grant launchability, reviewed trust, health, compatibility, or Evidence validity.

## Persisted files

M5.1 defines two independent JSON stores:

- lifecycle records, normally placed under the application data root as `lifecycle.json`;
- lifecycle operation records, normally placed under the application data root as `lifecycle-operations.json`.

Both stores use a versioned envelope and versioned records. They are bounded to 8 MiB and 4,096 records. Operation requests are additionally bounded to 256 selected Profiles and 256 item results.

## Lifecycle record

A lifecycle record contains:

- schema version;
- Profile ID;
- lifecycle state;
- canonical relative managed-directory identity;
- timestamps;
- optional archive, trash, retention, and source references;
- optional operation lock;
- recovery and limitation codes;
- optimistic revision.

Allowed states are `available`, `draft`, `archived`, `trashed`, and `invalid`.

Existing Profiles without lifecycle records are handled through an explicit, reported compatibility reconciliation. Safe managed Profile paths may be mapped to `available`; contradictory or unmanaged paths must be represented as limited/invalid rather than promoted. `available` does not mean healthy or reviewed.

## Operation journal

The operation journal reserves the approved Phase 5 operation vocabulary, but M5.1 does not implement later snapshot, restore, portable-definition, template, archive/trash data movement, permanent deletion, or batch product behavior.

Each operation records:

- version, stable ID, type, deterministic Profile selection, and optional idempotency identity;
- stage, status, timestamps, cancellation request, and safe cancellation stage;
- bounded item results, limitations, and recovery actions;
- managed relative staging or quarantine references;
- application version, platform, and optimistic revision.

Terminal statuses require a completion timestamp and item results. `completed` is accepted only when every item succeeded. Idempotent duplicate requests return the existing operation; contradictory reuse fails with a conflict.

## Coordination and cancellation

The coordinator provides the M5.1 policy foundation for future lifecycle operations:

- active browser sessions and protected dependent operations are checked before an operation is journaled;
- Profile locks are acquired for the complete selected set in one persistence transaction;
- a conflicting lock prevents partial acquisition;
- repeated requests with the same idempotency identity return the existing operation;
- a successfully locked pending operation moves to `running` and stage `locked`;
- a journal-transition failure releases newly acquired locks and leaves the accepted operation pending for startup reconciliation;
- cancellation is a durable request and does not imply completion or success;
- terminal operation state is written before locks are released, so a failed unlock remains visible and recoverable.

This package does not perform any snapshot, restore, archive, trash, delete, import, export, or template data movement.

## Read-only managed-storage inventory

The inventory scanner reports filesystem facts without repairing or deleting anything. It reports:

- expected managed Profile directories that are present or missing;
- bounded regular-file counts and byte sizes without opening browser files;
- unexpected managed Profile directories as orphans;
- non-directory orphan entries;
- symlinks, Windows reparse points, unsafe managed roots, and special files;
- cancelled, bounded, or otherwise incomplete scans as incomplete rather than successful.

Reports expose relative managed paths only. Duplicate lifecycle records that point to the same managed directory fail closed.

## Startup reconciliation

Startup reconciliation:

- creates missing compatibility records in one explicit, reported batch;
- changes interrupted non-terminal operations to `recovery-required`, never to `completed`;
- creates per-Profile `recovery-required` item results;
- clears locks only after their owning operation is terminal or confirmed missing;
- records recovery codes when stale locks are cleared;
- reports recognized staging and quarantine paths without deleting or resuming them;
- runs the read-only managed-storage inventory and preserves incomplete limitations.

Malformed, duplicate, unsupported, or contradictory persisted stores fail service initialization instead of being silently rewritten.

## Persistence and rollback

Stores:

- reject symlinks and non-regular store files;
- reject unsafe directory components;
- require private file permissions on Unix-like systems;
- reject oversized, malformed, unknown-field, trailing-data, duplicate-ID, duplicate-idempotency, and unsupported-version input;
- write through a private temporary file;
- flush the temporary file before rename;
- replace the authoritative file only after successful encoding and validation;
- retain the previous in-memory and on-disk state when persistence fails;
- use optimistic revisions to reject stale updates.

Downgrade behavior is fail-closed: an older build rejects unsupported future envelopes or records and does not rewrite them.

## Security and privacy

Lifecycle files may contain IDs, states, stages, timestamps, bounded counts, relative managed references, redacted reason codes, and recovery instructions. They must not contain:

- resolved credentials or vault contents;
- proxy passwords or generated private adapter configuration;
- Cookies, LocalStorage, IndexedDB, history, tokens, extension data, or arbitrary browser contents;
- executable bytes or copied Provider trust;
- browser or Network Evidence payloads.

## Validation completed for the module layer

The current module layer passes:

- Go formatting and vet;
- race-enabled lifecycle unit tests;
- Linux compilation and tests;
- Windows amd64 cross-compilation;
- persistence reopen and simulated write-failure rollback;
- strict schema, duplicate, unknown-field, future-version, symlink, conflict, cancellation, inventory-bound, and startup-reconciliation cases.

## Remaining M5.1 integration

The following work remains in Draft PR #52:

- open and reconcile lifecycle stores during desktop-service initialization;
- synchronize new Profile creation and cloning with lifecycle records;
- block launch for non-`available` states and block unsafe legacy deletion until the M5.2 data-movement workflow exists;
- expose lifecycle records, operations, inventory, cancellation availability, and reconciliation results through desktop bindings;
- add the minimum existing-design UI needed to distinguish lifecycle, runtime, health, operation, and storage state;
- add service, Wails, frontend, and real-filesystem integration tests.
