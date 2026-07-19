# Phase 5 — Profile Lifecycle and Day-to-Day Operations

Status: Planning
Phase: Phase 5
Owner decision required: Yes
Product implementation allowed: No

## User outcome

A user can preserve, recover, move, template, archive, and manage Veilium profiles without silently copying secrets, weakening Provider trust, reusing an identity unintentionally, or losing browser data when an operation is interrupted.

Phase 5 turns the existing profile registry and managed browser directories into an explicit lifecycle system. The priority is recoverability and truthful portability, not adding the largest possible set of profile-management features.

## Planning decision

Phase 5 will be delivered in this dependency order:

1. **M5.1 — Lifecycle Contract, Inventory, and Operation Journal**
2. **M5.2 — Safe Local Snapshot, Restore, Archive, and Trash**
3. **M5.3 — Portable Profile Definitions and Templates**
4. **M5.4 — Bounded Multi-profile Operations and Storage Management**

A separate activation decision must approve this plan and change Phase 5 to `Active` before any product code is implemented.

## Why this order

The current product already separates several kinds of state:

- `profiles.json` contains Profile metadata and local dependency references;
- each Profile has a Veilium-managed browser user-data directory;
- proxy secrets live only in the operating-system credential vault;
- Chromium kernels and proxy adapters have independent managed registries and integrity identities;
- browser and Network Evidence are stored independently from the Profile;
- runtime logs and temporary adapter/browser state are operational data, not Profile identity.

A safe backup or migration feature cannot treat all of those records as one portable folder. Phase 5 first defines lifecycle and recovery contracts, then implements same-machine recovery, and only then introduces portable definitions and bounded bulk operations.

## M5.1 — Lifecycle Contract, Inventory, and Operation Journal

### Goal

Create the versioned policy and durable operation foundation required by every later lifecycle feature.

### Required behavior

- define Profile lifecycle states with at least `available`, `draft`, `archived`, and `trashed`;
- keep lifecycle state distinct from runtime session state and derived Profile health;
- introduce a versioned operation record for snapshot, restore, archive, trash, permanent delete, export, import, template, and bulk actions;
- record operation type, selected Profiles, start/completion time, current stage, per-item result, cancellation state, limitations, and recovery action;
- prevent conflicting operations for the same Profile;
- block destructive lifecycle operations while a Profile browser or dependent evidence operation is active;
- inventory managed Profile directories and report missing, unexpected, or orphaned data without silently deleting it;
- calculate bounded storage usage and verify sufficient destination space before large copy or restore operations;
- recover or report interrupted staging, quarantine, and journal records on application startup;
- retain existing Profile records and directories until the replacement state is durably written and validated;
- provide dry-run/preflight output before any destructive or large operation.

### Acceptance

- existing Profiles remain readable without optimistic migration;
- an interrupted operation is restart-safe and produces an actionable recovery result;
- no lifecycle state can grant Provider trust, capability support, or Evidence validity;
- tests cover duplicate operations, active-session blocking, cancellation, persistence failure, stale locks, orphan detection, and startup reconciliation.

## M5.2 — Safe Local Snapshot, Restore, Archive, and Trash

### Goal

Protect users from accidental loss and make same-machine Profile recovery reliable before claiming cross-device portability.

### Local full snapshot boundary

A local full snapshot may contain:

- the selected Profile definition;
- the Profile's Veilium-managed browser user-data directory as opaque files;
- a versioned manifest, file-tree identity, file count, byte size, source platform, source application/schema versions, and dependency requirements.

It must not contain:

- resolved credential secrets;
- operating-system keyring contents;
- Chromium kernel or proxy-adapter binaries;
- temporary browser or adapter runtime files;
- private runtime logs;
- browser or Network Evidence unless a later explicit diagnostics-export contract is approved.

### Required behavior

- require the browser session to be stopped and the managed Profile directory to be stable;
- reject symlinks, junction/reparse escapes where applicable, devices, sockets, pipes, non-regular files, path traversal, duplicate paths, and files outside the managed root;
- enforce bounded file count, total bytes, individual-file size, destination space, and operation duration;
- write snapshots through private staging and publish them only after complete verification;
- verify a deterministic file-tree identity before accepting a snapshot or restore;
- restore into staging, validate dependencies and Profile metadata, then atomically activate or roll back;
- never overwrite the only healthy Profile copy before a verified replacement exists;
- classify full browser-data snapshots as same-user/same-machine recovery unless a future platform-specific migration review proves broader portability;
- add explicit archive, recoverable trash, restore-from-trash, retention, and permanent-delete workflows;
- quarantine managed Profile data before metadata deletion and roll back the move if persistence fails;
- never silently remove an orphaned or failed-restore directory.

