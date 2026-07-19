# Phase 5 Profile Lifecycle Contracts

Status: Proposed planning contract
Phase: Phase 5
Contract revision: 1

## Purpose

This document defines the logical records, state meanings, portability boundaries, recovery rules, and security requirements for Phase 5. It does not authorize product implementation and does not require exact Go type names or one specific archive container.

The Phase 5 user outcome, milestone order, non-goals, platform policy, validation matrix, and exit gates are governed by `docs/PHASE_05.md`.

## Contract principles

1. Profile metadata, browser user data, secrets, managed dependencies, Evidence, and runtime data are separate classes of state.
2. Local recovery and cross-device portability are different promises.
3. A copy is not a valid backup until its manifest and complete included content are verified.
4. The original healthy state remains recoverable until a replacement is durably validated.
5. Secrets are never implicitly portable.
6. Local record IDs and absolute paths are not portable identities.
7. Imported data cannot manufacture Provider trust, capability support, Profile health, or Evidence validity.
8. Destructive and long-running operations are preflighted, bounded, cancellable at safe checkpoints, and journaled.
9. Interrupted operations are reconciled explicitly; orphaned data is never deleted merely because it is unexpected.
10. Partial batch success remains partial and is reported per item.

## Required vocabularies

### Profile lifecycle state

Allowed meanings:

- `available` — the normal Profile record is present and may become launchable when all current validation passes;
- `draft` — imported or newly reconstructed Profile metadata exists, but one or more required dependencies or validations are unresolved;
- `archived` — the Profile and data are retained but ordinary launch and editing are intentionally disabled until unarchived;
- `trashed` — the Profile is in recoverable deletion storage and cannot launch;
- `invalid` — lifecycle metadata is missing, contradictory, unsafe, or cannot be reconciled without user action.

Lifecycle state is not runtime session state and is not Profile health. `available` does not mean healthy or reviewed.

### Lifecycle operation status

Allowed meanings:

- `pending` — accepted but not started;
- `running` — actively executing;
- `completed` — every required item completed successfully;
- `partial` — at least one independent item completed and at least one item was skipped, cancelled, or failed;
- `cancelled` — user or shutdown cancellation stopped the operation at a safe checkpoint;
- `failed` — the required result was not committed;
- `recovery-required` — durable staging, quarantine, or journal state requires explicit reconciliation;
- `recovered` — an interrupted operation was safely rolled back or completed during reconciliation.

A terminal status requires completion time and item results. An interrupted operation cannot become `completed` without verifying its committed result.

### Per-item result status

Allowed meanings:

- `succeeded`;
- `skipped`;
- `cancelled`;
- `failed`;
- `rolled-back`;
- `recovery-required`.

### Artifact scope

Allowed Phase 5 scopes:

- `local-full-snapshot` — Profile metadata plus opaque managed browser user data for supported same-machine recovery;
- `portable-definition` — configuration and dependency requirements without browser data, secrets, local paths, or managed binaries;
- `template` — reusable non-secret settings that always create a new identity;
- `operation-report` — bounded local summary of lifecycle execution without browser contents or secrets.

No scope implies cloud synchronization, external certification, or complete cross-platform browser-state portability.

### Dependency resolution status

Allowed meanings:

- `resolved` — mapped to a current local record whose required identity and validation pass;
- `missing` — no candidate local dependency exists;
- `incompatible` — a candidate exists but Provider, version, platform, architecture, digest, integrity, or capability requirements do not match;
- `user-action-required` — explicit selection, installation, credential entry, or acknowledgement is required;
- `unsupported` — the current application cannot satisfy the dependency contract.

Only `resolved` dependencies may contribute to a launchable imported Profile.

## Logical record: ProfileLifecycleRecord

A versioned lifecycle record must contain at least:

- schema version;
- Profile ID;
- lifecycle state;
- managed user-data directory identity or relative managed location;
- created and updated timestamps;
- optional archive or trash timestamp;
- optional retention deadline;
- optional source import/snapshot ID;
- current operation lock/reference;
- recovery or limitation codes;
- record revision for optimistic conflict detection.

It must not contain credential secrets, browser contents, executable bytes, private adapter configuration, or user-supplied optimistic health/trust values.

Existing Profiles without a lifecycle record must map conservatively to `available` only when their current metadata remains valid and their managed path is safe. No Profile is silently moved, archived, or deleted during compatibility mapping.

## Logical record: LifecycleOperation

A versioned operation record must contain at least:

