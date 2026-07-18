# Current Project Status

Last updated: 2026-07-19
Application version: 0.14.0-dev
Main baseline SHA: 094fea4f03c5a87e37f69a4868fd26e609673c6e
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.3 — Identity and Window Consistency
Current task: Implement Issue #22 as the single authorized Phase 4 product task

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. Work only on the current issue and milestone.

## Delivered baseline

M4.1 and M4.2 are complete.

M4.1 established Provider Contract v2, explicit trust and capability states, legacy compatibility, managed binary identity, and fail-closed advanced configuration.

M4.2 was delivered by Issue #20 and PR #21, squash-merged as `094fea4f03c5a87e37f69a4868fd26e609673c6e`. It added:

- versioned local Evidence Run and Observation records;
- private, bounded, independently deletable evidence storage;
- loopback-only controlled browser collection;
- top-level, same-origin iframe, and Worker observations;
- binding to the exact managed profile, session, Provider revision, and binary identity;
- cancellation, timeout, browser-exit, storage-failure, and shutdown cleanup;
- desktop report actions and views;
- Required Windows and Linux real-browser collection checks;
- `docs/REAL_BROWSER_EVIDENCE.md`.

M4.2 does not grant reviewed status to a production browser Provider.

## Current milestone

### M4.3 — Identity and Window Consistency

Current implementation issue: #22

M4.3 owns:

- versioned consistency rules and results;
- OS, browser, language, timezone, CPU, GPU mode, screen, available screen, window, viewport, scale, and DPR consistency;
- real-window sizing and restoration on supported Windows and Linux paths;
- documented tolerances for browser chrome, work areas, decorations, rounding, and controlled fixtures;
- pre-launch blocking of contradictory, impossible, unsupported, or stale required combinations;
- stable relaunch behavior;
- evidence-freshness invalidation;
- derived `healthy`, `degraded`, `blocked`, and `unknown` profile health;
- readable desktop reasons and recovery guidance without silently rewriting profiles.

## Exact next action

Read Issue #22 and audit:

1. `internal/fingerprint` validation and capabilities;
2. `internal/launch` planning and arguments;
3. `internal/supervisor` session and window lifecycle;
4. `internal/evidence` observations and freshness inputs;
5. profile persistence timestamps;
6. frontend profile-health presentation;
7. Windows/Linux controlled browser fixtures.

Prepare one scoped Draft PR for M4.3 only.

## Active prohibitions

Do not:

- begin M4.4 external network evidence;
- claim reviewed Provider status without exact applicable evidence;
- silently rewrite or auto-correct profiles;
- add new proxy protocols or pool operations;
- begin cookie, extension, migration, public API, MCP, sync, or release work;
- copy code from reference browsers or kernels;
- expose browser control beyond the existing local safety boundary;
- include unrelated refactors or broad UI redesign.

## Known risks

- no production browser Provider is reviewed yet;
- Windows and Linux window decoration and scaling need separate tolerances;
- headless CI does not prove real desktop-window support;
- freshness invalidation must be conservative;
- existing profiles must remain readable;
- macOS remains unclaimed.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The M4.3 PR must also pass the protected Windows, Linux, frontend, Wails, official-adapter, and applicable controlled window checks.

## Handoff

The next development session must work only on Issue #22. Out-of-scope findings become separate issues.
