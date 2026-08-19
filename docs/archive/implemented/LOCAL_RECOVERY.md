# Local Recovery Contracts

Status: M5.2 Stages 1–5 implementation and validation
Phase: Phase 5
Milestone: M5.2 — Safe Local Recovery
Authority: Issue #54

## Current boundary

Stages 1–5 provide versioned local recovery records, verified same-machine snapshots, restore to a new limited identity, archive/unarchive, recoverable trash/restore, explicit irreversible cleanup, conservative trash reconciliation, bounded Desktop/Wails methods, and a minimum local recovery workspace.

All operations reuse the M5.1 lifecycle journal, Profile locks, active-session and dependent-operation blockers, cancellation state, item results, managed references, and recovery codes. No parallel task system is introduced.

Automatic retention cleanup, cross-machine portability, templates, batch operations, a general filesystem browser, remote APIs, and Provider or Evidence expansion remain prohibited.

## Shared integrity rules

- Browser user-data files are treated as opaque bytes.
- Managed paths are canonical, relative, and contained inside reviewed application roots.
- Absolute paths, traversal, path collisions, links, reparse points, special files, and hard-link ambiguity fail closed.
- File count, individual size, total size, path length, encoded record size, and operation identity are bounded.
- Authoritative JSON uses strict decoding and rejects unknown fields, trailing data, unsupported versions, duplicates, and contradictions.
- Persistent catalogs use private temporary files and atomic replacement.
- The only healthy copy remains recoverable until replacement state is verified.
- Credential secrets, Kernel and adapter binaries, runtime data, private logs, and Evidence payloads are excluded.
- Lifecycle state does not manufacture health, Provider trust, compatibility, or applicable Evidence.

## Persisted records

### Local snapshot manifest

The immutable manifest records source Profile metadata identity, platform, canonical file entries, file and tree digests, bounded totals, non-secret dependency requirements, exclusions, portability limitations, and application/schema versions.

### Local recovery catalog

The catalog records the immutable manifest reference and digest, tree identity, file totals, verification state, timestamps, limitations, and optimistic revision.

### Trash catalog

The versioned Trash Catalog records:

- Trash ID and Profile ID;
- current operating system and architecture;
- exact original lifecycle state, managed directory, archive timestamp, source ID, recovery codes, and limitation codes;
- private trash reference;
- retained Profile-definition digest;
- browser file-tree digest, count, and total bytes;
- whether recoverable browser data is still present;
- visible trash and retention timestamps;
- optional irreversible-deletion timestamp;
- explicit status, limitations, and optimistic revision.

Only one Trash Record may exist for a Profile. Immutable identity fields cannot change during status transitions.

Trash statuses are:

- `pending` — the transaction is registered but not committed;
- `stored` — the Profile data is verified in private recoverable trash;
- `restoring` — restoration to the original managed location is in progress;
- `cleanup-pending` — explicit irreversible cleanup has crossed its confirmation gate but not yet completed;
- `deleted` — payload and Profile metadata are gone and bounded audit tombstones remain;
- `recovery-required` — files, lifecycle metadata, catalog state, or staging require manual reconciliation.

Retention is visible metadata only. No background process automatically deletes trash.

## Stage 1 — Contracts and persistence

Stage 1 provides versioned manifests and catalogs, deterministic file-tree and Profile-definition digests, canonical path validation, strict resource bounds, immutable manifest publication, atomic catalog persistence, optimistic revisions, explicit transitions, and persistence rollback.

## Stage 2 — Local snapshot creation

Snapshot creation:

1. validates the request before journaling;
2. blocks active or protected work and locks the source Profile;
3. verifies the managed source directory;
4. performs bounded link-safe traversal and destination-space preflight;
5. copies regular files into private staging while checking stable file identity;
6. hashes every file and revalidates the complete source and staged trees;
7. checks cancellation before publication;
8. atomically publishes the verified snapshot;
9. finalizes the catalog and M5.1 operation only after verification.

