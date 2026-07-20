# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: 51c469e51ec4cab4ade99efd83c2e6c26145f266
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.2 — Safe Local Recovery
Current task: Implement Issue #54 after this documentation handoff merges
Planned implementation branch: `agent/m5-2-safe-local-recovery`

## Operational rule

Phase 5 remains `Active`. After this handoff merges, product implementation is allowed only for Issue #54.

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

1. **Stage 1 — Contracts and persistence**
   - versioned manifests and operation records;
   - strict decoding, canonical relative paths, deterministic identity, bounds, persistence, and rollback tests.

2. **Stage 2 — Local snapshot creation**
   - preflight, private staging, opaque file copying, hashing, safe cancellation points, verification, publication, and recovery.

3. **Stage 3 — Restore to new identity**
   - strict verification, dependency requirement remapping, new identity activation, limited-state behavior, and rollback.

4. **Stage 4 — Local lifecycle storage operations**
   - archive and recovery state transitions, recoverable removal state, retention state, explicit final cleanup, and startup recovery.

5. **Stage 5 — Desktop/Wails API and minimum UI**
   - bounded preflight, progress, results, history, cancellation availability, local recovery list, recovery state, and confirmations.

6. **Stage 6 — Integration, documentation, protected CI, and Closing Review handoff**
   - Windows/Linux real-filesystem and failure-injection coverage;
   - final scope review;
   - PR readiness and owner merge decision;
   - dedicated M5.2 Closing Review after merge.

Do not begin a later stage until the current stage's relevant tests pass.

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
- Windows and Linux real-filesystem operation tests;
- artifact exclusion tests;
- Phase 4 and M5.1 regression tests;
- official adapter and browser Evidence checks;
- exact Windows reviewed-Chromium identity, Network Evidence, tamper, artifact, and cleanup checks.

## Exact next task

After this handoff PR merges:

1. create branch `agent/m5-2-safe-local-recovery`;
2. open one Draft PR for Issue #54 before substantial product work;
3. begin Stage 1 only;
4. update this file in the same PR as product code;
5. keep M5.3 and M5.4 blocked.
