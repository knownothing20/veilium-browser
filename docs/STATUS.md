# Current Project Status

Last updated: 2026-07-19
Application version: 0.15.0-dev
Main baseline SHA: dcfcee9e4c8b8587ae0c8c44a63103cb0c5c5d6c
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.4 — Live Browser Network Evidence and Compatibility Matrix
Current task: Finish review and protected merge of Issue #25 in Draft PR #27

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. Work only on Issue #25 and PR #27.

M4.4 is the final Phase 4 implementation milestone. After it merges, the only authorized next action is a separate Phase 4 Closing Review. Phase 5 remains blocked.

## Delivered baseline

M4.1, M4.2, and M4.3 are complete.

- M4.1 PR #19 established Provider Contract v2, exact managed binary identity, explicit trust/capability states, and fail-closed legacy compatibility.
- M4.2 PR #21, merged as `094fea4f03c5a87e37f69a4868fd26e609673c6e`, established controlled local real-browser Evidence, private reports, desktop inspection, and Windows/Linux fixtures.
- M4.3 PR #24, merged as `139907936179ee61d4fcd82b19125c1535bb8e2a`, established versioned consistency rules, Evidence freshness, managed browser windows, derived Profile health, desktop controls, and Windows/Linux real-window validation.

No production browser Provider is marked reviewed solely by these milestones.

## M4.4 delivered in PR #27

- independent versioned Network Evidence records tied to existing real-browser Evidence;
- explicit Exit-IP, WebRTC/STUN, and delegated-DNS observations;
- explicit replaceable or self-hostable ProbeSet configuration with no hidden public default;
- HTTPS or loopback-only HTTP endpoint policy and bounded response limits;
- route classification for Direct, HTTP, HTTPS, SOCKS5, local-auth bridge, Xray, and sing-box;
- privacy-preserving SHA-256 route identity without storing original proxy URLs or credentials;
- private atomic write-once report storage, retention, deletion, cancellation, timeout, and shutdown cleanup;
- loopback-only one-shot Collector with Host, Origin, content type, size, strict JSON, and CSP enforcement;
- controlled CDP Target lifecycle bound to the selected ready managed browser session;
- browser-observed Exit-IP, bounded STUN summaries, delegated-DNS trigger and result collection;
- reconciliation that fails WebRTC public-IP mismatches and degrades missing or unavailable evidence;
- Profile health integration without silently changing the Profile or route;
- desktop ProbeSet configuration, run, report, deletion, and compatibility-matrix surfaces;
- exact Provider/revision/browser/OS/architecture/binary/capability matrix contracts;
- custom and legacy Providers remain unable to receive reviewed verified status;
- controlled unit, privacy, storage, lifecycle, frontend, and real-Chromium tests;
- `docs/NETWORK_EVIDENCE.md` and `docs/COMPATIBILITY_MATRIX.md`.

## Remaining M4.4 merge gates

1. apply standard Go formatting and the final reviewed small fixes;
2. add the real Chromium Network Evidence fixture to protected Windows and Linux CI;
3. remove all temporary autofix or diagnostic workflows;
4. pass Governance and every protected CI job;
5. verify the final compare contains only M4.4 code, tests, CI, UI, and documents;
6. confirm no unresolved review thread;
7. update PR #27 to its final delivery description;
8. mark PR #27 ready and complete a protected squash merge.

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

## Known limitations

- controlled Required CI proves only the exact hosted Chromium, runner OS, architecture, and synthetic loopback Exit-IP fixture;
- production STUN and delegated-DNS claims require accepted evidence from the exact configured ProbeSet and runtime combination;
- unavailable probes remain explicit and cannot produce optimistic success;
- custom and legacy Provider trust remains conservative;
- macOS remains unclaimed.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The final PR must also pass protected Windows, Linux, frontend, Wails, official-adapter, controlled-probe, privacy, and real-browser checks.

## Handoff

After PR #27 merges, create a separate Phase 4 Closing Review issue and planning/review PR. Do not silently enter Phase 5.