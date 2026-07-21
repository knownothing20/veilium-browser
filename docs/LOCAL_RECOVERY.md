# Local Recovery Contracts

Status: M5.2 Stage 3 implementation
Phase: Phase 5
Milestone: M5.2 — Safe Local Recovery
Authority: Issue #54

## Current boundary

Stages 1–3 implement versioned local recovery records, verified same-machine snapshots, and restore to a new limited identity.

The implementation reuses the M5.1 lifecycle operation journal, Profile locks, runtime and dependent-operation blockers, cancellation state, storage inventory, and recovery records.

The current boundary does not provide archive, trash, permanent cleanup, Desktop APIs, or UI actions. Those remain blocked until the relevant later stage is activated.

## Persisted records

### Immutable local snapshot manifest

The manifest records:

- schema version and stable snapshot ID;
- same-machine artifact scope;
- source Profile ID, display name, Profile schema version, application version, operating system, and architecture;
- a canonical Profile-definition digest;
- canonical relative browser-file entries with byte size and SHA-256;
- deterministic file-tree identity, file count, and total bytes;
- non-secret Kernel, adapter, and credential requirements;
- explicit excluded-data classes;
- portability and limitation codes.

The manifest is published once and is not replaced in place.

### Local recovery catalog

The catalog records the local manifest reference, verification state, immutable digests and size summary, timestamps, limitations, and optimistic revision.

Catalog creation begins in a non-final state. Supported transitions are explicit. Stale revisions and changes to immutable identity fields fail with a conflict.

### M5.1 lifecycle records and journal

Snapshot and restore execution does not create a second task system. It uses the M5.1 journal and lifecycle records for:

- operation identity and idempotency;
- Profile selection and locking;
- current stage and safe cancellation stage;
- bounded item results and progress totals;
- limitations and recovery actions;
- managed staging references;
- terminal success, cancellation, failure, partial success, or recovery-required state.

## Canonical path policy

Manifest paths use slash-separated relative form. Validation rejects:

- absolute paths;
- `.` or `..` traversal;
- backslashes and alternate data stream separators;
- empty or repeated segments;
- trailing separators, spaces, or dots;
- control characters;
- oversized paths or segments;
- Windows reserved device names;
- Windows case-insensitive path collisions;
- duplicate or unsorted entries.

Filesystem operations independently reject symbolic links, junctions, reparse points, unsupported special entries, path escape, and hard-link ambiguity.

## Bounds

The contract defines explicit limits for:

- manifest and catalog encoded bytes;
- Profile-definition bytes;
- number of catalog records;
- file entries;
- individual file bytes;
- total file bytes;
- identifiers, text, codes, capabilities, paths, and path segments;
- operation duration;
- required destination space.

Manifest JSON encoding validates before materializing the complete output. A conservative entry-size estimate rejects a file-entry set that cannot fit the encoded manifest budget. The final encoded byte limit remains authoritative.

## Deterministic identity

The tree identity is SHA-256 over sorted canonical records containing:

1. relative path;
2. decimal byte size;
3. file SHA-256.

Windows path collision checks use case-insensitive keys while preserving the original canonical path in the manifest.

The Profile-definition digest uses canonical JSON object encoding. It represents the non-secret Profile definition only; it cannot grant Provider trust, compatibility, health, or Evidence applicability.

## Dependency requirements and exclusions

Dependency records contain requirements, not source local record IDs or executable paths.

- reviewed Kernel requirements need an exact binary or package-tree identity;
- official adapter requirements need an exact executable identity;
- Kernel and adapter platforms must match the snapshot source platform;
- credential records contain placeholders and required input classes only;
- resolved credential values remain exclusively in the operating-system vault.

Every manifest explicitly excludes:

- credential secrets;
- Kernel binaries;
- adapter binaries;
- runtime state;
- private runtime logs;
- browser and Network Evidence payloads.

Profile-definition validation rejects secret-like fields and URLs containing embedded usernames or passwords before an operation is journaled.

The implementation treats browser files as opaque data. It does not parse Cookies, LocalStorage, IndexedDB, history, page data, tokens, or extension data.

## Stage 1 — Contracts and persistence

Stage 1 provides:

- versioned manifest and catalog schemas;
- deterministic tree and Profile-definition digests;
- strict JSON decoding and resource bounds;
- immutable manifest publication;
- private atomic catalog persistence;
- optimistic revisions and explicit catalog transitions;
- rollback of in-memory and on-disk state when persistence fails;
- fail-closed behavior for unknown future schemas.

## Stage 2 — Local snapshot creation

### Transaction order

1. Validate the request before creating journal state.
2. Use the M5.1 coordinator to reject active sessions and conflicting operations, then lock the source Profile.
3. Resolve only the Profile lifecycle managed directory and verify it remains inside the application data root.
4. Traverse the source without following links and enforce file, path, size, duration, and destination-space bounds.
5. Create private operation-specific staging.
6. Copy regular files with bounded buffers while checking source identity before and after opening.
7. Hash every copied file and revalidate the complete source plan.
8. Generate the manifest, verify the complete staged tree, and persist a staged catalog record.
9. Check cancellation before publication.
10. Atomically rename verified staging into the immutable snapshot location.
11. Mark the catalog verified and only then finish the M5.1 operation successfully.

