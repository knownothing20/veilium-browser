# Current Project Status

Last updated: 2026-07-21
Application version: 0.15.0-dev
Main baseline SHA: 8097422edd06a648631394ab9ff8b987b0f7c313
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.2 — Safe Local Recovery
Current task: Implement Issue #54 Stage 4 on branch `agent/m5-2-safe-local-recovery`
Current implementation stage: Stage 4 — Local lifecycle storage operations

## Operational rule

Phase 5 remains `Active`. Product implementation is allowed only for Issue #54.

M5.3 and M5.4 remain blocked until the preceding milestone merges, passes a dedicated Closing Review, and this file advances again.

## Completed M5.1 handoff

M5.1 is complete:

- Issue #45 closed;
- PR #52 squash-merged as `51c469e51ec4cab4ade99efd83c2e6c26145f266`;
- Closing Review #53 passed and closed.

The M5.1 lifecycle records, operation journal, locks, blockers, cancellation state, storage inventory, startup recovery, Desktop integration, UI, tests, and documentation are the frozen foundation for M5.2.

## Current authority

Issue #54 is the single M5.2 implementation task.

M5.2 may implement only bounded same-machine recovery workflows described by Issue #54 and `docs/PHASE_05_CONTRACTS.md`.

All M5.2 work must:

- use the M5.1 journal, locks, blockers, cancellation state, inventory, and recovery records;
- require stopped runtime and protected dependent work;
- treat browser files as opaque data;
- reject unsafe, duplicate, linked, special, absolute, or out-of-root paths;
- enforce reviewed file, size, space, duration, manifest, and path bounds;
- use private staging, complete verification, atomic activation, and rollback;
- preserve the only healthy copy until replacement state validates;
- restore to a new Profile ID, managed directory, and fingerprint seed by default;
- remap dependency requirements without copying secrets or source record IDs;
- never broaden Provider trust, health, compatibility, or Evidence applicability;
- preserve interrupted or partial work as explicit recovery state.

## M5.2 implementation stages

Every development update must identify the current stage and the remaining stages.

1. **Stage 1 — Contracts and persistence — complete**
   - versioned manifest and catalog contracts;
   - canonical relative paths and deterministic file-tree identity;
   - non-secret dependency requirement records;
   - strict JSON boundaries, resource bounds, immutable manifest publication, atomic catalog persistence, explicit transitions, and rollback tests;
   - complete retained Governance and CI matrix passed on head `8cf514d3ea25685ee30903ba19e8f6f7eccf815e`.

2. **Stage 2 — Local snapshot creation — complete**
   - M5.1 operation locks, blockers, journal state, idempotency, and cancellation are reused;
   - bounded managed-directory preflight rejects path escape, links, reparse points, special entries, and hard-link ambiguity;
   - destination-space, file-count, per-file, total-byte, manifest, and duration bounds are enforced;
   - private staging copies opaque files with stable source identity and SHA-256 verification;
   - source and staged file sets are completely reverified before publication;
   - verified snapshots are atomically activated and catalogued;
   - cancellation, source changes, insufficient space, rename failure, cleanup failure, and catalog-finalization failure produce truthful rollback or recovery state;
   - complete retained Governance and CI matrix passed on head `361c39e8168696bfeb99266714d8b3c1a100ceaa`.

3. **Stage 3 — Restore to new identity — complete**
   - verified snapshots are completely revalidated before restore;
   - restore applies only to the current operating system, architecture, and same-machine scope;
   - current Kernel and adapter records are reverified before conservative dependency matching;
   - local IDs, executable paths, secrets, source Evidence, source Profile ID, and source fingerprint seed are not copied;
   - each idempotent restore receives one deterministic new Profile ID, managed directory, and fingerprint seed;
   - restored Profiles remain `draft` with explicit limitations until current validation and dependencies pass;
   - cancellation, snapshot tamper, target conflict, activation failure, metadata persistence failure, cleanup failure, and operation-finalization ambiguity produce truthful rollback or recovery state;
   - implementation, Windows/Linux tests, documentation, Governance, and the complete retained CI matrix passed on head `711b10d0486a63df4f9c7bf43887fdd9f1855287`.

4. **Stage 4 — Local lifecycle storage operations — active**
   - reversible archive and unarchive state transitions;
   - recoverable trash movement into a private managed boundary;
   - retention deadline and original managed identity preservation;
   - restore from trash through verified atomic movement;
   - explicit bounded permanent cleanup of trashed Profile data only;
   - startup reconciliation and recovery-required state for interrupted moves or cleanup;
   - no Desktop API or UI implementation.