- operation ID;
- operation schema revision;
- operation type;
- requested Profile IDs in deterministic order;
- source and destination descriptors without secrets;
- preflight summary;
- operation status;
- current stage;
- started, updated, and optional completion timestamps;
- bounded progress counts and bytes;
- cancellation-requested flag and safe cancellation stage;
- per-item results;
- limitations and recovery instructions;
- staging/quarantine references expressed only as managed relative identities where possible;
- application version and platform.

Operation types may include:

- `snapshot`;
- `restore`;
- `archive`;
- `unarchive`;
- `trash`;
- `restore-trash`;
- `permanent-delete`;
- `export-definition`;
- `import-definition`;
- `create-template`;
- `apply-template`;
- `bulk-metadata-update`;
- `bulk-health-refresh`;
- `storage-reconcile`.

The journal is not a general automation queue. It does not authorize scheduled or remote execution.

## Logical record: OperationItemResult

Each selected Profile or artifact result must contain:

- item identity;
- result status;
- start and completion timestamps when applicable;
- completed stage;
- bounded bytes/files processed;
- redacted reason and error code;
- committed output identity when successful;
- rollback or recovery identity when applicable;
- limitations.

Raw browser paths outside managed roots, credentials, proxy secrets, browser contents, and private logs are not included.

## Logical record: StorageInventory

A storage inventory must contain:

- generated timestamp;
- managed data-root identity;
- Profile metadata count;
- per-Profile managed-directory existence and bounded size summary;
- snapshot, trash, Evidence, and runtime-log size summaries;
- known operation staging/quarantine references;
- orphan candidates;
- missing expected data;
- unsafe or uninspectable entries;
- proposed repair actions;
- limitations and incomplete scan indicators.

Inventory is observational. It does not delete or repair data automatically.

Large scans must be cancellable and bounded by file count, bytes, duration, and concurrency.

## Logical record: LocalSnapshotManifest

A local full snapshot manifest must contain at least:

- manifest schema version;
- snapshot ID;
- artifact scope `local-full-snapshot`;
- source Profile ID and display name;
- source Profile metadata schema version;
- source application version;
- source OS and architecture;
- creation timestamp;
- Profile definition digest;
- included relative root names;
- deterministic file-tree digest;
- file count and total bytes;
- per-file identity or an equivalent deterministic authenticated tree sufficient to detect additions, removals, path changes, and byte changes;
- Kernel Provider/version and exact dependency requirements;
- proxy-adapter dependency requirements;
- explicit excluded-data list;
- portability classification and limitations;
- optional parent snapshot ID when incremental behavior is separately approved.

The manifest must never contain resolved vault secrets or claim that opaque browser data will decrypt or behave on another user account, machine, OS, architecture, or Provider combination.

### Snapshot file-tree rules

- all included paths are canonical relative paths;
- path comparison rules are documented for each claimed platform;
- absolute paths, traversal, empty paths, duplicate canonical paths, and path collisions are rejected;
- symlinks, junction/reparse escapes, hard-link ambiguity, devices, sockets, named pipes, and unsupported special files are rejected;
- each opened file is verified not to have been swapped after inspection;
- file count, per-file bytes, total bytes, path length, manifest size, and extraction/copy ratio are bounded;
- deterministic ordering is required;
- the final committed snapshot is immutable or treated as invalid if modified.

## Logical record: PortableProfileDefinition

A portable Profile definition must contain at least:

- schema version;
- artifact scope `portable-definition`;
- package ID;
- source application and Profile schema versions;
- export timestamp;
- display metadata, group, notes, and tags;
- fingerprint configuration;
- identity-transfer mode;
- validated non-secret route configuration or an explicit omitted-route marker;
- Kernel dependency requirement;
- optional adapter dependency requirement;
- optional credential requirement without secret value or source local credential ID;
- declared exclusions;
- limitations and compatibility requirements;
- canonical payload digest.

It must not contain:

- source Profile ID as the destination identity;
- absolute or local managed paths;
- local Kernel, adapter, or credential record IDs;
- executable paths or binary contents;
- operating-system vault values;
- browser user data;
- browser or Network Evidence;
- runtime sessions, logs, temporary directories, or generated adapter configurations.

A source ID may appear only as non-authoritative provenance if explicitly approved and must never be reused as the destination Profile ID.

## Logical record: DependencyRequirement

A portable dependency requirement must contain only the minimum needed for safe local resolution.

### Kernel requirement

- Provider ID and contract revision where known;
- browser version;
- supported OS/architecture constraints;
- reviewed/custom/legacy trust requirement without allowing promotion;
- exact executable/package identity when the source depends on a reviewed combination;
- required capability IDs and limitations.

### Adapter requirement

- adapter kind;
- declared version;
- official/custom identity requirement;
- executable identity when exact matching is required;
- supported route scheme;
- platform/architecture constraints;
- limitations.

