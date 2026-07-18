# Current Project Status

Last updated: 2026-07-18
Application version: 0.13.0-dev
Main baseline SHA: 863e88cbbbc1c904dfbcda967be028d36ccb9ece
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.1 — Kernel Provider Contract v2
Current task: Complete Issue #17 in Draft PR #19 and make every required check pass

## Operational rule

This is the first file to read after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. It identifies the only approved next task. It does not override the product charter or active phase document.

Phase 4 is active. Product implementation is allowed only inside the ordered Phase 4 milestones and the explicitly approved issue scope.

## Current state

Completed foundations include:

- clean-room core contracts and local profile persistence;
- Wails and React desktop profile workspace;
- verified local Chromium kernel registry;
- supervised browser process lifecycle on Windows and Unix-like systems;
- operating-system credential vault;
- authenticated HTTP, HTTPS, and SOCKS5 loopback bridges;
- proxy diagnostics;
- managed and supervised Xray and sing-box providers;
- pinned official adapter validation and explicit installer;
- repository governance, protected `main`, required pull requests, required checks, force-push protection, and deletion protection.

## Phase 4 approved outcome

At Phase 4 completion, users can select a reviewed browser-kernel provider, configure only supported capabilities, launch a profile, and receive local evidence showing whether the declared identity and selected network route were observed in the real browser session.

Capability states are:

- Verified;
- Partially verified;
- Unsupported;
- Unverified custom provider.

The authoritative phase scope, milestones, non-goals, platform policy, validation, rollback rules, and exit criteria are in `docs/PHASE_04.md`. The logical provider, capability, evidence, compatibility, and health contracts are in `docs/PHASE_04_CONTRACTS.md`.

## Current milestone

### M4.1 — Kernel Provider Contract v2

Current implementation issue: #17
Current Draft PR: #19

Implemented in the Draft branch:

- provider contract schema version 2;
- reviewed, custom, legacy, disabled, and invalid provider trust states;
- verified, partial, unsupported, unverified, and failed capability states;
- a generic `custom-chromium` contract;
- compatibility definitions for legacy `native-chromium` and `patched-chromium` IDs;
- fail-closed backend validation for advanced unverified claims;
- Provider-derived desktop bootstrap data instead of a hard-coded UI provider list;
- frontend trust and capability states, generic defaults, status labels, and disabled unsupported controls;
- derived managed-kernel binary identity that keeps integrity and reviewed trust separate;
- explicit predecessor and rollback policy for provider replacements;
- provider, validation, kernel identity, rollback, and frontend tests;
- updated kernel registry documentation;
- temporary formatting diagnostics removed after the generated result was applied.

Work still required before M4.1 can merge:

1. use required CI failures to find any remaining legacy boolean assumptions or incompatible tests;
2. add only the service, launch, or compatibility tests required by those failures;
3. confirm no legacy record is silently upgraded and generic custom launch remains usable;
4. make Governance, Go, frontend, Windows, Linux, desktop, and adapter checks pass;
5. update the PR description and this handoff with the exact next M4.1 or M4.2 task.

## Active prohibitions

Do not:

- implement the M4.2 evidence harness in PR #19;
- add window, viewport, DPR, live WebRTC, DNS, or exit-IP evidence;
- add new proxy protocols, transports, or proxy-pool operations;
- begin cookie, extension, full migration, Launch API, MCP, sync, or release work;
- copy source from reference browsers or kernels;
- select a reviewed provider from upstream marketing claims alone;
- claim advanced fingerprint support without exact provider contracts and later real-browser evidence;
- include unrelated refactors or broad UI redesign.

## Known risks

- no production provider is reviewed yet; M4.1 must not fabricate one;
- existing `patched-chromium` profiles with advanced settings remain readable but are blocked from claiming reviewed support until a reviewed replacement exists;
- integrity status and provider trust are separate and must not be presented as the same check;
- exact provider licensing, maintained artifacts, and real behavior remain future evidence work;
- macOS remains unclaimed until a real validation path exists.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The PR must also pass the protected-branch Windows, Linux, desktop-build, frontend, and official-adapter checks.

## Handoff

Continue only in Draft PR #19 and Issue #17. Use CI failures to find remaining legacy assumptions, update only files required by M4.1, and keep `docs/STATUS.md` synchronized with merged behavior and the next task.
