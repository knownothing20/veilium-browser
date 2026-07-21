# M5.2 Desktop Local Recovery

Status: Stage 5 implementation
Phase: Phase 5
Milestone: M5.2 — Safe Local Recovery
Authority: Issue #54

## Purpose

The Desktop/Wails layer exposes the already reviewed M5.2 local snapshot, restore, archive, recoverable-trash, restore-trash, and explicit permanent-delete transactions without introducing another task system or a general filesystem browser.

The UI is a minimum operational surface. The authoritative state remains the M5.1 lifecycle records and journal plus the M5.2 snapshot and trash catalogs.

## Desktop service boundary

`internal/desktop/local_recovery_service.go` owns the Desktop-facing orchestration boundary.

It:

- lazily opens the versioned local snapshot and trash catalogs under the application data root;
- runs conservative trash reconciliation when the Desktop recovery service starts;
- exposes bounded Profile preflight from lifecycle state, active-session state, operation locks, managed-storage inventory, and matching trash records;
- builds secret-free snapshot definitions and non-secret dependency requirements;
- invokes the Stage 2–4 executors with the existing M5.1 coordinator, journal, locks, cancellation, and recovery records;
- reports snapshot and restore file/byte progress without creating a parallel journal;
- reloads catalog state after operations and preserves recovery-required outcomes;
- keeps every action single-Profile and local-only.

## Wails methods

The Wails application exposes:

- `LocalRecoveryState`
- `LocalRecoveryPreflight`
- `ListLocalSnapshots`
- `GetLocalSnapshot`
- `ListLocalTrash`
- `RefreshLocalRecovery`
- `CreateLocalSnapshot`
- `RestoreLocalSnapshot`
- `ArchiveProfile`
- `UnarchiveProfile`
- `TrashProfile`
- `RestoreTrashedProfile`
- `PermanentlyDeleteTrashedProfile`
- `CancelLocalRecoveryOperation`

Long-running local file operations use bounded Desktop contexts. Cancellation requests still take effect only at safe stages declared by the underlying executor.

The legacy Wails `DeleteProfile` action now routes one eligible stopped Profile through recoverable trash with a visible retention deadline. The lower-level direct metadata deletion path remains fail-closed.

## Preflight contract

Preflight reports:

- current lifecycle state;
- managed-storage inventory status;
- active browser state;
- lifecycle lock state;
- whether snapshot, archive, unarchive, trash, restore-trash, and permanent delete are currently allowed;
- matching Trash ID and retention deadline when present;
- bounded reason codes for blockers.

Preflight is advisory for UI availability. Every executor independently revalidates the same safety and integrity requirements immediately before mutation.

## Minimum UI

The Local recovery workspace extends the existing desktop design and provides:

- snapshot, trash, running-operation, and recovery-finding summaries;
- Profile-level snapshot, archive, unarchive, and recoverable-trash actions;
- verified snapshot listing with restore-to-new-identity;
- recoverable trash listing with restore and permanent-delete controls;
- exact Profile-ID confirmation before irreversible deletion;
- operation stage, file/byte progress, history, and safe-cancellation availability;
- conservative startup reconciliation findings;
- a disabled browser-preview state when Wails is unavailable.

The UI polls bounded local state while open. It does not browse arbitrary directories, expose source filesystem paths as selectable destinations, or infer which interrupted copy is authoritative.

## Integrity and privacy

- Snapshot Profile definitions remove managed paths, local Kernel IDs and executable paths, adapter IDs, credential IDs, and fingerprint seeds.
- Credential values remain exclusively in the operating-system vault.
- Browser files remain opaque bytes.
- Restore defaults to a new Profile ID, managed directory, and fingerprint seed.
- Active sessions, lifecycle locks, unsafe or contradictory storage, unsupported states, and unverified records fail closed.
- Retention expiry never authorizes deletion.
- Permanent deletion requires exact Profile-ID confirmation and the Stage 4 verified trash boundary.
- Desktop state cannot create Provider trust, compatibility, health, or applicable Evidence.

## Explicit non-scope

Stage 5 does not add:

- portable or cross-machine Profile transfer;
- identity-preserving transfer;
- templates;
- multi-Profile batch actions;
- automatic retention cleanup;
- a general filesystem browser;
- remote APIs, MCP, cloud sync, or automation;
- Cookie, extension, secret, Kernel, adapter, Provider, fingerprint, compatibility, or Evidence expansion;
- macOS support claims.

## Validation

Stage 5 requires:

- Desktop service tests for preflight, archive/unarchive, snapshot listing, recoverable trash/restore, and irreversible confirmation failure;
- frontend typecheck, unit tests, and production build;
- Go formatting, vet, unit/race tests, and builds;
- Windows and Linux Wails builds;
- the retained Phase 4, M5.1, official-adapter, and reviewed-Chromium matrices;
- final confirmation that no M5.3 or M5.4 feature crossed the boundary.
