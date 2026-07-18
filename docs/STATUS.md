# Current Project Status

Last updated: 2026-07-19
Application version: 0.14.0-dev
Main baseline SHA: 1773ec77dbe2c9fcbeaecbc89bf206305ec16644
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.2 — Real-Browser Evidence Harness
Current task: Complete final review and required checks for Issue #20 / Draft PR #21

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
- repository governance, protected `main`, required pull requests, required checks, force-push protection, and deletion protection;
- M4.1 Provider Contract v2, explicit trust/capability states, legacy compatibility, managed binary identity, and fail-closed advanced configuration.

## Phase 4 approved outcome

At Phase 4 completion, users can select a reviewed browser-kernel provider, configure only supported capabilities, launch a profile, and receive local evidence showing whether the declared identity and selected network route were observed in the real browser session.

The authoritative phase scope, milestones, non-goals, platform policy, validation, rollback rules, and exit criteria are in `docs/PHASE_04.md`. The logical provider, capability, evidence, compatibility, and health contracts are in `docs/PHASE_04_CONTRACTS.md`.

## M4.2 delivery

M4.2 is implemented in Issue #20 and Draft PR #21. The branch now contains:

- versioned Evidence Run, Observation, context, run-status, and observation-status records;
- cryptographically random run IDs and strict validation bounds;
- private atomic write-once JSON reports, 1 MiB report limit, 30-day default retention, maximum-count pruning, profile filtering, and independent deletion;
- rejection of symlinked roots/files, non-regular files, malformed JSON, duplicate IDs, oversized records, and unsupported schemas;
- a loopback-only Collector with random one-time paths, Host/Origin checks, strict CSP, no-store headers, bounded request bodies, strict decoding, and single-use submission;
- allowlisted top-level, same-origin iframe, and same-origin Worker collection;
- UA, Client Hints, platform, language, timezone, hardware concurrency, screen, window, viewport, DPR, local WebRTC indicators, and fixed-surface digest observations;
- no external STUN, exit-IP, delegated DNS, arbitrary page, history, cookie, token, credential, download, or private proxy-configuration collection;
- a bounded loopback CDP Target client that can only open the controlled Collector URL and closes the temporary target after collection;
- an evidence Manager bound to the exact profile, ready managed session, Provider Contract revision, and managed binary identity;
- terminal passed, partial, failed, cancelled, and incomplete reports for success, mismatch, timeout, cancellation, browser exit, target/collector failures, and cleanup limitations;
- comparison rules that prevent custom or legacy observations from manufacturing reviewed Provider status;
- application-shutdown cancellation and cleanup;
- desktop Service and Wails bindings for run, cancel, list, get, delete, and active-state operations;
- a desktop profile action and local report view; historical reports remain reviewable while the browser is stopped;
- contract, store, Collector, Target, comparison, Manager, desktop-service, frontend-helper, and UI/build tests;
- real hosted Chromium collection in existing Required Windows and Linux CI jobs, including top-level, iframe, and Worker evidence;
- `docs/REAL_BROWSER_EVIDENCE.md` covering operation, privacy, retention, statuses, cleanup, test boundary, and deferred work.

M4.2 does not grant reviewed status to a production Provider. Real-browser CI proves the controlled collection chain for the exact hosted test binary only.

## Remaining work before M4.2 merge

1. synchronize the desktop application constant with `0.14.0-dev`;
2. make Governance and all seven CI jobs pass on the final commit;
3. confirm there are no temporary diagnostic workflows or unresolved review threads;
4. update PR #21 with final validation evidence;
5. mark PR #21 ready and squash-merge it through protected `main`.

## Next milestone

Issue #22 — M4.3 Identity and Window Consistency — is created but blocked until PR #21 merges. After merge it becomes the single authorized Phase 4 implementation task.

## Active prohibitions

Do not:

- add live external exit-IP, WebRTC/STUN, or delegated-domain DNS probes before M4.4;
- implement final window/viewport/DPR correction policy before M4.3;
- assign reviewed status to a provider without exact provider, binary, platform, and real-browser evidence;
- add new proxy protocols, transports, or proxy-pool operations;
- begin cookie, extension, full migration, Launch API, MCP, sync, or release work;
- copy source from reference browsers or kernels;
- collect arbitrary page contents, browsing history, cookies, tokens, credentials, private proxy configuration, or uncontrolled URLs;
- expose CDP or evidence collection beyond loopback;
- include unrelated refactors or broad UI redesign.

## Known risks

- no production Provider is reviewed yet;
- evidence from custom or legacy Providers may describe observations but cannot create reviewed Provider status;
- M4.3 still owns final identity/window consistency, tolerances, freshness, and profile-health policy;
- M4.4 still owns external network evidence and the generated compatibility matrix;
- macOS remains unclaimed until a real validation path exists.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

Every product PR must also pass the protected-branch Windows, Linux, desktop-build, frontend, official-adapter, and relevant real-browser checks.

## Handoff

Continue only in Draft PR #21 until final checks pass and it is merged. After merge, update Issue #22 from blocked to active and begin one scoped M4.3 Draft PR. Do not add M4.3 or M4.4 product behavior to PR #21.
