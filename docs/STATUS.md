# Current Project Status

Last updated: 2026-07-19
Application version: 0.14.0-dev
Main baseline SHA: dcfcee9e4c8b8587ae0c8c44a63103cb0c5c5d6c
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.4 — Live Browser Network Evidence and Compatibility Matrix
Current task: Implement Issue #25 on `agent/m4-4-network-evidence`

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
Current implementation branch: `agent/m4-4-network-evidence`

The first M4.4 batch establishes:

- independent versioned Network Evidence records tied to existing Evidence Runs;
- explicit run, observation, probe, route, and compatibility states;
- probe sets with no hidden default third-party endpoint;
- HTTPS or loopback-only exit-IP definitions;
- explicit STUN and delegated-DNS definitions;
- mandatory replaceable or self-hostable probe policy;
- route classification and SHA-256 route identity without storing the original route;
- exact Provider/version/OS/architecture/binary/capability compatibility entries;
- trust rules preventing custom or legacy Providers from producing verified matrix entries;
- automatic conversion of expired accepted evidence to stale;
- contract, privacy, route, and matrix tests.

This batch does not yet execute browser probes or change Profile health.

## Remaining M4.4 work

1. open one scoped Draft PR for Issue #25;
2. add private Network Evidence storage and lifecycle management;
3. extend the controlled browser page for explicit exit-IP, STUN, and delegated-DNS probes;
4. bind collection to the selected managed session and route;
5. integrate route evidence with Profile health;
6. add desktop report and probe-configuration surfaces;
7. generate compatibility records and repository documentation;
8. add controlled Windows/Linux real-browser probe fixtures;
9. complete Phase 4 closing-review handoff.

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

Continue only on Issue #25 and its single Draft PR. After M4.4 merges, create a separate Phase 4 closing-review PR; do not silently enter Phase 5.