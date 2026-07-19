# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: 759dd7ab6689c244e28ce9d09b63e9f2bac1878c
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: Phase 4 Done — Phase 5 Planning Preparation
Current task: Define Phase 5 profile lifecycle and operations scope in Issue #37 without implementing product code

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`.

Phase 4 is `Done` and frozen. Work only on Issue #37 and the dedicated Phase 5 planning documents and pull request. Phase 5 remains `Planned`; product implementation is blocked until a separate activation decision explicitly permits it.

## Phase 4 closure result

Issue #35 recorded **Pass** against:

- implementation baseline `49ae2de6cb652d789c97aa961c0007513362bb6f`;
- Closing-state baseline `759dd7ab6689c244e28ce9d09b63e9f2bac1878c`.

The delivered Phase 4 baseline includes:

- M4.1 Provider Contract v2 — PR #19;
- M4.2 Real-Browser Evidence Harness — PR #21;
- M4.3 Identity and Window Consistency — PR #24;
- M4.4 Network Evidence and Compatibility Matrix — PR #27;
- First Exact Reviewed Provider Path — PR #34.

## Exact reviewed Provider result

- one exact reviewed official Chromium Snapshot Provider exists for Windows amd64;
- Provider `official-chromium-snapshot-win64` revision `1` uses Chromium `152.0.7960.0`, Snapshot revision `1664436`;
- archive, `chrome.exe`, and the complete 261-file Package Tree identities are embedded and immutable;
- installation is explicit, license-acknowledged, bounded, fail-closed, atomic, idempotent, and non-updating;
- reviewed Providers cannot use the generic single-file import path;
- the same managed binary passed identity, managed-window/consistency, and controlled Network Evidence;
- changing a package dependency downgrades the package to `modified`;
- compatibility rejects nearby versions, Linux, arm64, custom trust, and every other non-exact combination;
- unsupported stock Chromium advanced fingerprint controls remain unsupported.

## Validation and recorded CI risk

Governance, Go quality, frontend, Windows/Linux Wails, official adapter, Linux generic-browser, exact Windows reviewed-browser, Network Evidence, compatibility, tamper, and build checks passed.

The first Closing-state Windows Evidence attempt installed and verified the exact package, then encountered a GitHub Runner temporary-directory Chromium Sandbox access denial. The identical Job passed on a fresh Runner without any code, binary, assertion, or flag change. This is recorded as a non-blocking CI-environment reliability risk and does not broaden the Provider support claim.

## Known limitations to retain

- reviewed browser trust covers only the exact Windows amd64 Snapshot package;
- Linux browser CI validates the generic Evidence harness, not a reviewed Linux Provider;
- macOS and other architectures are unclaimed;
- controlled Network Evidence proves the collection path for a synthetic direct route, not every proxy, STUN service, delegated DNS zone, or user network;
- Chromium Snapshot provenance and SHA-256 pins do not provide publisher signing, transparency logs, or reproducible-build proof;
- stock Chromium advanced fingerprint overrides remain unsupported;
- future Provider revisions require a new immutable identity and Evidence chain;
- Phase 5 and Phase 6 deferrals remain unauthorized until their dedicated planning and activation work.

## Current planning scope

Issue #37 must define:

1. the Phase 5 user outcome and non-goals;
2. the priority and dependency order for profile lifecycle and day-to-day operations;
3. portable, machine-bound, secret, Provider-specific, and excluded data;
4. schema, compatibility, migration, rollback, cancellation, interruption, and recovery policy;
5. Windows/Linux/macOS support boundaries;
6. security, privacy, licensing, clean-room, and Provider/Evidence preservation rules;
7. the validation matrix and Phase 5 exit gates;
8. a dedicated `docs/PHASE_05.md` plus ROADMAP and STATUS updates.

No Phase 5 product code, schema migration, cookie or extension implementation, backup format, batch operation, automation API, MCP, sync, release, second reviewed Provider, proxy expansion, or unrelated UI redesign is authorized by Issue #37 alone.

## Required planning validation

```bash
python scripts/check_project_governance.py
make check
```

The Phase 5 planning pull request must remain documentation-only unless a separately reviewed activation decision changes `Product implementation allowed` to `Yes`.

## Handoff

Complete Issue #37 through a dedicated Draft planning pull request. Keep Phase 5 `Planned` or `Planning` and product implementation blocked until the scope, contracts, milestones, validation, and exit gates are reviewed and explicitly activated.