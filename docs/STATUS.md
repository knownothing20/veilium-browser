# Current Project Status

Last updated: 2026-07-19
Application version: 0.14.0-dev
Main baseline SHA: 7306215085b578755d4980180edb9f451e5a9f14
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: M4.3 — Identity and Window Consistency
Current task: Finish review and merge of Issue #22 in Draft PR #24

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. Work only on the current issue and milestone.

M4.4 Issue #25 exists only as a blocked handoff. It must not begin until PR #24 is merged and Issue #25 is explicitly activated.

## Delivered baseline

M4.1 and M4.2 are complete.

M4.1 established Provider Contract v2, explicit trust and capability states, legacy compatibility, managed binary identity, and fail-closed advanced configuration.

M4.2 was delivered by Issue #20 and PR #21, squash-merged as `094fea4f03c5a87e37f69a4868fd26e609673c6e`. It added controlled loopback real-browser evidence, private bounded reports, desktop report views, and Required Windows/Linux collection checks.

M4.2 does not grant reviewed status to a production browser Provider.

## Current milestone

### M4.3 — Identity and Window Consistency

Current implementation issue: #22
Current Draft PR: #24

The Draft branch now contains:

- optional explicit `windowWidth`, `windowHeight`, and `deviceScaleFactor` fields;
- backward-compatible screen-to-window fallback without rewriting existing profiles;
- versioned WindowPlan, observed WindowState, consistency Result, Check, Health, WindowSource, and EvidenceInput contracts;
- shared preflight enforcement for profile create/update, launch-plan generation, and runtime start;
- effective WindowPlan launch arguments rather than treating screen size as the browser window contract;
- deterministic consistency input digests stored on new Evidence Runs;
- conservative freshness invalidation for profile, Provider, binary, runtime, harness, and rules changes;
- evidence-derived `healthy`, `degraded`, `blocked`, and `unknown` profile health;
- a read-only desktop health service and Profile-row report view;
- explicit user-controlled window width, height, and DPR editing through the normal profile update path;
- a bounded loopback CDP window controller that permits only get-window, set-bounds, and readback commands;
- one Runtime Supervisor wrapper shared by Direct, built-in bridge, Xray, and sing-box launch paths;
- fail-closed browser shutdown when managed-window application fails;
- observed-window cleanup on stop and application shutdown;
- unit, desktop, lifecycle, frontend, Windows, Linux, and real-Chromium controlled-window tests;
- `docs/IDENTITY_WINDOW_CONSISTENCY.md`.

## Remaining M4.3 merge gates

1. obtain passing final Governance and all protected CI checks;
2. remove the temporary read-only M4.3 diagnostic workflow;
3. verify the final compare contains no diagnostic or autofix workflow;
4. confirm no unresolved review thread;
5. update PR #24 to the final delivery description;
6. mark PR #24 ready and complete a protected squash merge.

## Next milestone — blocked

Issue #25 defines M4.4 — Live Browser Network Evidence and Compatibility Matrix.

It remains blocked until PR #24 merges. After activation, M4.4 owns browser-observed exit IP, controlled WebRTC/STUN and delegated DNS route evidence, route-health integration, and generated exact-combination compatibility records.

## Active prohibitions

Do not:

- begin M4.4 external network evidence before PR #24 merges;
- claim reviewed Provider status without exact applicable evidence;
- silently rewrite or auto-correct profiles;
- add new proxy protocols or pool operations;
- begin cookie, extension, migration, public API, MCP, sync, or release work;
- copy code from reference browsers or kernels;
- expose browser control beyond the existing local safety boundary;
- include unrelated refactors or broad UI redesign.

## Known risks

- no production browser Provider is reviewed yet;
- hosted headless/window fixtures prove only their exact controlled environments, not general desktop compatibility;
- Windows and Linux decorations and display scaling require bounded platform-aware tolerances;
- freshness invalidation must remain conservative;
- existing profiles and M4.2 Evidence reports must remain readable;
- macOS remains unclaimed.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

The M4.3 PR must also pass the protected Windows, Linux, frontend, Wails, official-adapter, and controlled real-browser window checks.

## Handoff

Until PR #24 merges, continue only in Issue #22 and Draft PR #24. After merge, activate Issue #25 and make it the single authorized product task.