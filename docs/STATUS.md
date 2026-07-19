# Current Project Status

Last updated: 2026-07-19
Application version: 0.14.0-dev
Main baseline SHA: 139907936179ee61d4fcd82b19125c1535bb8e2a
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.4 — Live Browser Network Evidence and Compatibility Matrix
Current task: Implement Issue #25 as the single authorized Phase 4 product task

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. Work only on Issue #25 and M4.4.

M4.4 is the final implementation milestone before a separate Phase 4 closing review.

## Delivered baseline

M4.1, M4.2, and M4.3 are complete.

- M4.1 PR #19 established Provider Contract v2, exact managed binary identity, explicit trust/capability states, and fail-closed legacy compatibility.
- M4.2 PR #21, merged as `094fea4f03c5a87e37f69a4868fd26e609673c6e`, established controlled local real-browser Evidence, private reports, desktop inspection, and Windows/Linux fixtures.
- M4.3 PR #24, merged as `139907936179ee61d4fcd82b19125c1535bb8e2a`, established versioned consistency rules, Evidence freshness, managed browser windows, derived Profile health, desktop controls, and Windows/Linux real-window validation.

No production browser Provider is marked reviewed solely by these milestones.

## Current milestone

### M4.4 — Live Browser Network Evidence and Compatibility Matrix

Current implementation issue: #25

M4.4 owns:

- browser-observed exit-route evidence;
- controlled WebRTC/STUN and delegated-DNS route checks;
- classification of Direct, built-in bridge, Xray, and sing-box routes;
- health integration so route mismatch or leak cannot be healthy;
- replaceable or self-hostable probe definitions;
- exact Provider/version/OS/architecture/capability compatibility records;
- generated compatibility documentation;
- Phase 4 completion handoff.

## Exact next action

Read Issue #25 and audit:

1. `internal/evidence` contracts and lifecycle;
2. current proxy diagnostics and route classification;
3. proxy bridge and adapter runtime lifecycle;
4. controlled browser page and Target flow;
5. consistency health integration;
6. desktop Evidence report UI;
7. Windows/Linux controlled Chromium fixtures.

Create one scoped Draft PR for M4.4 only.

## Active prohibitions

Do not:

- begin Phase 5 work;
- silently change Profile or route configuration;
- grant reviewed Provider status from network evidence alone;
- add new proxy protocols, pool rotation, or batch operations;
- begin public API, MCP, sync, cookie, extension, migration, or release work;
- rely on a single non-replaceable probe;
- claim macOS support without real validation;
- include unrelated refactors or broad UI redesign.

## Known risks

- different network probes observe different layers and may disagree;
- probe unavailability must remain explicit rather than optimistic;
- CI proves only exact tested combinations;
- custom and legacy Provider trust remains conservative;
- macOS remains unclaimed.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The M4.4 PR must also pass protected Windows, Linux, frontend, Wails, official-adapter, controlled-probe, privacy, and relevant real-browser checks.

## Handoff

The next development session must work only on Issue #25. After M4.4 merges, create a separate Phase 4 closing-review PR; do not silently enter Phase 5.