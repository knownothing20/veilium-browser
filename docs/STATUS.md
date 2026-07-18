# Current Project Status

Last updated: 2026-07-18
Application version: 0.13.0-dev
Main baseline SHA: 1773ec77dbe2c9fcbeaecbc89bf206305ec16644
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.2 — Real-Browser Evidence Harness
Current task: Complete Issue #20 in Draft PR #21 and make every required check pass

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

## Current milestone

### M4.2 — Real-Browser Evidence Harness

Current implementation issue: #20
Current Draft PR: #21

Implemented in the Draft branch so far:

- versioned evidence run, observation, context, run-status, and observation-status records;
- validation bounds for identifiers, timestamps, terminal states, observations, values, and limitations;
- cryptographically random local evidence run IDs;
- private evidence directory and report permissions;
- atomic write-once JSON reports with strict decoding and a 1 MiB bound;
- explicit retention, maximum-run pruning, profile filtering, get, list, and independent deletion;
- rejection of symlinked storage roots and non-regular report files;
- tests for private permissions, round trips, expiry, maximum-run pruning, duplicate writes, symlink rejection, and value bounds.

Work still required before M4.2 can merge:

1. implement the controlled loopback evidence page and one-time submission token;
2. collect allowlisted top-level, same-origin iframe, and worker observations without arbitrary page inspection;
3. implement bounded CDP target creation, navigation, closure, timeout, cancellation, and browser-exit handling;
4. compare declared profile values, provider contract states, binary identity, and observed values;
5. persist terminal passed, partial, failed, cancelled, and incomplete reports;
6. integrate the evidence manager into desktop service and Wails bindings;
7. add an explicit desktop action and readable local report view;
8. add real-browser integration on Windows and Linux where the exact test binary is available;
9. document evidence privacy, retention, limitations, and operation;
10. make Governance, Go, frontend, Windows, Linux, desktop, adapter, and real-browser checks pass;
11. update this handoff with the exact first M4.3 task.

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

- no production provider is reviewed yet;
- evidence from custom or legacy providers may describe observations but cannot create reviewed provider status;
- evidence collection must not mutate browser profiles or become a general browsing-inspection interface;
- local paths and normalized browser identity values require bounded, private storage;
- browser exit, timeout, malformed submission, storage failure, and cancellation must preserve profile data and clean temporary resources;
- macOS remains unclaimed until a real validation path exists.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

Every product PR must also pass the protected-branch Windows, Linux, desktop-build, frontend, official-adapter, and relevant real-browser checks.

## Handoff

Continue only in Issue #20 and Draft PR #21. Implement the local controlled evidence chain in dependency order, use tests and CI to drive failure-path fixes, and do not begin M4.3 or M4.4 behavior in this PR.
