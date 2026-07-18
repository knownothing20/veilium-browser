# Current Project Status

Last updated: 2026-07-18
Application version: 0.13.0-dev
Main baseline SHA: 863e88cbbbc1c904dfbcda967be028d36ccb9ece
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.2 — Real-Browser Evidence Harness
Current task: Begin Issue #20 after M4.1 PR #19 is merged

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

The authoritative phase scope, milestones, non-goals, platform policy, validation, rollback rules, and exit criteria are in `docs/PHASE_04.md`. The logical provider, capability, evidence, compatibility, and health contracts are in `docs/PHASE_04_CONTRACTS.md`.

## M4.1 delivery

M4.1 — Kernel Provider Contract v2 is delivered by Issue #17 and PR #19.

The merged behavior establishes:

- provider contract schema version 2;
- reviewed, custom, legacy, disabled, and invalid provider trust states;
- verified, partial, unsupported, unverified, and failed capability states;
- `custom-chromium` as the generic local-import path without reviewed fingerprint claims;
- compatibility definitions for legacy `native-chromium` and `patched-chromium` records;
- mandatory source, license, platform, architecture, version, executable, and provenance fields for reviewed providers;
- fail-closed validation for advanced unsupported or unverified fingerprint configuration;
- Provider-derived desktop bootstrap data rather than a hard-coded UI provider list;
- frontend trust and capability labels with unsupported controls disabled;
- managed-kernel binary identity that keeps integrity and reviewed trust separate;
- explicit provider predecessor, replacement, and rollback policy;
- tests for legacy compatibility, reviewed-candidate representation, trust boundaries, binary identity, failure behavior, launch behavior, and frontend states;
- updated kernel registry documentation.

M4.1 deliberately does not select or claim a production reviewed browser provider. A custom or legacy binary can remain integrity-verified without gaining reviewed capability status.

## Current milestone

### M4.2 — Real-Browser Evidence Harness

Current implementation issue: #20

The first M4.2 task is to implement a versioned, local evidence harness that observes a Veilium-managed real browser session through controlled top-level, iframe, and worker contexts and stores a private redacted report.

Issue #20 controls the exact scope, privacy boundaries, failure paths, supported observations, tests, and non-scope. It remains blocked until PR #19 merges.

## Active prohibitions

Do not:

- add live external exit-IP, WebRTC/STUN, or delegated-domain DNS probes before M4.4;
- implement final window/viewport/DPR correction policy before M4.3;
- assign reviewed status to a provider without exact provider, binary, platform, and real-browser evidence;
- add new proxy protocols, transports, or proxy-pool operations;
- begin cookie, extension, full migration, Launch API, MCP, sync, or release work;
- copy source from reference browsers or kernels;
- collect arbitrary page contents, browsing history, cookies, tokens, credentials, or private proxy configuration;
- include unrelated refactors or broad UI redesign.

## Known risks

- no production provider is reviewed yet;
- existing `patched-chromium` profiles with advanced settings remain readable but cannot claim reviewed support until an exact reviewed replacement and evidence exist;
- integrity status and provider trust remain separate;
- M4.2 evidence must not mutate browser profiles or become a general browsing-inspection interface;
- macOS remains unclaimed until a real validation path exists.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

Every product PR must also pass the protected-branch Windows, Linux, desktop-build, frontend, and official-adapter checks.

## Handoff

After PR #19 merges, read Issue #20 and create one scoped Draft PR for M4.2 only. Do not reopen M4.1 design, add a reviewed provider from upstream claims, or begin later milestone work without a separately reviewed planning change.