Snapshot creation never moves or changes the original Profile data.

## Stage 3 — Restore to a new identity

Restore defaults to a new deterministic Profile ID, new managed directory, and independently derived fingerprint seed. It never overwrites the source Profile or reuses source local Kernel, adapter, credential, or Evidence identities.

The restore transaction strictly revalidates the snapshot, maps current local dependencies conservatively, builds a reviewed replacement definition, copies into private staging, verifies the entire staged tree, atomically activates the new browser directory, persists Profile metadata, and leaves the new lifecycle record `draft` until current dependencies and validation pass.

Cancellation, tamper, target conflict, activation failure, metadata persistence failure, cleanup failure, and operation-finalization ambiguity have explicit rollback or recovery-required outcomes.

## Stage 4 — Local lifecycle storage operations

### Archive and unarchive

Archive changes lifecycle metadata only; it does not move browser files.

- `available` and `draft` Profiles may be archived.
- The exact origin state is persisted.
- Unarchive restores the exact origin state rather than always promoting to `available`.
- Existing limitations are preserved.
- Active sessions, protected work, conflicting locks, cancellation, unsafe managed paths, and contradictory timestamps fail closed.
- Lifecycle persistence failure leaves the original state unchanged.
- A committed state with failed journal finalization receives an explicit recovery code.

### Recoverable trash transaction

Recoverable trash preserves Profile metadata and moves only the Profile-owned managed browser directory.

Transaction order:

1. validate request, retention metadata, lifecycle state, Profile metadata, and current blockers;
2. acquire the M5.1 operation lock;
3. resolve exactly `profiles/<ProfileID>` and safely inventory and hash the complete browser tree;
4. serialize and digest the non-secret retained Profile definition;
5. create a `pending` Trash Record before moving files;
6. create private operation staging and persist the Profile definition there;
7. check cancellation and revalidate retained Profile metadata;
8. rename the managed browser directory into private staging;
9. verify every moved file and the complete deterministic tree identity;
10. atomically publish staging to `local-recovery/trash/<TrashID>`;
11. change the Trash Record to `stored`;
12. only then commit lifecycle state `trashed`, timestamps, retention metadata, and trash markers;
13. finish the M5.1 operation and release the lock.

If any precommit cleanup or rollback cannot remove its catalog record, the record is retained as `recovery-required`; recovery metadata is never deleted before the only authoritative copy is restored.

### Restore from trash

Restore-from-trash requires lifecycle state `trashed`, a `stored` Trash Record, an absent original target directory, and retained Profile metadata whose digest still matches the Trash Record.

Transaction order:

1. verify the stored Profile definition and complete browser tree;
2. transition the Trash Record to `restoring`;
3. move the private trash root into operation staging;
4. verify it again after the move;
5. atomically activate browser data at the exact original managed directory;
6. restore the exact original lifecycle state, archive timestamp, source ID, recovery codes, and limitations;
7. remove the Trash Record;
8. clean owned staging and finalize the operation.

Target conflicts never overwrite existing files. Failure before activation returns the payload to private trash. Failure after activation attempts both lifecycle and payload rollback. Any failed catalog reset, rollback, or staging cleanup becomes explicit recovery state.

### Explicit irreversible cleanup

Irreversible cleanup requires an exact confirmation equal to the Profile ID. Retention expiry alone never authorizes deletion.

The operation:

1. revalidates the `stored` Trash Record, retained Profile metadata, and full payload;
2. transitions the record to `cleanup-pending`;
3. moves the owned trash root into private delete staging;
4. verifies the staged payload again;
5. records the irreversible operation stage;
6. deletes only the verified browser-data subtree inside the owned staging boundary;
7. deletes the matching Profile metadata;
8. removes remaining owned staging;
9. changes lifecycle state to an `invalid` audit tombstone;
10. changes the Trash Record to a `deleted` audit tombstone with `DataPresent=false` and a deletion timestamp;
11. finalizes the operation and releases the lock.