### Credential requirement

- stable package-local placeholder ID;
- route/authentication kind;
- optional user-facing label;
- whether username and secret must be re-entered;
- no source local credential ID, account key, username, or secret unless a separately reviewed explicit metadata policy allows a non-secret label.

The importer creates new local references after explicit resolution. Source record IDs are never treated as portable identifiers.

## Logical record: ProfileTemplate

A Profile template must contain:

- schema version;
- template ID and name;
- display defaults, group, notes, and tags;
- Provider-compatible fingerprint settings without a reusable seed;
- optional non-secret route defaults under existing validation;
- dependency requirements;
- creation/update timestamps;
- limitations.

Applying a template always creates:

- a new Profile ID;
- a new managed user-data directory;
- a new fingerprint seed;
- no inherited Evidence;
- unresolved local dependencies when exact automatic resolution is not safe.

A template cannot be used to clone an existing browser user-data directory or credential.

## Identity-transfer modes

### New identity

- default mode;
- destination receives a new Profile ID, directory, and seed;
- no source Evidence is applicable;
- dependency resolution and current validation are required.

### Preserve identity

- explicit advanced mode;
- may preserve the fingerprint seed and identity-relevant configuration;
- destination still receives a new local Profile ID and managed path;
- prior Evidence references are not imported and current local Evidence is stale/absent;
- the UI must warn against simultaneous use of the same identity on multiple devices or Profiles;
- exact dependencies must be remapped and revalidated;
- Provider trust cannot be copied from source metadata;
- operation history records the explicit choice without storing secrets.

Phase 5 does not promise that opaque browser user data is safely portable under this mode.

## Snapshot and restore transaction rules

1. The source browser and conflicting operations are stopped or the operation is rejected.
2. Preflight validates managed roots, lifecycle state, dependencies, file bounds, destination space, and cancellation context.
3. A private staging location is created inside an approved managed root or destination boundary.
4. Content is copied without following links and is verified against the manifest/tree identity.
5. Metadata and content are durably synchronized where the platform supports it.
6. The completed snapshot or restored Profile is activated atomically where possible.
7. The previous healthy state remains in quarantine until the new state and metadata are verified.
8. Persistence failure rolls back activation.
9. Cleanup failure is reported and does not change a successful data result into silent deletion.
10. Startup reconciliation identifies unfinished stages and offers safe completion, rollback, or manual recovery.

Restore never overwrites an existing Profile by ID without an explicit, separately confirmed replacement operation. Default restore creates a new local Profile record.

## Archive, trash, and deletion rules

### Archive

- retains Profile metadata and managed browser data;
- blocks launch until unarchived;
- does not alter Kernel, adapter, credential, or Evidence records;
- is reversible.

### Trash

- moves managed Profile data to private quarantine/trash before deleting or changing authoritative metadata;
- records retention and original identity;
- blocks launch and ordinary editing;
- is recoverable until permanent deletion;
- persistence failure rolls back the move.

### Permanent delete

- requires explicit confirmation and stopped runtime;
- verifies the target remains inside the managed trash boundary;
- never follows links or deletes shared Kernel, adapter, credential, Evidence, or unrelated data;
- records a bounded result without browser contents;
- partial deletion becomes `recovery-required`, not success;
- no automatic orphan scan may trigger permanent deletion.

## Dependency remapping and launchability

An imported or restored Profile becomes launchable only after:

1. lifecycle state is `available`;
2. a current local Kernel is resolved and verified;
3. any required adapter is resolved and verified;
4. any required credential has been explicitly created/selected in the local OS vault;
5. route validation passes;
6. fingerprint Provider/capability validation passes;
7. identity/window consistency preflight passes;
8. managed user-data path validation passes;
9. no conflicting lifecycle operation is active.

Health and Evidence may remain `unknown` or stale until current local Evidence is collected. Launch policy continues to follow the frozen Provider and consistency contracts rather than import metadata.

## Evidence and compatibility rules

- browser and Network Evidence are independently stored and excluded from Phase 5 Profile artifacts;
- snapshot/restore does not rewrite Evidence records to a new Profile ID;
- any change to Profile ID, Provider, binary/package identity, OS, architecture, route identity, fingerprint input, consistency input, or Evidence harness applicability makes prior Evidence non-applicable;
- preserving a fingerprint identity does not preserve Evidence validity;
- templates never inherit Evidence;
- imported reviewed Provider metadata is only a dependency requirement, not a reviewed local result;
- compatibility is regenerated from current local contracts and accepted Evidence.

## Multi-profile operation rules

