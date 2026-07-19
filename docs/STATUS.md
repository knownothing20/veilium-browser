# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: 531f56d49cebc79cf6aee7a24d8f972d6275ce6b
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: Phase 5 Activation Review
Current task: Review and decide Phase 5 activation in Issue #40 without implementing product code

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`.

Work only on Issue #40 and its documentation-only activation decision. Phase 5 remains `Planning`. Product implementation is blocked until a reviewed activation PR explicitly changes `Product implementation allowed` to `Yes` and identifies exactly one M5.1 implementation issue.

Do not create Go, React, Wails, persisted-schema, snapshot/restore, portable-definition, template, Cookie, extension, bulk-operation, automation, proxy-expansion, sync, or release code during the activation review.

## Completed planning packet

Issue #37 and PR #39 are complete. PR #39 merged as `531f56d49cebc79cf6aee7a24d8f972d6275ce6b` after Governance and the complete existing CI matrix passed.

The approved planning packet contains:

- `docs/PHASE_05.md` — user outcome, four dependency-ordered milestones, non-goals, platform policy, validation matrix, exit criteria, and activation gate;
- `docs/PHASE_05_CONTRACTS.md` — lifecycle, operation, storage inventory, local snapshot, portable definition, template, dependency, migration, cancellation, recovery, resource-bound, and platform contracts;
- `docs/ROADMAP.md` — Phase 5 as the current `Planning` phase;
- this status handoff.

Planning completion does not activate Phase 5 and does not prove that any lifecycle feature is implemented.

## Proposed Phase 5 outcome

A user can preserve, recover, move, template, archive, and manage Veilium Profiles without silently copying secrets, weakening Provider trust, reusing identities unintentionally, or losing browser data during interrupted operations.

## Proposed milestone order

1. **M5.1 — Lifecycle Contract, Inventory, and Operation Journal**
   - lifecycle states, operation records, per-item results, locks, cancellation, storage inventory, and startup reconciliation.
2. **M5.2 — Safe Local Snapshot, Restore, Archive, and Trash**
   - same-machine full snapshots, deterministic tree verification, staging, rollback, recoverable trash, and explicit permanent deletion.
3. **M5.3 — Portable Profile Definitions and Templates**
   - secret-free configuration packages, dependency remapping, new-identity default, explicit identity preservation, and templates.
4. **M5.4 — Bounded Multi-profile Operations and Storage Management**
   - bounded bulk metadata/lifecycle/export/health operations, storage inspection, cancellation, and truthful per-item results.

## Activation decision scope

Issue #40 must review and either approve or block:

1. the recoverability-first user outcome;
2. the four milestone order;
3. the separation between local full snapshots, portable definitions, templates, secrets, dependencies, Evidence, and runtime data;
4. new-identity default and explicit identity-preserving transfer;
5. Windows/Linux targets and macOS/cross-platform non-claims;
6. security, privacy, licensing, migration, interruption, rollback, and resource-bound contracts;
7. the validation matrix and Phase 5 exit gates;
8. all explicit non-goals and inherited Phase 4 boundaries.

## Required first implementation scope

If activation is approved, exactly one M5.1 implementation issue must be created or identified. It may cover only:

- versioned Profile lifecycle states;
- versioned operation journal and per-item results;
- operation conflict/locking and active-session blocking;
- cancellation checkpoints;
- managed-storage inventory and orphan/missing reporting;
- startup reconciliation for interrupted journal, staging, and quarantine state;
- conservative compatibility for existing Profiles.

It must not include snapshot containers, restore, archive/trash data movement, portable export/import, templates, or multi-profile UI from later milestones.

## Frozen Phase 4 boundaries

Phase 4 remains `Done` and frozen:

- reviewed browser trust remains restricted to the exact Windows amd64 Chromium Snapshot package;
- lifecycle metadata cannot create Provider trust, capability support, compatibility, health, or Evidence validity;
- imported or restored dependencies require current local verification;
- OS-vault secrets remain non-portable by default;
- custom and legacy Providers remain unpromoted;
- unsupported, modified, missing, stale, contradictory, and unverifiable states fail closed or remain explicitly limited.

## Explicit non-scope during activation

Issue #40 does not authorize:

- product-code implementation of any Phase 5 milestone;
- Cookie import, export, or editing;
- extension installation or package management;
- secret export or cloud synchronization;
- bundling/updating Chromium, Xray, or sing-box in Profile artifacts;
- a second reviewed Provider or new fingerprint controls;
- proxy expansion, pools, rotation, scheduling, or account farming;
- bulk Profile start or general automation;
- public Launch API, CDP gateway, MCP, or cloud sync;
- release signing, auto-update, SBOM, or reproducible builds;
- unrelated broad UI redesign.

## Activation outcomes

### Approve

Create a documentation-only activation PR that:

- changes `docs/PHASE_05.md` from `Planning` to `Active`;
- changes `Product implementation allowed: No` to `Yes`;
- identifies the single M5.1 implementation issue;
- updates ROADMAP and STATUS in the same PR;
- preserves all contracts, non-goals, and inherited Phase 4 boundaries;
- passes Governance and the complete existing CI matrix.

### Block

Keep Phase 5 in `Planning`, record the precise plan gap, and create one narrow planning correction. Product implementation remains blocked.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The activation handoff must remain documentation-only and must not create a temporary workflow or leave unresolved review threads.