### Acceptance

- local snapshot and restore preserve the selected Profile definition and complete managed directory byte identity;
- cancellation and simulated disk/persistence failure leave the original Profile usable;
- restore rejects modified manifests, missing files, added files, unsafe entries, dependency mismatch, and unsupported platform claims;
- trash and restore are recoverable and do not affect unrelated Profiles, kernels, adapters, credentials, or Evidence.

## M5.3 — Portable Profile Definitions and Templates

### Goal

Allow users to move configuration safely and create repeatable Profiles without pretending that machine-bound browser state or secrets are portable.

### Portable definition package

A portable definition may include:

- display metadata, group, notes, and tags;
- fingerprint configuration;
- a declared identity-transfer mode;
- non-secret proxy route configuration that already passes existing inline-secret validation;
- required Kernel Provider/version/identity descriptors;
- required proxy-adapter kind/version/identity descriptors;
- unresolved credential requirements without the secret value;
- source schema and application versions, limitations, and compatibility notes.

It must omit or replace:

- local Profile ID and absolute `userDataDir`;
- local kernel, adapter, and credential record IDs;
- executable paths and managed binary contents;
- operating-system vault secrets;
- browser user-data files;
- runtime sessions, logs, temporary files, and Evidence records.

### Import modes

- **New identity** is the default. It creates a new Profile ID, managed directory, and fingerprint seed.
- **Preserve identity** is an explicit advanced choice. It may retain the fingerprint seed and identity settings, but it must mark prior Evidence stale, require dependency remapping and revalidation, warn against simultaneous use of the same identity, and never inherit reviewed claims from the source machine.
- Imported definitions enter `draft` state until every required dependency is mapped and all current validation passes.

### Templates

- templates contain reusable non-secret configuration only;
- applying a template always creates a new Profile ID, new managed directory, and new fingerprint seed;
- templates cannot contain credential IDs, secrets, browser data, local executable paths, Evidence, or runtime state;
- Provider-specific settings remain constrained by the selected current Provider Contract.

### Acceptance

- exported packages contain no secret material or local absolute paths;
- imports are deterministic, versioned, bounded, and fail closed on unknown required fields or incompatible dependencies;
- no imported or templated Profile becomes launchable until Kernel, adapter, credential, fingerprint, consistency, and route validation pass;
- custom and legacy Providers remain unpromoted;
- preserving an identity invalidates applicability of old local Evidence until new Evidence is collected.

## M5.4 — Bounded Multi-profile Operations and Storage Management

### Goal

Reduce repetitive manual work while keeping operations reviewable, cancellable, and separate from general automation.

### Approved operation classes

- add or remove tags;
- move Profiles between groups;
- archive, unarchive, trash, or restore selected Profiles;
- export portable definitions for selected Profiles;
- run bounded dependency-integrity and Profile-health refreshes;
- stop selected running Profiles with explicit confirmation;
- inspect per-Profile browser-data, snapshot, trash, Evidence, and log storage usage;
- identify orphaned or stale managed data and generate a repair plan.

### Required behavior

- show the exact selected set and preflight before execution;
- use bounded concurrency and avoid starting many browser or network processes implicitly;
- expose cancellation at safe checkpoints;
- return per-item success, partial, skipped, cancelled, or failed results;
- never report a batch as successful when any required item failed;
- preserve successful independent items while making partial results explicit;
- retain a bounded local operation history without secrets or browser contents;
- require stronger confirmation for permanent deletion or identity-preserving export/import.

### Explicitly deferred

- bulk Profile start;
- scheduled actions;
- proxy rotation or account farming;
- general browser automation;
- public Launch API, unified CDP gateway, or MCP;
- cloud synchronization.

Those belong to later controlled-automation planning and do not gain authority from Phase 5.

## Data classification

### Portable by default

- Profile display metadata;
- group, notes, and tags;
- fingerprint settings under explicit identity-transfer rules;
- versioned non-secret dependency requirements;
- non-secret route configuration that passes existing validation;
- template definitions.

### Local or machine-bound

- browser user-data directories and browser-encrypted state;
- operating-system credential-vault secrets;
- local kernel and adapter record IDs, executable paths, and managed binaries;
- runtime sessions, temporary files, and process logs;
- local Profile directory paths.

### Provider- and Evidence-bound

- reviewed Provider identity and binary/package identity;
- capability claims;
- browser and Network Evidence;
- compatibility and health results derived from current dependencies and configuration.

### Excluded from Phase 5 portable artifacts

- plaintext or decrypted secrets;
- cookies or browser storage exported as separately editable records;
- arbitrary browsing history or page contents;
- screenshots, downloads, runtime logs, and generated private adapter configuration;
- application release assets or updater state.

