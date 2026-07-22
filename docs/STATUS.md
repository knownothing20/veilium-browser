# Current Project Status

Last updated: 2026-07-21
Application version: 0.15.0-dev
Main baseline SHA: 8097422edd06a648631394ab9ff8b987b0f7c313
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.2 — Safe Local Recovery
Current task: Owner review and merge decision for PR #56
Current implementation stage: M5.2 Stages 1–6 complete — ready for owner review

## Operational rule

Phase 5 remains `Active`. Product implementation remains limited to Issue #54 until PR #56 is merged and M5.2 passes its dedicated Closing Review.

M5.3 and M5.4 remain blocked until that post-merge review passes and a separate documentation-only handoff advances this file.

## Completed M5.1 handoff

M5.1 is complete:

- Issue #45 closed;
- PR #52 squash-merged as `51c469e51ec4cab4ade99efd83c2e6c26145f266`;
- Closing Review #53 passed and closed.

The M5.1 lifecycle records, operation journal, locks, blockers, cancellation state, storage inventory, startup recovery, Desktop integration, UI, tests, and documentation remain the frozen foundation for M5.2.

## Current authority

Issue #54 is the single M5.2 implementation task.

M5.2 implements only bounded same-machine recovery workflows described by Issue #54, `docs/PHASE_05_CONTRACTS.md`, `docs/LOCAL_RECOVERY.md`, and `docs/LOCAL_RECOVERY_DESKTOP.md`.

All M5.2 work:

- uses the M5.1 journal, locks, blockers, cancellation state, inventory, and recovery records;
- requires stopped runtime and protected dependent work;
- treats browser files as opaque data;
- rejects unsafe, duplicate, linked, special, absolute, or out-of-root paths;
- enforces reviewed file, size, space, duration, manifest, and path bounds;
- uses private staging, complete verification, atomic activation, and rollback;
- preserves the only healthy copy until replacement state validates;
- restores to a new Profile ID, managed directory, and fingerprint seed by default;
- remaps dependency requirements without copying secrets or source record IDs;
- never broadens Provider trust, health, compatibility, or Evidence applicability;
- preserves interrupted or partial work as explicit recovery state.

## M5.2 implementation stages

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

6. **Stage 6 — Integration, documentation, protected CI, and owner-review handoff — complete**
   - contract, Desktop, status, and pre-merge review documentation are synchronized;
   - final changed-file, non-scope, secret, path, rollback, lifecycle, confirmation, and recovery-state review passed;
   - no temporary workflow or diagnostic artifact remains in the changed-file set;
   - no PR conversation comment or inline review thread remains unresolved;
   - the Evidence collector keeps its declared shutdown deadline authoritative and force-closes only the bounded loopback server if Chromium retains an active request; a deterministic regression test covers this cleanup boundary;
   - Governance and the complete retained CI matrix passed on head `74ad752d56d56f6c0350437b250172a900bb7e08`;
   - `docs/M5_2_IMPLEMENTATION_REVIEW.md` records the pre-merge verdict `READY FOR OWNER REVIEW`;
   - PR #56 may be marked ready for review after the final documentation-only head remains green, but the owner retains the merge decision;
   - a dedicated M5.2 Closing Review remains mandatory after merge.

## Completed scope

M5.2 provides:

- versioned same-machine full snapshot records and strict catalogs;
- bounded, staged, verified, atomic snapshot creation;
- restore to a new limited identity with current local dependency remapping;
- reversible archive/unarchive;
- recoverable trash, exact restore-trash, retention metadata, and explicit irreversible cleanup;
- conservative startup reconciliation;
- bounded Desktop/Wails preflight, state, progress, history, cancellation, and action APIs;
- a minimum Local recovery workspace extending the existing design;
- Windows/Linux, failure-path, persistence, frontend, Wails, official-adapter, and reviewed-Chromium regression coverage.

## Non-scope

Issue #54 does not authorize and PR #56 does not implement:

- portable cross-machine Profile transfer;
- cross-platform full browser-state claims;
- identity-preserving portable transfer;
- templates;
- Cookie or extension management;
- secret export;
- multi-Profile batch operations;
- automatic retention or orphan cleanup;
- a general filesystem browser;
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

Issue #49 remains a separate hosted-runner reliability investigation. PR #56 does not weaken Sandbox or Evidence requirements.

## Required validation

The reviewed implementation passed:

```bash
python scripts/check_project_governance.py
make check
```

The protected matrix also passed:

- Go formatting, vet, race/unit tests, and builds;
- frontend typecheck, tests, and production build;
- Windows and Linux Wails builds;
- strict schema and filesystem safety fixtures;
- persistence, staging, activation, rollback, interruption, cancellation, and storage-failure tests;
- active-session, protected-operation, conflict, and idempotency tests;
- Windows and Linux real-filesystem operation tests;
- artifact exclusion tests;
- Phase 4 and M5.1 regression tests;
- official adapter and browser Evidence checks, including bounded collector shutdown;
- exact Windows reviewed-Chromium identity, Network Evidence, tamper, artifact, and cleanup checks.

## Exact next task

1. confirm the final documentation-only head passes Governance and protected CI;
2. mark PR #56 ready for owner review;
3. owner reviews the complete PR and chooses whether to squash-merge it;
4. do not start M5.3 or M5.4 before merge and dedicated Closing Review;
5. after merge, create a dedicated M5.2 Closing Review against the merged main commit;
6. only a separate documentation-only handoff may authorize the next milestone.
