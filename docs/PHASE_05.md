# Phase 5 — Profile Lifecycle and Day-to-Day Operations

Status: Active
Phase: Phase 5
Owner decision required: No
Product implementation allowed: Yes

## Activation record

The project owner approved Phase 5 on 2026-07-20 through Activation Review Issue #40.

The approved planning baseline is:

- Planning Issue #37;
- Planning PR #39, merged as `531f56d49cebc79cf6aee7a24d8f972d6275ce6b`;
- `docs/PHASE_05_CONTRACTS.md`;
- Windows reviewed-Chromium CI reliability Hotfix #43, merged as `6d4b04a9668c87cc110a4c0d423909d45649b529`;
- Activation Review Issue #40;
- the documentation-only activation pull request that changes this document to `Active`.

Exactly one implementation issue is authorized at activation:

- Issue #45 — **M5.1 Lifecycle Contract, Inventory, and Operation Journal**.

`Product implementation allowed: Yes` does not authorize arbitrary Phase 5 work. Product changes must remain within the single current issue named by `docs/STATUS.md`. M5.2, M5.3, and M5.4 remain blocked until the preceding milestone closes through a dedicated review and STATUS advances.

## User outcome

A user can preserve, recover, move, template, archive, and manage Veilium Profiles without silently copying secrets, weakening Provider trust, reusing an identity unintentionally, or losing browser data when an operation is interrupted.

Phase 5 turns the existing Profile registry and managed browser directories into an explicit lifecycle system. The priority is recoverability and truthful portability, not the largest possible feature set.

## Milestone order

Phase 5 must be delivered in this dependency order:

1. **M5.1 — Lifecycle Contract, Inventory, and Operation Journal**
2. **M5.2 — Safe Local Snapshot, Restore, Archive, and Trash**
3. **M5.3 — Portable Profile Definitions and Templates**
4. **M5.4 — Bounded Multi-profile Operations and Storage Management**

A later milestone must not begin before the current milestone is merged, reviewed against its acceptance criteria, and handed off through `docs/STATUS.md`.

## Why this order

The product separates several kinds of state:

- `profiles.json` contains Profile metadata and local dependency references;
- every Profile has a Veilium-managed browser user-data directory;
- proxy secrets live only in the operating-system credential vault;
- Chromium kernels and proxy adapters have independent registries and integrity identities;
- browser and Network Evidence are stored independently from the Profile;
- runtime logs, installer staging, and temporary adapter/browser state are operational data rather than Profile identity.

A safe backup or migration feature cannot treat all application state as one portable directory. Phase 5 first establishes lifecycle and recovery records, then same-machine recovery, then portable definitions, and finally bounded multi-Profile operations.

## M5.1 — Lifecycle Contract, Inventory, and Operation Journal

### Goal

Create the versioned policy and durable operation foundation required by every later lifecycle feature.

### Current authority

M5.1 is the only active implementation milestone and is governed by Issue #45.

### Required behavior

- define versioned Profile lifecycle states with at least `available`, `draft`, `archived`, and `trashed`;
- keep lifecycle state separate from runtime session state and derived Profile health;
- add a versioned operation journal with stable IDs, stages, timestamps, per-item results, cancellation state, limitations, and recovery actions;
- prevent conflicting operations from owning the same Profile simultaneously;
- make duplicate or idempotent requests deterministic;
- block destructive or storage-affecting lifecycle work while the Profile browser or a protected dependent operation is active;
- inventory managed Profile directories and report present, missing, unexpected, orphaned, or unsafe state without deleting anything;
- calculate bounded file-count and byte-size summaries where safe;
- reconcile interrupted journal, staging, quarantine, and stale-lock state at startup;
- retain existing Profile records and directories until replacement state is durably written and validated;
- expose truthful lifecycle, operation, inventory, cancellation, and recovery status through the existing desktop/API boundaries.

### Acceptance

- existing Profiles remain readable through conservative, versioned compatibility;
- lifecycle, runtime, health, operation, and inventory states remain distinct;
- interrupted or cancelled operations cannot become successful implicitly;
- stale locks and startup interruption produce actionable recovery results;
- inventory reports missing, orphaned, unsafe, symlinked, reparse-escaped, special, malformed, duplicate, oversized, and unsupported state without automatic deletion;
- Windows and Linux filesystem tests cover every claimed M5.1 path;
- secrets, cookies, browser storage contents, history, arbitrary page data, and Evidence payloads are not inspected or persisted;
- Phase 4 Provider, binary, capability, compatibility, Evidence, and fail-closed regressions remain green.

### M5.1 non-scope

Issue #45 must not implement:

- snapshot containers or restore execution;
- archive/trash directory movement or permanent deletion;
- portable export/import;
- templates;
- multi-Profile batch UI or operations;
- Cookie or extension management;
- automation, MCP, sync, proxy rotation, or release work.

