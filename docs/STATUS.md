# Current Project Status

Last updated: 2026-07-21
Application version: 0.15.0-dev
Main baseline SHA: 8097422edd06a648631394ab9ff8b987b0f7c313
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.2 — Safe Local Recovery
Current task: Complete Issue #54 Stage 6 on branch `agent/m5-2-safe-local-recovery`
Current implementation stage: Stage 6 — Integration, documentation, protected CI, and owner-review handoff

## Operational rule

Phase 5 remains `Active`. Product implementation is allowed only for Issue #54.

M5.3 and M5.4 remain blocked until M5.2 merges, passes a dedicated Closing Review, and this file advances again.

## Completed M5.1 handoff

M5.1 is complete:

- Issue #45 closed;
- PR #52 squash-merged as `51c469e51ec4cab4ade99efd83c2e6c26145f266`;
- Closing Review #53 passed and closed.

The M5.1 lifecycle records, operation journal, locks, blockers, cancellation state, storage inventory, startup recovery, Desktop integration, UI, tests, and documentation remain the frozen foundation for M5.2.

## Current authority

Issue #54 is the single M5.2 implementation task.

M5.2 may implement only bounded same-machine recovery workflows described by Issue #54, `docs/PHASE_05_CONTRACTS.md`, `docs/LOCAL_RECOVERY.md`, and `docs/LOCAL_RECOVERY_DESKTOP.md`.

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

4. **Stage 4 — Local lifecycle storage operations — complete**
   - reversible archive and unarchive preserve the exact origin lifecycle state;
   - recoverable trash moves only the Profile-owned managed browser directory into a private verified boundary;
   - retention deadline and original managed identity are preserved;
   - restore-trash completely revalidates and atomically restores the exact original managed location and lifecycle metadata;
   - explicit permanent cleanup requires exact Profile confirmation and removes only verified owned trash data and matching Profile metadata;
   - startup reconciliation reports interrupted and contradictory states without moving or deleting data automatically;
   - failure after a commit or irreversible boundary remains truthful through rollback or `recovery-required` state;
   - Governance and the complete retained CI matrix passed on head `4e3b0c39f561e1a00cf863663739dff5f6c49753`.

5. **Stage 5 — Desktop/Wails API and minimum UI — complete**
   - bounded local recovery state, snapshot and trash listing, snapshot detail, and Profile preflight APIs;
   - snapshot, restore-to-new-identity, archive, unarchive, recoverable trash, restore-trash, permanent-delete, refresh, and safe-cancellation Wails actions;
   - Desktop progress projection from the authoritative M5.1 operation journal rather than a second task system;
   - conservative startup trash reconciliation when the Desktop recovery service initializes;
   - legacy Wails Profile deletion routed through recoverable trash while direct metadata deletion remains fail-closed;
   - minimum Local recovery workspace with Profile actions, verified snapshot cards, trash cards, operation progress/history, exact irreversible confirmation, and recovery-required findings;
   - browser preview remains non-operational and no general filesystem browser is introduced;
   - Desktop service tests, Go formatting/vet/unit/race/build, frontend typecheck/tests/build, Windows/Linux Wails builds, official adapters, Linux browser checks, and exact Windows reviewed-Chromium checks passed on head `8035a4ac53c1cafe85c129b1239ad9677a5f8fbc`.

6. **Stage 6 — Integration, documentation, protected CI, and owner-review handoff — active**
   - final contract and Desktop documentation synchronization;
   - final changed-file, non-scope, secret, path, rollback, lifecycle, confirmation, and recovery-state review;
   - review-thread and temporary-artifact check;
   - protected Governance and CI confirmation on the final documentation head;
   - PR description and readiness update;
   - owner merge decision;
   - dedicated M5.2 Closing Review after merge.

Do not begin M5.3 or M5.4 during Stage 6.

## Stage 6 allowed work

Stage 6 may add or change only:

- implementation-status and local recovery documentation;
- the pre-merge implementation review record;
- PR description, review readiness, and reviewer-facing evidence;
- corrections discovered by the final M5.2 scope, safety, or protected-CI review;
- test-only or CI-only corrections required to preserve existing protected assertions without broadening product scope.

Stage 6 must not add new product capability, automatic retention cleanup, orphan deletion, remote APIs, multi-Profile batch operations, templates, portable transfer, or any Provider/Evidence expansion.

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
- persistence, staging, activation, rollback, interruption, cancellation, and storage-failure tests;
- active-session, protected-operation, conflict, and idempotency tests;
- Windows and Linux real-filesystem operation tests for every claimed stage;
- artifact exclusion tests;
- Phase 4 and M5.1 regression tests;
- official adapter and browser Evidence checks;
- exact Windows reviewed-Chromium identity, Network Evidence, tamper, artifact, and cleanup checks.

## Current review evidence

- Stage 5 protected Governance and CI passed on `8035a4ac53c1cafe85c129b1239ad9677a5f8fbc`.
- No PR conversation comment or inline review thread is currently unresolved.
- The final changed-file list contains no workflow file and no temporary diagnostic artifact.
- `docs/M5_2_IMPLEMENTATION_REVIEW.md` records the pre-merge functional, safety, privacy, platform, and non-scope review.

## Exact next task

1. run Governance and the complete retained CI matrix on the final Stage 6 documentation head;
2. confirm the PR diff and review state remain clean;
3. mark Stage 6 complete and PR #56 ready for owner review;
4. do not merge without the owner decision;
5. after merge, create and perform the dedicated M5.2 Closing Review before any M5.3 handoff.
