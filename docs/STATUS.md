# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: 35dcb17062a9932e20f226b6ec3fb64a2f6772e8
Current phase: Phase 5
Current phase document: docs/PHASE_05.md
Current milestone: M5.1 — Lifecycle Contract, Inventory, and Operation Journal
Current task: Implement the bounded M5.1 scope in Issue #45 after the Phase 5 activation PR merges

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`.

This activation pull request is documentation-only. It does not contain product code.

After this activation PR merges:

- Phase 5 is `Active`;
- product implementation is allowed only for Issue #45;
- use one short-lived M5.1 implementation branch and one Draft PR;
- update this STATUS file in the same pull request as product-code changes;
- do not begin M5.2, M5.3, or M5.4 work until M5.1 closes through a dedicated review and STATUS advances.

`Product implementation allowed: Yes` is not broad authority. Any product work outside Issue #45 remains blocked.

## Activation decision

The project owner approved Phase 5 on 2026-07-20 through Issue #40.

The approved planning and activation packet includes:

- Planning Issue #37;
- Planning PR #39, merged as `531f56d49cebc79cf6aee7a24d8f972d6275ce6b`;
- `docs/PHASE_05.md`;
- `docs/PHASE_05_CONTRACTS.md`;
- Windows reviewed-Chromium CI reliability Hotfix #43, merged as `6d4b04a9668c87cc110a4c0d423909d45649b529`;
- Activation Review Issue #40;
- this documentation-only activation PR;
- M5.1 implementation Issue #45.

The approved Phase 5 outcome is:

> A user can preserve, recover, move, template, archive, and manage Veilium Profiles without silently copying secrets, weakening Provider trust, reusing identities unintentionally, or losing browser data during interrupted operations.

## Current authorized implementation: Issue #45

M5.1 may implement only the lifecycle foundation.

### Versioned lifecycle state

- lifecycle states with at least `available`, `draft`, `archived`, and `trashed`;
- conservative compatibility for existing Profiles;
- lifecycle state remains separate from runtime state and derived Profile health;
- lifecycle state cannot grant Provider trust, capability support, compatibility, or Evidence validity.

### Versioned operation journal

- stable operation IDs and schema versions;
- operation type, selected Profile IDs, stages, timestamps, terminal state, per-item result, cancellation state, limitations, and recovery action;
- deterministic duplicate and idempotency behavior;
- private, atomic, bounded persistence with strict decoding and rollback behavior.

M5.1 may reserve operation vocabulary for future milestones, but it must not implement those later operations.

### Conflict and cancellation control

- one conflicting lifecycle operation per Profile;
- active browser and protected dependent-operation blocking;
- cancellation only at safe checkpoints;
- interrupted or cancelled work cannot become successful implicitly;
- application shutdown preserves enough state for startup reconciliation.

### Managed-storage inventory

- report expected Profile directories that are present or missing;
- report unexpected or orphaned managed directories;
- report unsafe paths, links, junction/reparse escapes, special entries, or paths outside the managed root;
- provide bounded file-count and byte-size summaries where safe;
- remain read-only and non-destructive in M5.1.

### Startup reconciliation

- reconcile interrupted journal records and stale locks;
- report recognized leftover staging or quarantine state;
- report missing, orphaned, malformed, duplicate, unsupported, or contradictory state;
- provide actionable recovery status without automatically completing destructive work.

### Desktop/API boundary

Expose only the bounded state needed to understand:

- Profile lifecycle;
- runtime state;
- Profile health;
- operation state and cancellation availability;
- storage/inventory findings and recovery actions.

The UI extends the existing design and must not introduce an unrelated redesign.

## M5.1 explicit non-scope

Issue #45 does not authorize:

- snapshot container creation;
- snapshot or restore execution;
- archive/trash directory movement or permanent deletion;
- portable Profile export/import;
- templates;
- Cookie import, export, or editing;
- extension installation or package management;
- export of operating-system vault secrets;
- multi-Profile batch UI or operations;
- bulk Profile start, scheduling, proxy rotation, account farming, general automation, public API, MCP, or cloud sync;
- Provider, Kernel, adapter, fingerprint, proxy-protocol, compatibility, or Evidence claim expansion;
- macOS lifecycle support claims;
- release signing, auto-update, SBOM, or reproducible-build work.

## Frozen Phase 4 boundaries

Phase 4 remains `Done` and frozen:

- reviewed browser trust remains restricted to the exact Windows amd64 Chromium Snapshot package;
- custom and legacy Providers remain unpromoted;
- lifecycle records cannot create reviewed trust or Evidence applicability;
- imported or restored dependencies require current local verification;
- operating-system vault secrets remain non-portable by default;
- unsupported, modified, missing, stale, contradictory, and unverifiable states fail closed or remain explicitly limited;
- the user-local Windows CI staging fix changes only Runner reliability, not the browser binary, Sandbox, Evidence, or compatibility claims.

## Required M5.1 validation

```bash
python scripts/check_project_governance.py
make check
```

The M5.1 Draft PR must additionally pass the complete applicable matrix:

- Go formatting, vet, race/unit tests, and builds;
- frontend typecheck, tests, and production build;
- Windows and Linux Wails builds;
- schema compatibility and migration fixtures;
- atomic persistence and rollback failures;
- duplicate/conflict, active-session, cancellation, shutdown, and startup-reconciliation tests;
- Windows and Linux real-filesystem inventory tests;
- symlink, junction/reparse, traversal, special-file, malformed-data, duplicate-ID, oversized-record, and unsupported-version rejection;
- secret and browser-content exclusion checks;
- official adapter and Linux browser Evidence checks;
- exact Windows reviewed-Chromium installation, identity/window Evidence, Network Evidence, dependency-tamper, build, artifact, and safe-cleanup checks.

## M5.1 delivery and closure

1. Implement Issue #45 only.
2. Keep commits dependency ordered and reviewable.
3. Keep M5.2–M5.4 code out of the M5.1 PR.
4. Record schema, migration, rollback, security, privacy, platform, and recovery behavior in module documentation.
5. After implementation merges, open a dedicated M5.1 Closing Review.
6. Advance STATUS to M5.2 only after that review passes.
