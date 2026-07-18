# Current Project Status

Last updated: 2026-07-19
Application version: 0.14.0-dev
Main baseline SHA: 7306215085b578755d4980180edb9f451e5a9f14
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.3 — Identity and Window Consistency
Current task: Complete Issue #22 in Draft PR #24

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. Work only on the current issue and milestone.

## Delivered baseline

M4.1 and M4.2 are complete.

M4.1 established Provider Contract v2, explicit trust and capability states, legacy compatibility, managed binary identity, and fail-closed advanced configuration.

M4.2 was delivered by Issue #20 and PR #21, squash-merged as `094fea4f03c5a87e37f69a4868fd26e609673c6e`. It added controlled loopback real-browser evidence, private bounded reports, desktop report views, and Required Windows/Linux collection checks.

M4.2 does not grant reviewed status to a production browser Provider.

## Current milestone

### M4.3 — Identity and Window Consistency

Current implementation issue: #22
Current Draft PR: #24

Implemented in the Draft branch so far:

- optional explicit `windowWidth`, `windowHeight`, and `deviceScaleFactor` fields;
- backward-compatible fallback to existing screen dimensions without rewriting profiles;
- WindowPlan and observed WindowState contracts;
- versioned consistency Result, Check, Health, WindowSource, and EvidenceInput contracts;
- pre-launch Provider, platform, screen, window, and DPR checks;
- deterministic consistency input digests for evidence freshness;
- observed window parsing and initial controlled tolerances;
- evidence-backed `healthy`, `degraded`, `blocked`, and `unknown` derivation;
- initial compatibility, mismatch, stale-evidence, and reviewed-evidence tests.

Work still required before M4.3 can merge:

1. integrate preflight into profile save, launch-plan generation, and runtime start;
2. make launch arguments and LaunchPlan use the effective window rather than screen dimensions;
3. add bounded loopback CDP window apply/readback behavior;
4. record consistency input digests on new Evidence runs and keep old reports readable but stale;
5. move final screen/window/viewport/DPR policy into M4.3 health evaluation;
6. expose current profile health and readable reasons through desktop and frontend layers;
7. add Windows/Linux controlled-window fixtures and tolerance documentation;
8. create the M4.4 handoff issue and update this file;
9. make Governance and every Required CI job pass;
10. complete review and protected squash merge.

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

The M4.3 PR must also pass the protected Windows, Linux, frontend, Wails, official-adapter, and applicable controlled-window checks.

## Handoff

Continue only in Issue #22 and Draft PR #24. Out-of-scope findings become separate issues.