Directory cleanup rechecks containment, links, reparse points, and special entries immediately before removal. Failure after the irreversible boundary is never reported as success; the Trash Record and lifecycle record become recovery-required with truthful information about whether browser data remains.

### Startup reconciliation

The Stage 4 reconciler is observational and conservative. It never guesses which copy is authoritative and never deletes or moves data automatically.

For each Trash Record it compares:

- lifecycle state and lock;
- original managed source-path presence;
- private trash-root presence;
- operation staging and quarantine references;
- retained Profile metadata presence and digest;
- Trash Record status and `DataPresent` claim.

Healthy `stored` and `deleted` states remain unchanged. Interrupted `pending`, `restoring`, or `cleanup-pending` records, stale locks, duplicate source/trash copies, missing Profile metadata, changed Profile definitions, unsafe paths, or contradictory tombstones become `recovery-required` and receive bounded recovery findings.

## Stage 5 — Desktop/Wails API and minimum UI

The Desktop boundary opens the versioned catalogs, initializes conservative trash reconciliation, and projects the existing M5.1 operation journal into bounded progress state.

It exposes:

- local recovery state, snapshot and trash lists, snapshot details, and Profile preflight;
- snapshot creation and restore to a new identity;
- archive and unarchive;
- recoverable trash and exact restore-trash;
- permanent deletion with exact Profile-ID confirmation;
- refresh/reconciliation and safe cancellation requests.

Preflight reports lifecycle state, managed-storage inventory, active browser state, operation locks, matching trash identity, retention metadata, and action availability. It is advisory only; every executor performs authoritative validation again before changing state.

The minimum Local recovery workspace provides Profile actions, verified snapshot cards, recoverable trash cards, operation stage/file/byte progress, bounded history, safe-cancellation availability, and recovery-required findings. Browser preview mode remains read-only and disabled for mutations.

The existing Wails Profile delete affordance routes eligible stopped Profiles through recoverable trash. The lower-level direct metadata deletion path remains fail-closed.

Full Desktop details are recorded in `docs/LOCAL_RECOVERY_DESKTOP.md`.

## Validation coverage

The retained suite covers:

- strict schema, path, digest, size, persistence, and rollback contracts;
- snapshot success, idempotency, cancellation, source mutation, insufficient space, publication failure, and cleanup failure;
- restore new identity, dependency resolution, secret exclusion, idempotency, tamper rejection, conflicts, activation rollback, metadata rollback, and cleanup recovery;
- archive/unarchive round trips for `available` and `draft` origins;
- archive blockers, cancellation, unsafe path rejection, persistence failure, and finalization ambiguity;
- recoverable trash and exact restore for `available`, `draft`, and `archived` origins;
- Trash Catalog transitions, uniqueness, reopen, optimistic revision, and persistence rollback;
- active-session and protected-operation blockers;
- cancellation before move, rename failure, lifecycle persistence failure, catalog cleanup failure, and target conflict;
- changed or missing retained Profile metadata;
- exact irreversible confirmation and irreversible-cleanup failure;
- bounded audit tombstones and idempotent retry;
- healthy and contradictory startup reconciliation;
- Desktop preflight, archive/unarchive, snapshot listing, trash/restore, and failed irreversible confirmation;
- frontend typecheck, unit tests, production build, and browser-preview gating;
- Windows and Linux unit, real-filesystem, and Wails behavior;
- official adapter, Linux browser, and exact Windows reviewed-Chromium identity, Evidence, Network Evidence, tamper, artifact, and cleanup regressions.

Issue #49 remains a separate hosted-runner reliability risk. M5.2 does not weaken Sandbox, identity, Network Evidence, tamper, artifact, or cleanup requirements.

Stage 6 may perform only final integration, documentation, scope review, protected-CI confirmation, PR readiness, and the post-merge Closing Review handoff. M5.3 and M5.4 remain blocked.
