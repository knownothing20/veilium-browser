# Phase 5 — Profile Lifecycle and Day-to-Day Operations

Status: Active
Phase: Phase 5
Owner decision required: No
Product implementation allowed: Yes

## Activation record

Phase 5 was approved through Activation Review Issue #40 and activated through PR #47.

The planning contract remains defined by:

- Planning Issue #37;
- Planning PR #39;
- `docs/PHASE_05_CONTRACTS.md`;
- `docs/ROADMAP.md`;
- `docs/STATUS.md`.

## Milestone record

M5.1 is complete:

- implementation Issue #45;
- implementation PR #52, squash-merged as `51c469e51ec4cab4ade99efd83c2e6c26145f266`;
- Closing Review #53 — PASS.

M5.2 is the current milestone and Issue #54 is the only authorized implementation issue.

M5.3 and M5.4 remain blocked until the preceding milestone merges, passes a dedicated Closing Review, and `docs/STATUS.md` advances.

## Milestone order

1. M5.1 — Lifecycle Contract, Inventory, and Operation Journal
2. M5.2 — Safe Local Recovery
3. M5.3 — Portable Profile Definitions and Templates
4. M5.4 — Bounded Multi-profile Operations and Storage Management

A later milestone must not begin before the current milestone is merged, reviewed, and handed off through `docs/STATUS.md`.

## Current M5.2 authority

Issue #54 may extend the M5.1 lifecycle foundation with bounded same-machine recovery workflows.

All M5.2 implementation must:

- use the M5.1 lifecycle records, operation journal, locks, blockers, cancellation state, inventory, and startup recovery;
- keep Profile metadata, lifecycle state, runtime state, health, Provider trust, compatibility, and Evidence separate;
- require stopped runtime and protected dependent work before managed storage changes;
- treat browser files as opaque data;
- enforce reviewed path, entry, file, size, space, duration, manifest, and staging bounds;
- verify replacement state before activation;
- preserve the only healthy copy until replacement state validates;
- restore into a new Profile identity by default;
- remap dependency requirements without copying secrets or source record IDs;
- preserve interrupted or partial work as explicit recovery state;
- never broaden Provider trust, compatibility, health, or Evidence applicability.

Detailed M5.2 scope, acceptance criteria, and non-scope are authoritative in Issue #54. Logical records and transitions remain authoritative in `docs/PHASE_05_CONTRACTS.md`.

## Frozen boundaries

- secrets remain in the operating-system credential vault;
- custom and legacy Providers remain unpromoted;
- reviewed browser trust remains restricted to the approved Windows amd64 package;
- artifacts cannot create reviewed trust or applicable Evidence;
- Windows and Linux are the minimum claimed lifecycle targets;
- macOS and cross-platform full browser-state recovery remain unclaimed;
- no general filesystem browser, remote API, MCP, cloud sync, or unrelated UI redesign is authorized.

## Validation

Every milestone requires the strongest applicable subset of:

- governance and documentation checks;
- formatting, static analysis, unit tests, race tests, and builds;
- frontend typecheck, tests, and production build;
- Windows and Linux Wails builds;
- strict schema, path, entry, bound, interruption, persistence, and rollback tests;
- Windows and Linux real-filesystem integration tests;
- artifact exclusion checks;
- Phase 4 and earlier Phase 5 regression checks.

## Implementation control

The current implementation authority is Issue #54 only.

Each milestone requires:

1. a short-lived implementation branch;
2. one Draft PR with `docs/STATUS.md` updated alongside product code;
3. Governance and the complete applicable CI matrix;
4. a dedicated Closing Review before STATUS advances;
5. no later-milestone work before explicit handoff.
