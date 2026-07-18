# Current Project Status

Last updated: 2026-07-18
Application version: 0.12.0-dev
Main baseline SHA: 524888c8b86e5f003e0c9532a4d176dee36cade2
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.1 — Kernel Provider Contract v2
Current task: Implement Issue #17 as the single authorized Phase 4 product task

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
- repository governance, required PRs, eight required checks, force-push protection, and deletion protection;
- passing Go, frontend, Windows, Linux, and official-adapter CI on the current baseline.

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

The task is to replace the optimistic provider-capability model with a versioned trust and capability contract that supports:

- reviewed and custom provider classification;
- exact provenance, license, platform, version, and binary identity;
- explicit verified, partial, unsupported, unverified, and failed capability meanings;
- compatibility for existing `native-chromium` and `patched-chromium` records;
- fail-closed save and launch behavior;
- provider disable and rollback policy.

## Exact next action

Read Issue #17 and the existing fingerprint, kernel, domain, desktop-service, and frontend provider code. Prepare one scoped Draft PR for M4.1 only.

The M4.1 PR must not implement the M4.2 evidence harness, M4.3 consistency work, M4.4 network probes, or deferred Phase 5/6 features.

## Required M4.1 validation

At minimum:

```bash
python scripts/check_project_governance.py
make check
```

The implementation must also include:

- provider-contract schema and policy unit tests;
- legacy compatibility tests;
- modified, missing, unsupported, and ambiguous failure-path tests;
- frontend tests for capability-state and blocking behavior;
- Windows and Linux policy/build checks;
- migration and rollback analysis if persisted formats change.

## Active prohibitions

Do not:

- add new proxy protocols, transports, or proxy-pool operations;
- begin cookie, extension, full migration, Launch API, MCP, sync, or release work;
- copy source from reference browsers or kernels;
- select a reviewed provider from upstream marketing claims alone;
- claim advanced fingerprint support without exact provider contracts and later real-browser evidence;
- include unrelated refactors or broad UI redesign in the M4.1 PR.

## Known risks

- no reviewed browser provider has yet completed the Phase 4 evidence chain;
- current capability booleans are legacy declarations and cannot directly become `Verified`;
- exact provider licensing, maintained artifacts, and supported patch behavior still require review during M4.1 and M4.2;
- macOS remains unclaimed until a real validation path exists;
- `docs/STATUS.md` must be updated by every product-code PR, including the first M4.1 implementation PR.

## Handoff

The next development session must work only on Issue #17. New findings outside M4.1 are recorded as separate issues and do not enter the implementation PR unless required for correctness or safety and explicitly approved through scope review.