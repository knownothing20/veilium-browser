# Current Project Status

Last updated: 2026-07-22
Application version: 0.15.0-dev
Main baseline SHA: ffcf25d94cd821c82f07cc49fc61130d3e02fcdb
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.3 — Portable Profile Definitions and Templates
Current task: Issue #58 — implement M5.3 portable Profile definitions and templates
Current implementation stage: authorized handoff — implementation not yet started

## Operational rule

Phase 5 remains `Active`. Product implementation is limited to Issue #58 until the M5.3 implementation PR merges and M5.3 passes its dedicated Closing Review.

M5.4 remains blocked until that post-merge review passes and a separate documentation-only handoff advances this file.

GitHub Actions are currently paused by the owner. M5.3 development must not create, enable, modify, manually trigger, or rerun workflows. Changes must be developed and reviewed in concentrated batches, with local or offline validation first and no Actions-based iterative debugging.

## Completed milestones

### M5.1 — Lifecycle Contract, Inventory, and Operation Journal

M5.1 is complete:

- implementation Issue #45 closed;
- implementation PR #52 squash-merged as `51c469e51ec4cab4ade99efd83c2e6c26145f266`;
- Closing Review #53 passed and closed.

The M5.1 lifecycle records, operation journal, locks, blockers, cancellation state, per-item results, storage inventory, startup reconciliation, Desktop integration, UI, tests, and documentation remain the authoritative lifecycle foundation.

### M5.2 — Safe Local Recovery

M5.2 is complete:

- implementation Issue #54 closed by PR #56;
- implementation PR #56 squash-merged as `ffcf25d94cd821c82f07cc49fc61130d3e02fcdb`;
- Closing Review #57 passed and closed.

The merged M5.2 baseline provides strict same-machine snapshots, restore to a new limited identity, archive/unarchive, recoverable trash, exact restore-trash, explicit permanent deletion, conservative startup reconciliation, Desktop/Wails recovery APIs, minimum UI, and Windows/Linux safety coverage.

M5.2 remains local recovery only. It does not authorize portable browser data, secret export, templates, or multi-Profile operations.

## Current authority

Issue #58 is the single M5.3 implementation task.

M5.3 implements only portable non-secret Profile definitions, dependency requirements and remapping, explicit identity-transfer modes, and templates as defined by Issue #58 and `docs/PHASE_05_CONTRACTS.md`.

All M5.3 work must:

- reuse the M5.1 lifecycle records, operation journal, locks, blockers, cancellation, item results, and recovery state;
- reuse strict parsing, canonical digest, bounded records, private staging, conflict, idempotency, rollback, and recovery principles established by M5.2;
- keep Profile metadata, browser data, secrets, managed dependencies, Evidence, runtime state, health, Provider trust, and compatibility separate;
- export only validated non-secret configuration;
- exclude source local Profile IDs as destination identity, managed paths, local Kernel/adapter/credential IDs, executable paths, binaries, vault values, browser data, Evidence, runtime files, and generated adapter configuration;
- resolve dependencies only against current verified local records;
- require explicit local credential creation or selection without importing source credentials;
- create a new local Profile ID and managed path for every import;
- use new identity by default with a new fingerprint seed;
- allow preserve-identity behavior only as an explicit advanced choice with warnings, current dependency validation, and no inherited Evidence or trust;
- keep imported Profiles `draft` or explicitly limited until all current local validation passes;
- ensure templates always create a new Profile ID, managed directory, and fingerprint seed;
- never broaden Provider trust, capability support, compatibility, health, or Evidence applicability.

## M5.3 implementation stages

1. **Stage 1 — Contracts and persistence**
   - versioned portable-definition and template schemas;
   - canonical deterministic encoding and payload digest;
   - dependency requirement and identity-transfer records;
   - strict decoding, bounds, transitions, persistence, and downgrade behavior.

2. **Stage 2 — Portable export**
   - export preflight and validated non-secret route handling;
   - exclusion enforcement for secrets, local IDs/paths, binaries, browser data, runtime state, and Evidence;
   - private local artifact publication and bounded operation result.

