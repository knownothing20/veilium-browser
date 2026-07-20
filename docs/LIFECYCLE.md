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

The first M5.1 slice defines two independent JSON stores:

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

Existing Profiles without lifecycle records are not rewritten during read-only compatibility inspection. A compatibility record may be constructed as `available` only when the caller has already proved the Profile metadata and managed path are safe. `available` does not mean healthy or reviewed.

## Operation journal

The operation journal reserves the approved Phase 5 operation vocabulary, but M5.1 does not implement later snapshot, restore, portable-definition, template, archive/trash data movement, permanent deletion, or batch product behavior.

Each operation records:

- version, stable ID, type, deterministic Profile selection, and optional idempotency identity;
- stage, status, timestamps, cancellation request, and safe cancellation stage;
- bounded item results, limitations, and recovery actions;
- managed relative staging or quarantine references;
- application version, platform, and optimistic revision.

Terminal statuses require a completion timestamp and item results. `completed` is accepted only when every item succeeded. Idempotent duplicate requests return the existing operation; contradictory reuse fails with a conflict.

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

## Current implementation boundary

This module currently supplies the policy and persistence foundation only. Service integration, Profile locks, active-session blocking, read-only storage inventory, startup reconciliation, and bounded desktop/UI presentation remain in the same M5.1 Draft PR and must build on these stores without broadening Phase 5 scope.