## M5.2 — Safe Local Snapshot, Restore, Archive, and Trash

### Goal

Protect users from accidental loss and make same-machine Profile recovery reliable before claiming broader portability.

### Local full snapshot boundary

A local full snapshot may contain:

- the selected Profile definition;
- the Profile's Veilium-managed browser user-data directory as opaque files;
- a versioned manifest, deterministic file-tree identity, file count, byte size, source platform, source schema/application versions, and dependency requirements.

It must not contain:

- resolved credential secrets or operating-system keyring contents;
- Chromium kernel or proxy-adapter binaries;
- temporary runtime files or private runtime logs;
- browser or Network Evidence unless a separately reviewed diagnostics-export contract is approved.

### Required behavior

- require the browser and dependent operations to be stopped;
- reject traversal, links, junction/reparse escapes, devices, sockets, pipes, special files, duplicate paths, and files outside the managed root;
- enforce bounded file count, total bytes, individual-file size, destination space, and operation duration;
- write and restore through private staging, complete verification, atomic activation, and rollback;
- never overwrite the only healthy Profile copy before a verified replacement exists;
- treat full browser-data snapshots as same-user/same-machine recovery unless a later tested platform review proves more;
- implement explicit archive, recoverable trash, restore-from-trash, retention, and permanent-delete workflows;
- never silently remove orphaned or failed-restore directories.

M5.2 remains blocked until M5.1 closes and STATUS explicitly advances.

## M5.3 — Portable Profile Definitions and Templates

### Goal

Move configuration safely and create repeatable Profiles without pretending that machine-bound browser state or secrets are portable.

### Portable definition package

A portable definition may include:

- display metadata, group, notes, and tags;
- fingerprint configuration under explicit identity-transfer rules;
- validated non-secret route configuration;
- Kernel Provider/version/identity requirements;
- proxy-adapter kind/version/identity requirements;
- unresolved credential requirements without secret values;
- source schema/application versions, limitations, and compatibility notes.

It must omit or replace:

- local Profile IDs and absolute `userDataDir` paths;
- local Kernel, adapter, and credential record IDs;
- executable paths and managed binaries;
- operating-system vault secrets;
- browser user-data files;
- runtime sessions, logs, temporary files, and Evidence.

### Identity modes

- **New identity** is the default and creates a new Profile ID, managed directory, and fingerprint seed.
- **Preserve identity** is explicit, advanced, warns against simultaneous use, requires dependency remapping and current validation, and makes prior Evidence non-applicable.
- imported definitions remain `draft` until every dependency and current validation requirement passes.
- templates always create a new Profile ID, directory, and fingerprint seed.

M5.3 remains blocked until M5.2 closes and STATUS explicitly advances.

## M5.4 — Bounded Multi-profile Operations and Storage Management

### Goal

Reduce repetitive work while keeping operations reviewable, cancellable, bounded, and separate from general automation.

### Approved operation classes

- add or remove tags;
- move Profiles between groups;
- archive, unarchive, trash, or restore selected Profiles;
- export portable definitions for selected Profiles;
- run bounded dependency-integrity and Profile-health refreshes;
- stop selected running Profiles with explicit confirmation;
- inspect Profile, snapshot, trash, Evidence, and log storage usage;
- identify orphaned or stale managed data and generate a non-destructive repair plan.

### Required behavior

- show the selected set and preflight before execution;
- use bounded concurrency;
- expose cancellation only at safe checkpoints;
- return truthful per-item success, partial, skipped, cancelled, or failed results;
- never report a batch successful when a required item failed;
- retain bounded local operation history without secrets or browser contents;
- require stronger confirmation for permanent deletion and identity-preserving transfer.

M5.4 remains blocked until M5.3 closes and STATUS explicitly advances.

## Data classification

### Portable by default

- Profile display metadata;
- group, notes, and tags;
- fingerprint settings under explicit identity-transfer rules;
- versioned non-secret dependency requirements;
- validated non-secret route configuration;
- template definitions.

### Local or machine-bound

- browser user-data directories and browser-encrypted state;
- operating-system credential-vault secrets;
- local Kernel/adapter record IDs, executable paths, and managed binaries;
- runtime sessions, temporary files, and process logs;
- local Profile directory paths.

### Provider- and Evidence-bound

- reviewed Provider and binary/package identity;
- capability claims;
- browser and Network Evidence;
- compatibility and health results derived from current dependencies and configuration.

### Excluded from portable artifacts

- plaintext or decrypted secrets;
- cookies or browser storage as editable records;
- browsing history or arbitrary page contents;
- screenshots, downloads, runtime logs, and generated private adapter configuration;
- release assets or updater state.

