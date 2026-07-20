# Local Recovery Contracts

Status: M5.2 Stage 1 implementation
Phase: Phase 5
Milestone: M5.2 — Safe Local Recovery
Authority: Issue #54

## Stage 1 boundary

Stage 1 defines versioned local recovery records and persistence only. It does not copy, move, restore, activate, archive, or clean browser directories.

The later M5.2 stages must reuse the M5.1 lifecycle operation journal, Profile locks, runtime and dependent-operation blockers, cancellation state, storage inventory, and startup recovery.

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

Catalog creation always begins as `pending`. Supported later transitions are explicit. Stale revisions and changes to immutable identity fields fail with a conflict.

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

Stage 1 records path identities only. Filesystem traversal begins in Stage 2 and must independently reject symlinks, junctions, reparse points, and special entries.

## Bounds

The contract defines explicit limits for:

- manifest and catalog encoded bytes;
- Profile-definition bytes;
- number of catalog records;
- file entries;
- individual file bytes;
- total file bytes;
- identifiers, text, codes, capabilities, paths, and path segments.

Manifest JSON encoding performs validation before materializing the complete output. A conservative entry-size estimate rejects a file-entry set that cannot fit the encoded manifest budget. The final encoded byte limit remains authoritative.

## Deterministic identity

The tree identity is SHA-256 over sorted canonical records containing:

1. relative path;
2. decimal byte size;
3. file SHA-256.

Windows path collision checks use case-insensitive keys while preserving the original canonical path in the manifest.

The Profile-definition digest uses canonical JSON object encoding. The digest represents the non-secret Profile definition only; it cannot grant Provider trust, compatibility, health, or Evidence applicability.

## Dependency requirements

Dependency records contain requirements, not current local record IDs or executable paths.

- reviewed Kernel requirements need an exact binary or package-tree identity;
- official adapter requirements need an exact executable identity;
- Kernel and adapter platforms must match the snapshot source platform;
- credential records contain placeholders and required input classes only;
- resolved credential values remain exclusively in the operating-system vault.

## Required exclusions

Every manifest explicitly excludes:

- credential secrets;
- Kernel binaries;
- adapter binaries;
- runtime state;
- private runtime logs;
- browser and Network Evidence payloads.

Stage 1 does not inspect browser contents such as Cookies, LocalStorage, IndexedDB, browsing history, page data, tokens, or extension data.

## Persistence and rollback

Manifest and catalog persistence:

- uses strict JSON decoding with unknown-field and trailing-data rejection;
- rejects oversized, unsupported-version, malformed, duplicate, contradictory, linked, reparse, or non-regular authoritative files;
- requires private managed directories and files;
- writes through a private temporary file;
- flushes the temporary file before publication;
- publishes immutable manifests without replacing an existing manifest;
- replaces the catalog atomically;
- preserves in-memory and on-disk state when catalog persistence fails;
- uses optimistic revisions for catalog updates.

Unsupported future schemas fail closed and are not rewritten by an older build.

## Stage 1 validation

The Stage 1 suite covers:

- valid manifest and deterministic identity;
- unsafe and colliding path identities;
- contradictory size summaries and artifact scope;
- dependency identity and source-platform contradictions;
- strict JSON round trips and unknown-field rejection;
- immutable manifest publication;
- private file, symlink, and authoritative-file checks;
- catalog creation, transitions, reopen, duplicate rejection, stale revision, future version, and persistence rollback.

Stage 2 remains blocked until the Stage 1 code, documentation, Windows/Linux builds, and retained Phase 4/M5.1 matrix pass.
