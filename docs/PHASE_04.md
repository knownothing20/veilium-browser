# Phase 4 — Verified Browser Capability and Evidence

Status: Active
Phase: Phase 4
Owner decision required: No
Product implementation allowed: Yes

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

Reviewed trust is granted only to an exact Provider revision, source, license/provenance, OS, architecture, browser version, archive identity, executable identity, and review record.

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

## Corrective milestone — First Exact Reviewed Provider Path

Issue #28 completed the first closing review with a `Blocked` result: M4.1–M4.4 passed, but the catalog contains no exact `reviewed` browser Provider.

Issue #30 is the only authorized product task.

It may add exactly one pinned official Windows x64 Chromium Snapshot Provider with:

- one fixed snapshot Revision and official HTTPS archive URL;
- archive size and SHA-256;
- bounded archive layout and executable path;
- browser-reported version;
- executable size and SHA-256;
- Provider revision, source, Chromium license/provenance notes, review time, and limitations;
- fail-closed install/import, verification, missing/modified/incompatible tests, and rollback;
- M4.2 identity, M4.3 window/consistency, and M4.4 Network Evidence from the same exact binary;
- exact compatibility entries that do not generalize to another revision, platform, or binary.

Normal application behavior must not resolve `LAST_CHANGE`, select a moving latest build, bundle the archive, silently replace an intact reviewed binary, or manufacture unsupported stock-Chromium fingerprint capabilities.

The frozen corrective boundary is documented in `docs/PHASE_04_CORRECTIVE_PLAN.md`.

## Supported-platform policy

- The corrective reviewed path is Windows amd64 only.
- Linux remains validated for existing generic Evidence fixtures but is not part of this first reviewed Provider claim.
- macOS remains unclaimed.
- Other revisions, architectures, Providers, and platforms require separate reviewed evidence.

## Explicit non-scope

Do not add:

- a second Provider or platform;
- Chromium source building or patching;
- moving latest downloads or background updates;
- new fingerprint controls;
- proxy protocols, pools, rotation, or batch operations;
- cookies, extensions, backup/migration, public API, MCP, sync, or release work;
- Phase 5 planning or implementation;
- broad UI redesign or unrelated refactors.

## Data, privacy, and recovery rules

- Evidence and Provider schemas remain versioned.
- Existing custom and legacy records remain readable and unpromoted.
- Downloads and extraction are bounded and reject traversal, links, unexpected layout, hash, size, version, and executable mismatches.
- Evidence contains no cookies, tokens, browsing content, proxy secrets, or private runtime configuration.
- Failed verification disables the reviewed claim without deleting Profile data.
- No reviewed Provider revision silently replaces another; rollback preserves an intact prior revision.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The corrective PR must also pass protected Windows exact-download, archive/executable verification, Provider, identity Evidence, managed-window, Network Evidence, frontend, Wails, privacy, and rollback checks.

## Phase exit criteria

Phase 4 may return to `Closing` only when:

- one exact `reviewed` Windows amd64 Chromium Snapshot Provider exists;
- its archive and executable identities are pinned and fail closed;
- the same exact binary completes applicable identity, window/consistency, and network Evidence;
- compatibility references the exact Provider revision, browser version, platform, architecture, binary identity, Evidence runs, limitations, and review time;
- custom and legacy Providers remain unchanged;
- all protected checks pass;
- unresolved limitations and Phase 5 candidates are recorded.

Phase 4 becomes `Done` only through a separate closure decision that reruns every exit gate and updates `ROADMAP.md` and `STATUS.md`.

## Current authorized work

The only authorized task is Issue #30, **establish first reviewed official Chromium Provider path**. After it merges, return Phase 4 to `Closing` and rerun the closing review. Phase 5 remains blocked.