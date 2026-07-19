# Phase 4 — Verified Browser Capability and Evidence

Status: Closing
Phase: Phase 4
Owner decision required: No
Product implementation allowed: No

## User outcome

A user can select a reviewed browser Provider, configure only supported capabilities, launch a managed profile, and receive local evidence showing whether the declared identity, managed window, and selected network route were observed in the real browser.

Capability states remain explicit:

- `verified` — exact reviewed Provider/version/platform/binary evidence passed;
- `partial` — applicable evidence exists with documented limitations;
- `unsupported` — the exact Provider path does not support the capability;
- `unverified` — custom or legacy path without reviewed evidence;
- `failed` — evidence contradicts or invalidates the claim.

A filename, launch flag, source URL, successful start, mock, or hosted CI browser alone is not reviewed evidence.

## Provider policy

Reviewed trust is granted only to an exact Provider revision, source, license/provenance, OS, architecture, browser version, archive identity, executable identity, complete Package Tree identity, and review record.

Custom and legacy Providers remain usable only through their declared generic behavior. They cannot inherit reviewed claims from another binary or Evidence run.

Detailed logical contracts live in `docs/PHASE_04_CONTRACTS.md`.

## Delivered milestones

### M4.1 — Kernel Provider Contract v2

Delivered in PR #19:

- versioned Provider definitions and trust states;
- exact binary identity and integrity metadata;
- explicit capability states;
- legacy compatibility and rollback/failure behavior;
- service and UI fail-closed enforcement.

### M4.2 — Real-Browser Evidence Harness

Delivered in PR #21:

- controlled loopback Evidence Collector and CDP Target lifecycle;
- top-level, iframe, and worker identity observations;
- private versioned reports with retention and deletion;
- exact Provider/binary/Profile comparison;
- Windows and Linux real-browser fixtures.

### M4.3 — Identity and Window Consistency

Delivered in PR #24:

- versioned consistency rules and freshness inputs;
- managed window application and observed state;
- fail-closed launch preflight;
- derived Profile health and desktop inspection;
- Windows and Linux real-window validation.

### M4.4 — Network Evidence and Compatibility Matrix

Delivered in PR #27:

- explicit replaceable or self-hostable ProbeSets;
- browser-observed Exit IP, bounded WebRTC/STUN summaries, and delegated DNS evidence;
- privacy-preserving route identity;
- Profile health integration;
- exact compatibility contracts and desktop reports;
- protected Windows and Linux real-browser Network Evidence fixtures.

Supporting documents:

- `docs/NETWORK_EVIDENCE.md`
- `docs/COMPATIBILITY_MATRIX.md`
- `docs/IDENTITY_WINDOW_CONSISTENCY.md`

### Corrective — First Exact Reviewed Provider Path

Delivered in PR #34:

- one immutable official Chromium Snapshot for Windows amd64;
- exact Provider revision, Chromium version, Snapshot revision, source, license/provenance, review time, and limitations;
- exact archive size/SHA-256, executable path/size/SHA-256, and complete 261-file Package Tree identity;
- explicit user acknowledgement and no moving-latest, bundling, background update, or silent replacement;
- bounded secure extraction, atomic activation, idempotency, rollback, and complete-package verification;
- generic single-executable import blocked for reviewed Providers;
- same managed `chrome.exe` used for M4.2 identity, M4.3 window/consistency, and M4.4 controlled Network Evidence;
- dependency tamper downgrades the package to `modified`;
- compatibility rejected outside the exact Provider revision, browser version, Windows amd64 platform, architecture, trust, and binary identity.

The reviewed Provider contract and operational boundary are documented in `docs/OFFICIAL_CHROMIUM_PROVIDER.md` and `docs/PHASE_04_CORRECTIVE_PLAN.md`.

## Supported-platform policy

- The reviewed Chromium Provider claim is Windows amd64 only.
- Linux remains validated for the generic Evidence harness and official adapter smoke tests but is not part of the reviewed browser Provider claim.
- macOS remains unclaimed.
- Other revisions, architectures, Providers, and platforms require separate reviewed evidence.

## Explicit non-scope

Do not add during Closing Review:

- a second Provider or platform;
- Chromium source building or patching;
- moving-latest downloads or background updates;
- new fingerprint controls;
- proxy protocols, pools, rotation, or batch operations;
- cookies, extensions, backup/migration, public API, MCP, sync, or release work;
- Phase 5 planning implementation or product implementation;
- broad UI redesign or unrelated refactors.

## Data, privacy, licensing, and recovery rules

- Evidence and Provider schemas remain versioned.
- Existing custom and legacy records remain readable and unpromoted.
- Downloads and extraction are bounded and reject traversal, links, unexpected layout, hash, size, version, executable, and Package Tree mismatches.
- Evidence contains no cookies, tokens, browsing content, proxy secrets, or private runtime configuration.
- Failed verification disables the reviewed claim without deleting Profile data.
- No reviewed Provider revision silently replaces another; rollback preserves an intact prior revision.
- Chromium Snapshot provenance and SHA-256 pins do not claim publisher signing, transparency logs, or reproducible-build proof.
- Stock Chromium advanced fingerprint overrides remain unsupported.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The closure baseline must also pass protected Windows exact-download, archive/executable/Package Tree verification, Provider, identity Evidence, managed-window, Network Evidence, frontend, Wails, privacy, compatibility, and dependency-tamper checks.

## Phase exit criteria

Issue #35 must verify that:

- Provider Contract v2 and legacy compatibility are frozen;
- one exact reviewed Windows amd64 Chromium Snapshot Provider exists;
- archive, executable, and complete-package identities are pinned and fail closed;
- the same exact managed binary completes applicable identity, window/consistency, and Network Evidence;
- compatibility references the exact Provider revision, browser version, platform, architecture, binary identity, Evidence, limitations, and review time;
- custom and legacy Providers remain unchanged and cannot inherit reviewed claims;
- unsupported advanced stock-Chromium controls remain fail-closed;
- unsafe, unsupported, modified, missing, stale, contradictory, and incompatible states remain explicit;
- unresolved risks and deferred work are recorded;
- the complete governance and technical validation matrix passes on the closure baseline.

Phase 4 becomes `Done` only through a separate closure decision PR after Issue #35 records a Pass result and updates `docs/ROADMAP.md` and `docs/STATUS.md`.

## Current authorized work

The only authorized task is Issue #35, **Phase 4 final closing evidence and exit gates**. Product implementation is blocked. Phase 5 remains blocked until a dedicated closure PR marks Phase 4 `Done`; Phase 5 product work then remains blocked until a separate planning and activation PR is reviewed.