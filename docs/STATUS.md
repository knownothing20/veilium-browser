# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: 1d3b12582200328668061bb0dc382b3d24871fc0
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: Phase 5 Planning — Profile Lifecycle and Day-to-Day Operations
Current task: Complete Issue #37 and Draft PR #39 without implementing product code

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`.

Work only on Issue #37, Draft PR #39, `docs/PHASE_05.md`, `docs/PHASE_05_CONTRACTS.md`, and the required ROADMAP/STATUS planning updates.

Phase 5 is `Planning`. Product implementation is blocked. Do not create Go, React, Wails, schema-migration, backup-format, Cookie, extension, automation, proxy-expansion, sync, or release code until a separate reviewed activation decision sets `Product implementation allowed: Yes`.

## Frozen Phase 4 baseline

Phase 4 is `Done` and remains frozen.

The delivered baseline includes:

- Provider Contract v2 and legacy compatibility;
- real-browser identity Evidence;
- managed-window/consistency and Profile health;
- controlled Network Evidence and exact compatibility contracts;
- one exact reviewed official Chromium Snapshot Provider for Windows amd64;
- immutable archive, executable, and complete 261-file Package Tree identity;
- explicit license acknowledgement, secure extraction, atomic activation, rollback, and dependency-tamper detection;
- protected Windows/Linux Go, frontend, Wails, adapter, browser, Evidence, and build validation.

Phase 5 lifecycle artifacts cannot broaden Provider trust, capability status, platform support, or Evidence applicability from that frozen baseline.

## Current implementation facts used by the plan

The planning packet is based on the merged product structure:

- `profiles.json` contains Profile metadata, fingerprint settings, route configuration, and local dependency references;
- each Profile has a Veilium-managed browser user-data directory;
- deleting a Profile currently removes metadata but does not manage its browser-data directory;
- proxy secrets live only in the operating-system credential vault;
- Kernel and adapter binaries use independent managed registries and integrity records;
- browser and Network Evidence are independently stored, retained, and deleted;
- runtime logs, adapter runtime files, and temporary installer data are operational state rather than portable Profile identity.

Therefore Phase 5 does not define one naive all-inclusive portable folder. It separates local recovery, portable definitions, secrets, dependencies, and Evidence.

## Proposed Phase 5 user outcome

A user can preserve, recover, move, template, archive, and manage Veilium Profiles without silently copying secrets, weakening Provider trust, reusing identities unintentionally, or losing browser data during interrupted operations.

## Proposed milestone order

1. **M5.1 — Lifecycle Contract, Inventory, and Operation Journal**
   - lifecycle states, operation records, locks, cancellation, storage inventory, and startup reconciliation.
2. **M5.2 — Safe Local Snapshot, Restore, Archive, and Trash**
   - same-machine full snapshots, deterministic tree verification, staging, rollback, recoverable trash, and permanent deletion.
3. **M5.3 — Portable Profile Definitions and Templates**
   - secret-free configuration packages, dependency remapping, new-identity default, explicit identity preservation, and templates.
4. **M5.4 — Bounded Multi-profile Operations and Storage Management**
   - safe bulk metadata/lifecycle/export/health operations, storage inspection, cancellation, and truthful per-item results.

The complete scope and exit gates are in `docs/PHASE_05.md`. Logical records and safety contracts are in `docs/PHASE_05_CONTRACTS.md`.

## Key planning decisions

### Recovery before portability

- same-machine snapshot/restore is implemented before any broader migration claim;
- a full browser-data snapshot is treated as local/machine-bound unless a platform-specific tested path proves otherwise;
- the original healthy Profile remains recoverable until a restore is verified and committed.

### Portable definitions are configuration-only

Portable definitions exclude:

- browser user data;
- OS-vault secrets;
- local Profile IDs and absolute paths;
- local Kernel, adapter, and credential record IDs;
- executable binaries;
- runtime sessions, logs, temporary files, and Evidence.

They carry dependency requirements that must be explicitly mapped to current local records.

### Identity behavior

- new identity is the default import mode;
- identity preservation is explicit, warns against simultaneous use, and makes old Evidence non-applicable;
- templates always create a new Profile ID, managed directory, and fingerprint seed.

### Bounded operations only

Phase 5 may plan bulk tags/groups, archive/trash, definition export, integrity/health refresh, storage inspection, and explicit stop operations.

It does not authorize bulk start, scheduling, proxy rotation, account farming, public automation APIs, MCP, or cloud sync.

## Data and platform boundaries

- Profile metadata and validated non-secret configuration may be portable;
- browser user data and OS-vault secrets are local/machine-bound;
- Kernel/adapter identity and Evidence remain Provider-, binary-, platform-, route-, and configuration-bound;
- Windows and Linux are the proposed implementation and CI targets;
- local full snapshot behavior requires real filesystem integration tests on every claimed platform;
- the reviewed browser Provider remains Windows amd64 only;
- macOS and cross-platform full browser-state migration remain unclaimed.

## Explicit non-scope

Issue #37 and PR #39 do not authorize:

- Cookie import, export, or editing;
- extension installation or package management;
- exporting or synchronizing credential-vault secrets;
- bundling or updating Chromium, Xray, or sing-box in Profile artifacts;
- a second reviewed browser Provider or new fingerprint controls;
- proxy protocol expansion, pools, rotation, or scheduled health actions;
- bulk Profile start or general automation;
- public Launch API, CDP gateway, MCP, or cloud sync;
- release signing, auto-update, SBOM, or reproducible builds;
- unrelated broad UI redesign.

## Required planning validation

```bash
python scripts/check_project_governance.py
make check
```

The planning PR must also confirm:

- the final diff is documentation-only;
- ROADMAP, STATUS, and PHASE_05 metadata agree;
- the contract document contains no product-code claims presented as already implemented;
- Phase 4 remains frozen;
- Phase 5 remains non-implementable;
- no temporary workflow or unresolved review thread remains.

## Activation handoff

After Issue #37 and PR #39 are reviewed, a separate activation decision must:

1. approve the Phase 5 outcome, milestone order, contracts, platform policy, non-goals, validation matrix, and exit criteria;
2. change Phase 5 from `Planning` to `Active`;
3. change `Product implementation allowed` to `Yes`;
4. create or identify exactly one M5.1 implementation issue;
5. update ROADMAP and STATUS in the same activation pull request.

Until then, no Phase 5 product implementation is authorized.