5. **Stage 5 — Desktop/Wails API and minimum UI — blocked**
   - bounded preflight, progress, results, history, cancellation availability, local recovery list, recovery state, and confirmations.

6. **Stage 6 — Integration, documentation, protected CI, and Closing Review handoff — blocked**
   - Windows/Linux real-filesystem and failure-injection coverage;
   - final scope review;
   - PR readiness and owner merge decision;
   - dedicated M5.2 Closing Review after merge.

Do not begin a later stage until the current stage's relevant tests pass.

## Stage 4 allowed work

Stage 4 may add only:

- archive requests and results mapped to the M5.1 journal;
- reversible lifecycle transitions between `available` or `draft` and `archived` while preserving Profile metadata and managed browser data;
- unarchive with state, lock, timestamp, and managed-directory validation;
- trash requests that move one stopped Profile's managed browser directory into an operation-owned private trash or quarantine boundary before authoritative metadata is changed;
- retention deadline, original managed directory, source identity, and recovery-reference recording;
- restore-from-trash that verifies the private source, destination vacancy, lifecycle state, Profile metadata, and atomic move before making the Profile usable again;
- explicit permanent cleanup only for one confirmed `trashed` Profile whose data remains inside the reviewed managed trash boundary;
- bounded cleanup that never follows links and never removes Kernel, adapter, credential, Evidence, snapshot, runtime, or unrelated data;
- deterministic rollback or `recovery-required` state for persistence failure, move failure, partial cleanup, interruption, or contradictory state;
- startup reconciliation for owned archive, trash, restore-trash, and permanent-cleanup artifacts;
- idempotency, conflict, active-session, dependent-operation, cancellation, Windows/Linux real-filesystem, and failure-injection tests;
- Stage 4 documentation.

Stage 4 must not add automatic retention cleanup, orphan deletion, multi-Profile batch operations, a general filesystem browser, Desktop APIs, or UI actions.

## Non-scope

Issue #54 does not authorize:

- portable cross-machine Profile transfer;
- cross-platform full browser-state claims;
- identity-preserving portable transfer;
- templates;
- Cookie or extension management;
- secret export;
- multi-Profile batch operations;
- automatic retention or orphan cleanup;
- remote APIs, MCP, cloud sync, or general automation;
- Provider, Kernel, adapter, fingerprint, proxy-protocol, compatibility, or Evidence expansion;
- macOS support claims;
- release and updater work.

## Frozen boundaries

Phase 4 and M5.1 remain frozen:

- reviewed browser trust remains restricted to the approved Windows amd64 package;
- custom and legacy Providers remain unpromoted;
- lifecycle artifacts cannot create reviewed trust or applicable Evidence;
- restored dependencies must map to current local verified records;
- vault secrets remain non-portable;
- lifecycle state remains independent from runtime, health, trust, compatibility, and Evidence;
- unsupported, missing, unsafe, partial, contradictory, and unverifiable state fails closed or remains explicitly limited.

Issue #49 remains a separate hosted-runner reliability investigation. It must not be addressed by weakening Sandbox or Evidence requirements inside Issue #54.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The implementation PR must also pass:

- Go formatting, vet, race/unit tests, and builds;
- frontend typecheck, tests, and production build;
- Windows and Linux Wails builds;
- strict schema and filesystem safety fixtures;
- persistence, staging, activation, rollback, interruption, cancellation, and storage-failure tests as each stage becomes active;
- active-session, protected-operation, conflict, and idempotency tests;
- Windows and Linux real-filesystem operation tests for every claimed stage;
- artifact exclusion tests;
- Phase 4 and M5.1 regression tests;
- official adapter and browser Evidence checks;
- exact Windows reviewed-Chromium identity, Network Evidence, tamper, artifact, and cleanup checks.

## Exact next task

1. implement Stage 4 archive and unarchive transitions first;
2. implement recoverable trash movement, restore-trash, explicit permanent cleanup, and startup reconciliation only after archive tests pass;
3. add active-session, conflict, idempotency, cancellation, rollback, interruption, and Windows/Linux real-filesystem tests;
4. run Stage 4 tests and the complete retained matrix;
5. keep Stages 5–6, M5.3, and M5.4 blocked until Stage 4 passes.