## Compatibility and migration policy

- all new persisted records and artifacts are schema-versioned;
- migration is explicit, idempotent, and conservative;
- original Profile metadata and managed directories remain recoverable until replacement state is validated;
- unknown required fields or unsupported future schemas fail closed;
- unknown optional fields are preserved when safe rather than silently discarded;
- imports never rewrite existing records in place by default;
- downgrade behavior is documented before schema changes merge;
- a Profile may become more conservative after import or migration, but never more trusted without current local verification and Evidence;
- dependency remapping creates current local references and does not copy source record IDs;
- Evidence applicability is recomputed after any identity, Provider, binary, route, platform, or relevant configuration change.

## Security, privacy, and licensing

- lifecycle APIs remain local and use existing authenticated/desktop boundaries;
- snapshot and import parsers are bounded and reject traversal, links, special files, duplicate paths, decompression bombs, and oversized manifests;
- secrets remain in the operating-system vault and are never silently exported;
- artifacts use private permissions by default and are treated as sensitive local files;
- operations never bundle third-party Chromium, Xray, or sing-box binaries or bypass their licensing/provenance rules;
- restore and import do not broaden the exact reviewed Chromium Provider claim;
- lifecycle logs contain IDs, stages, sizes, and redacted errors only, not browser contents or credentials;
- permanent deletion is explicit and cannot occur as automatic orphan cleanup.

## Platform policy

- logical contracts and portable-definition packages are platform-neutral;
- Windows and Linux are the minimum implementation and CI targets for Phase 5 lifecycle behavior;
- local full-snapshot behavior must be integration-tested on each claimed platform;
- the reviewed browser Provider remains Windows amd64 only;
- Linux lifecycle support does not create a reviewed Linux browser Provider;
- macOS remains unclaimed until a real tested lifecycle path and keychain/filesystem analysis are added;
- cross-platform full browser-state migration is not claimed in Phase 5.

## Explicit non-goals

Phase 5 does not authorize:

- Cookie import, export, or editing as a separate feature;
- extension installation or package management;
- claiming complete cross-machine portability of Chromium user data;
- exporting or synchronizing operating-system vault secrets;
- bundling or updating browser kernels and proxy adapters inside Profile packages;
- a second reviewed browser Provider, revision, platform, or architecture;
- new fingerprint controls or optimistic compatibility claims;
- proxy protocol expansion, proxy pools, rotation, or scheduled health operations;
- general automation, public API, MCP, cloud sync, or release/update work;
- broad UI redesign unrelated to lifecycle operations.

## Required validation matrix

Every implementation milestone must include the strongest applicable subset of:

- governance and documentation checks;
- format, static analysis, unit tests, race tests, and builds;
- frontend typecheck, tests, and production build;
- Windows and Linux Wails builds;
- schema compatibility and golden-fixture tests;
- unsafe archive/path/link/special-file and size-bound tests;
- cancellation, interruption, disk-full, persistence-failure, and rollback tests;
- active-session and conflicting-operation tests;
- cross-platform import/export compatibility tests;
- real filesystem snapshot/restore integration tests on every claimed platform;
- secret-scanning tests proving portable artifacts exclude vault material;
- Provider, binary identity, Evidence freshness, and dependency-remapping regressions;
- long-running and large-directory tests with bounded resources.

## Phase exit criteria

Phase 5 may close only when:

- lifecycle states and operation records are versioned and recoverable;
- interrupted operations reconcile without silent deletion or optimistic success;
- local snapshots, restore, archive, trash, and permanent deletion pass rollback and integrity tests;
- portable definitions and templates exclude secrets and machine-local paths;
- imports require explicit dependency remapping and current validation;
- identity-preserving transfer invalidates stale Evidence and remains explicit;
- bounded multi-profile operations return truthful per-item results and support cancellation;
- storage inventory and orphan handling are reviewable and non-destructive by default;
- Windows and Linux claimed paths pass the approved integration matrix;
- retained Phase 4 Provider, capability, Evidence, privacy, and fail-closed boundaries remain intact;
- unresolved risks and later-phase deferrals are recorded;
- a dedicated Phase 5 Closing Review marks the phase `Done`.

## Activation gate

This document is a planning proposal. Product implementation remains blocked.

To activate Phase 5, a separate reviewed activation decision must:

1. approve the user outcome, milestone order, contracts, platform policy, non-goals, validation matrix, and exit criteria;
2. set Phase 5 status to `Active`;
3. set `Product implementation allowed: Yes`;
4. create or identify exactly one M5.1 implementation issue;
5. update `docs/ROADMAP.md` and `docs/STATUS.md` in the same planning/activation pull request.

Until then, Issue #37 and its documentation-only planning pull request are the only authorized work.