## Compatibility and migration policy

- every new persisted record and artifact is schema-versioned;
- migration is explicit, idempotent, conservative, and rollback-aware;
- original Profile metadata and managed directories remain recoverable until replacement state is validated;
- unknown required fields and unsupported future schemas fail closed;
- unknown optional fields are preserved when safe;
- imports do not rewrite existing records in place by default;
- dependency remapping creates current local references instead of copying source record IDs;
- a Profile may become more conservative, but never more trusted without current local verification and applicable Evidence;
- Evidence applicability is recomputed after identity, Provider, binary, route, platform, or relevant configuration changes.

Detailed logical records and state-transition rules are authoritative in `docs/PHASE_05_CONTRACTS.md`.

## Security, privacy, and licensing

- lifecycle APIs remain local and use existing authenticated desktop boundaries;
- lifecycle stores use private permissions, atomic persistence, strict decoding, bounded records, and safe path handling;
- secrets remain in the operating-system vault and are never silently exported;
- lifecycle operations do not inspect browser contents unrelated to their bounded file/storage contract;
- snapshot/import parsers must reject traversal, links, special files, duplicate paths, decompression bombs, and oversized manifests;
- artifacts do not bundle Chromium, Xray, or sing-box or bypass their license/provenance rules;
- restore/import cannot broaden the exact reviewed Chromium Provider claim;
- logs contain bounded IDs, stages, sizes, and redacted errors—not credentials or browser contents;
- permanent deletion is explicit and never automatic orphan cleanup;
- no unauthenticated remote control, generic filesystem browser, public Launch API, or MCP is introduced.

## Platform policy

- logical contracts and portable definitions are platform-neutral;
- Windows and Linux are the minimum implementation and CI targets for Phase 5 lifecycle behavior;
- local snapshot/restore requires real filesystem integration tests on every claimed platform;
- the reviewed browser Provider remains Windows amd64 only;
- Linux lifecycle support does not create a reviewed Linux browser Provider;
- macOS remains unclaimed until a real tested lifecycle path and keychain/filesystem analysis exist;
- cross-platform full browser-state migration is not claimed in Phase 5.

## Explicit non-goals

Phase 5 does not authorize:

- Cookie import, export, or editing as a separate feature;
- extension package management;
- complete cross-machine portability of Chromium user data;
- export or synchronization of operating-system vault secrets;
- bundling or updating browser kernels and proxy adapters inside Profile packages;
- a second reviewed browser Provider, revision, platform, or architecture;
- new fingerprint controls or optimistic compatibility claims;
- proxy protocol expansion, pools, rotation, scheduling, or account farming;
- bulk Profile start or general browser automation;
- public API, CDP gateway, MCP, cloud sync, or release/update work;
- broad UI redesign unrelated to lifecycle operations.

## Required validation matrix

Every implementation milestone must include the strongest applicable subset of:

- governance and documentation checks;
- formatting, static analysis, unit tests, race tests, and builds;
- frontend typecheck, tests, and production build;
- Windows and Linux Wails builds;
- schema compatibility and golden fixtures;
- unsafe path/link/special-file and size-bound tests;
- cancellation, interruption, disk-full, persistence-failure, and rollback tests;
- active-session and conflicting-operation tests;
- cross-platform compatibility tests for claimed portable artifacts;
- real filesystem integration tests on every claimed platform;
- secret-scanning tests for portable artifacts;
- Provider, binary identity, Evidence freshness, dependency-remapping, and tamper regressions;
- long-running and large-directory tests with bounded resources.

## Phase exit criteria

Phase 5 may close only when:

- lifecycle states and operation records are versioned and recoverable;
- interrupted operations reconcile without silent deletion or optimistic success;
- local snapshots, restore, archive, trash, and permanent deletion pass rollback and integrity tests;
- portable definitions and templates exclude secrets and machine-local paths;
- imports require explicit dependency remapping and current validation;
- identity-preserving transfer invalidates stale Evidence and remains explicit;
- bounded multi-Profile operations return truthful per-item results and support cancellation;
- storage inventory and orphan handling are reviewable and non-destructive by default;
- Windows and Linux claimed paths pass the approved matrix;
- Phase 4 Provider, capability, Evidence, privacy, compatibility, and fail-closed boundaries remain intact;
- unresolved risks and later-phase deferrals are recorded;
- a dedicated Phase 5 Closing Review marks the phase `Done`.

## Implementation control

The current implementation authority is Issue #45 only.

Each milestone requires:

1. a short-lived implementation branch;
2. one Draft PR with STATUS updated in the same change set when product code changes;
3. Governance and the complete applicable CI matrix;
4. a milestone closing review before STATUS advances;
5. no work from a later milestone before explicit handoff.