- the selected Profile set is fixed by the accepted preflight revision;
- Profiles added or changed after preflight are skipped or cause a conflict result, not silently included;
- operations use bounded concurrency;
- destructive operations are serialized per Profile;
- bulk start and scheduled operations are prohibited in Phase 5;
- stop operations may run concurrently only within a reviewed bound;
- cancellation prevents new items from starting and stops active items only at safe checkpoints;
- each item persists its own result;
- aggregate status is derived from item results;
- retries use the same operation ID or explicit predecessor reference and remain idempotent.

## Migration and compatibility rules

1. New schemas are versioned independently: lifecycle, operation, snapshot manifest, portable definition, and template.
2. Existing `profiles.json` records remain readable.
3. Compatibility mapping does not silently write new records during read-only inspection.
4. Migration is explicit, idempotent, and produces a report.
5. Original records and directories remain recoverable until new records are durably validated.
6. Unknown required fields fail closed.
7. Unknown optional fields are preserved when safe or reported as limitations.
8. Downgrade behavior is documented before schema changes merge.
9. Import never replaces an existing Profile by default.
10. Conservative loss of launchability is acceptable when dependencies cannot be proven; optimistic trust promotion is forbidden.

## Cancellation, interruption, and recovery

- every long operation accepts cancellation;
- cancellation is checked before opening the next file/item and before activation;
- cancellation after activation begins follows an explicit commit/rollback boundary;
- abrupt process exit leaves a durable journal and private staging/quarantine state;
- startup reconciliation never guesses that an unverified staging directory is complete;
- stale operation locks require owner/process/session checks and a recovery report;
- repeated reconciliation is idempotent;
- recovery actions are bounded to managed paths and the recorded operation identity;
- unresolved ambiguity becomes `recovery-required`.

## Security and privacy boundaries

- artifact creation and parsing occur locally;
- artifacts use private permissions by default;
- no plaintext-secret fallback is introduced;
- no browser data is parsed for Cookie, LocalStorage, IndexedDB, history, extension, or token extraction;
- proxy routes must already pass inline-secret rejection before export;
- manifest and payload sizes, fields, lists, strings, counts, paths, and compression ratios are bounded;
- parsers reject traversal, absolute paths, canonical collisions, links, special files, malformed encodings, duplicate required records, and trailing unparsed data;
- errors and operation reports are redacted;
- artifacts are not automatically uploaded, synchronized, or sent to external services;
- Kernel and adapter binaries are never bundled by a Profile artifact;
- third-party licenses and provenance remain governed by their independent managed registries.

## Platform contract

- portable definitions and templates use platform-neutral logical fields but may include platform constraints;
- Windows and Linux are required implementation/CI targets before Phase 5 can close;
- local full snapshots are claimed only on platforms with real filesystem integration tests;
- Windows reparse/junction behavior and Linux symlink/mount behavior require platform-specific escape tests;
- macOS remains unsupported until keychain, filesystem, and Wails lifecycle behavior are tested;
- reviewed Chromium support remains the exact Windows amd64 Provider defined in Phase 4;
- lifecycle support on another platform does not grant reviewed browser support there.

## Resource bounds

Exact numeric values may be selected during implementation, but contracts require explicit limits for:

- Profiles per operation;
- concurrent item workers;
- files per snapshot/restore;
- total and per-file bytes;
- path and filename lengths;
- manifest and report size;
- compression/extraction ratio;
- operation duration and inactivity timeout;
- operation history count/retention;
- trash and snapshot retention;
- startup reconciliation work;
- UI progress update rate.

Limits must produce actionable failures and cannot silently truncate a supposedly complete backup.

## Milestone ownership

### M5.1 owns

- lifecycle states;
- lifecycle records;
- operation journal and per-item results;
- operation locking/conflict rules;
- storage inventory;
- cancellation and startup reconciliation foundation;
- compatibility mapping for existing Profiles.

### M5.2 owns

- local full snapshot manifest and tree identity;
- local snapshot/restore transaction;
- archive, trash, restore, retention, and permanent deletion;
- disk-space and filesystem safety;
- rollback and failure-path evidence.

### M5.3 owns

- portable Profile definition;
- dependency requirements and remapping;
- new-identity and preserve-identity modes;
- templates;
- secret/path/binary exclusion evidence;
- import compatibility and Evidence invalidation.

### M5.4 owns

- bounded multi-profile selection and preflight;
- bulk metadata/lifecycle/export/health operations;
- storage-management UI and repair plans;
- per-item progress, cancellation, partial results, and bounded history;
- explicit separation from scheduled or general automation.

## Activation requirement

These contracts remain proposed until the Phase 5 planning packet is reviewed. A separate activation decision must approve them, set Phase 5 to `Active`, permit product implementation, and identify one M5.1 implementation issue.