### Failure and cancellation

- cancellation is checked before traversal, between files, before verification, and before publication;
- source changes fail the operation and do not publish a snapshot;
- insufficient space fails before copying;
- publication failure removes owned staging when safe;
- cleanup failure or catalog finalization ambiguity becomes recovery-required;
- the original Profile and browser data are never moved or changed by snapshot creation;
- retries use operation and idempotency identities and cannot silently publish a different snapshot.

## Stage 3 — Restore to a new identity

### Identity and applicability

Restore defaults to a new identity and never overwrites the source Profile.

The destination receives:

- a new deterministic Profile ID derived from the snapshot and idempotency request;
- a new managed Profile directory;
- a new fingerprint seed derived independently from the source seed;
- no copied source local Kernel, adapter, or credential record IDs;
- no inherited browser or Network Evidence;
- lifecycle state `draft` with explicit limitations.

Only a verified same-user, same-machine snapshot matching the current operating system and architecture is applicable.

### Dependency resolution

- the snapshot stores requirements rather than source local IDs;
- selected Kernel and adapter records are reverified through their current stores before matching;
- exact Provider, version, platform, architecture, digest, package, official identity, route scheme, and trust requirements fail closed when they do not match;
- explicit capability requirements remain unresolved when the current frozen Provider contract cannot prove them without guessing;
- credential metadata never proves that vault material exists;
- a dependency requiring a secret remains `user-action-required` and its local credential reference is not written into the restored Profile;
- unresolved dependencies keep the restored Profile limited and non-launchable.

### Restore transaction order

1. Derive the destination identity and reserve a `draft` lifecycle record.
2. Create and lock the M5.1 restore operation for that new identity.
3. Re-read the catalog, manifest, Profile definition, and every snapshot file.
4. Recompute manifest, file, and tree identities and reject contradictions or modification.
5. Resolve current local dependencies conservatively and record limitations.
6. Build a reviewed replacement Profile definition with the new ID, directory, seed, and safe local references only.
7. Copy browser files into private restore staging and verify the complete staged result.
8. Check cancellation before activation.
9. Atomically activate the new browser-data directory.
10. Persist the new Profile metadata.
11. Remove restore staging and finish the operation, leaving the lifecycle state as `draft`.

### Rollback and recovery

- cancellation before activation removes staging and the lifecycle reservation;
- snapshot modification fails before activation;
- an existing target ID or directory is a conflict and is never overwritten;
- directory activation failure rolls back staging;
- Profile metadata persistence failure removes the newly activated directory;
- rollback never changes the source Profile or source snapshot;
- cleanup failure after successful activation preserves the restored Profile and records partial/recovery state;
- operation-finalization ambiguity records a lifecycle recovery code rather than claiming silent success;
- idempotent retry returns the same restored identity and verifies that its committed data still matches the source snapshot.

## Persistence and privacy

Authoritative persistence:

- uses strict JSON decoding with unknown-field and trailing-data rejection;
- rejects oversized, unsupported-version, malformed, duplicate, contradictory, linked, reparse, or non-regular authoritative files;
- requires private managed directories and files;
- writes through a private temporary file;
- flushes the temporary file before publication;
- publishes immutable manifests without replacing an existing manifest;
- replaces catalogs and lifecycle journals atomically;
- preserves the only healthy source copy until a replacement is verified;
- never uploads or synchronizes recovery artifacts automatically.

## Validation coverage

The retained suite covers:

- valid manifest and deterministic identity;
- unsafe and colliding paths;
- contradictory size summaries and artifact scope;
- dependency identity and source-platform contradictions;
- strict JSON round trips and unknown-field rejection;
- immutable manifest publication and catalog rollback;
- private files, symlinks, reparse points, special entries, and hard-link ambiguity;
- snapshot success, idempotency, cancellation, source mutation, insufficient space, publication failure, cleanup failure, and catalog-finalization failure;
- restore to a new ID and seed;
- conservative dependency resolution and secret exclusion;
- restore idempotency, cancellation, snapshot tamper rejection, target conflicts, activation failure, metadata persistence failure, and cleanup recovery;
- Windows and Linux unit and real-filesystem behavior;
- frontend, Wails, official adapter, Linux browser, and exact Windows reviewed-Chromium regression checks.

Issue #49 remains a separate hosted-runner reliability risk in the exact Windows reviewed-Chromium Sandbox/Evidence job. It must not be addressed by weakening Sandbox, identity, Network Evidence, tamper, artifact, or cleanup requirements inside M5.2.

Stage 4 remains blocked until Stage 3 implementation, documentation, Windows/Linux validation, and the retained matrix pass on the same current Head.