3. **Stage 3 — Import and dependency remapping**
   - strict artifact parse and tamper detection;
   - current local Kernel/adapter/credential resolution;
   - deterministic new Profile creation without overwrite;
   - draft/limited lifecycle state, idempotency, rollback, cancellation, and interruption recovery.

4. **Stage 4 — Identity-transfer modes**
   - default new-identity behavior;
   - explicit preserve-identity mode with warnings and simultaneous-use risk;
   - no inherited Provider trust, compatibility, health, or Evidence.

5. **Stage 5 — Templates**
   - bounded template create/list/get/update/delete behavior;
   - apply-template transaction with new Profile ID, directory, and seed;
   - no browser data, credential, binary, local-path, or Evidence inheritance.

6. **Stage 6 — Desktop/Wails API and minimum UI**
   - export preflight and artifact creation;
   - import preview, exclusions, limitations, dependency resolution, identity choice, and confirmation;
   - template management and apply flow;
   - progress/history/cancellation projected from the authoritative M5.1 journal;
   - browser preview remains non-operational and no general filesystem browser is introduced.

7. **Stage 7 — Integration, documentation, and owner-review handoff**
   - complete contract, exclusion, tamper, failure-path, platform, Desktop, and frontend review;
   - no workflow or temporary diagnostic artifact;
   - dedicated implementation review before owner merge decision;
   - mandatory post-merge M5.3 Closing Review before M5.4.

## Explicit non-scope

Issue #58 does not authorize:

- portable browser user-data transfer or cross-platform full browser-state claims;
- Cookie, LocalStorage, IndexedDB, history, token, extension, or credential extraction/import/export;
- secret export or plaintext-secret fallback;
- bundling Kernel or adapter binaries;
- automatic artifact upload, cloud sync, remote API, MCP, or general automation;
- multi-Profile batch operations or storage-management repair actions owned by M5.4;
- bulk start, scheduling, proxy rotation, account farming, or unattended identity reuse;
- Provider, Kernel, adapter, fingerprint-capability, proxy-protocol, compatibility, or Evidence expansion;
- macOS support claims;
- release/updater work or unrelated UI redesign.

## Frozen boundaries

Phase 4, M5.1, and M5.2 remain frozen:

- reviewed browser trust remains restricted to the approved Windows amd64 package;
- custom and legacy Providers remain unpromoted;
- imported metadata cannot create reviewed trust, compatibility, health, capability support, or applicable Evidence;
- vault secrets remain local and non-portable;
- browser and Network Evidence remain independently stored and excluded;
- local record IDs and absolute paths are never portable identities;
- unsupported, missing, unsafe, partial, contradictory, and unverifiable state fails closed or remains explicitly limited.

## Required validation

M5.3 requires the strongest applicable offline/local subset of:

- governance and documentation consistency;
- Go formatting, vet, unit/race tests, and builds;
- frontend typecheck, unit tests, and production build;
- strict schema, canonical digest, tamper, duplicate, unknown-field, trailing-data, and bounds tests;
- secret/path/local-ID/binary/browser-data/Evidence exclusion tests;
- dependency resolution, incompatibility, and current-local-validation tests;
- new-identity and preserve-identity behavior tests;
- import/export/template idempotency, cancellation, interruption, conflict, persistence, rollback, and recovery tests;
- Windows and Linux platform-specific tests where behavior differs;
- Desktop/Wails and frontend tests;
- Phase 4, M5.1, and M5.2 regression review.

Actions must remain paused unless the owner separately authorizes a final validation run. Lack of a new Actions run must be recorded truthfully; it must not be represented as a passing new CI result.

## Exact next task

1. merge the documentation-only handoff that names Issue #58 as the single current task;
2. create one short-lived M5.3 implementation branch from the merged handoff baseline;
3. open one Draft PR for the complete M5.3 milestone;
4. develop in concentrated stages without workflow changes or iterative remote pushes;
5. perform local/offline validation and static review before any consolidated publication;
6. do not start M5.4 before M5.3 merges and passes a dedicated Closing Review.
