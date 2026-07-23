# Current Project Status

Last updated: 2026-07-23
Application version: 0.15.0-dev
Main baseline SHA: ffcf25d94cd821c82f07cc49fc61130d3e02fcdb
Current phase: Phase 5
Current milestone: Consolidated M5.3, M5.4, and M5.5 product completion
Current task: Implement and validate the Chinese browser workspace on PR #59 branch `agent/handoff-m5-3`
Current implementation stage: M5.3/M5.4 services and desktop surfaces are implemented; M5.5 localization, task navigation, browser-environment workspace, guided editor, and batch-page integration are in active implementation

## Operational rule

PR #59 and branch `agent/handoff-m5-3` are the only remaining Phase 5 development path. No additional development branch, pull request, temporary issue, handoff PR, closing-review PR, or workflow is authorized.

GitHub Actions remain unavailable and must not be created, enabled, modified, manually triggered, or rerun. Connector-side static work may continue in concentrated commits, but compilation, tests, Wails execution, Windows packaging, and manual smoke testing remain unverified until actually run in a suitable environment.

## Completed frozen baseline

### M5.1 — Lifecycle foundation

Merged and frozen: versioned lifecycle records, authoritative operation journal, per-Profile locks, active-session blockers, cancellation state, per-item results, bounded storage inventory, startup reconciliation, Desktop integration, and lifecycle UI.

### M5.2 — Safe local recovery

Merged and frozen: verified same-machine snapshots, restore to a new identity, archive/unarchive, recoverable trash, exact restore-trash, explicit permanent deletion, conservative reconciliation, and Local recovery workspace.

## Implemented in PR #59 before M5.5

- strict portable Profile definitions with canonical integrity and exclusion of secrets, browser data, binaries, local IDs/paths, runtime data, logs, and Evidence;
- import preview, dependency remapping, new-identity default, advanced preserve-identity mode, and private templates;
- bounded bulk metadata, recoverable lifecycle, portable export, health refresh, storage inventory, manual repair plans, operation history, cancellation, and redacted report export;
- visible Desktop surfaces for M5.3 and M5.4;
- existing M5.1/M5.2 locks, journal, idempotency, per-item results, rollback, and recovery state reused rather than replaced.

## M5.5 authorized productization

The owner directed implementation of `docs/CHINESE_BROWSER_WORKSPACE_DEVELOPMENT_PLAN.md` in PR #59. Authorized work includes:

- typed `zh-CN` messages with an English fallback;
- Chinese document metadata, typography, formatting, common states, confirmations, and high-frequency errors;
- browser environments as the default workspace;
- primary navigation for environments, network, recovery, batch management, and settings;
- advanced pages for runtime sessions, browser kernels, and credentials;
- visible “打开浏览器” and “关闭浏览器” actions;
- simplified environment rows with technical diagnostics moved behind “更多” and technical details;
- guided Chinese create/edit environment flow using existing Profile and Provider contracts;
- Multi-Profile tools integrated as a normal page instead of a floating dock;
- responsive desktop layout for 1366×768 and 1920×1080;
- no management-language coupling to Profile identity parameters.

## Current implementation batch

The active batch covers:

1. governance alignment for M5.5;
2. typed localization dictionaries and dictionary-shape test;
3. Chinese application shell, five-item primary navigation, and advanced navigation;
4. browser-environment default page, metrics, search/filter, Chinese empty states, and primary open/close actions;
5. Chinese guided Profile editor with advanced Provider details progressively disclosed;
6. Multi-Profile floating dock removal and full-page batch-management integration;
7. CJK typography, readable table/action sizes, responsive sidebar, and product UI overrides;
8. HTML language/title and startup/error fallback localization.

## Frozen boundaries

- browser contents remain opaque and separate from Profile metadata;
- vault secrets remain local and non-portable;
- local IDs and absolute paths are not portable identities;
- UI text cannot manufacture Provider trust, capability support, compatibility, health, or Evidence;
- bulk lifecycle exposes recoverable trash only and never bulk permanent deletion;
- bulk export never overwrites an existing file and never stores destination paths in lifecycle item results;
- health refresh, storage inventory, and repair plans remain observational;
- application language remains separate from Profile language, timezone, platform, and fingerprint values;
- unsupported, unsafe, contradictory, missing, modified, or unverifiable state fails closed or remains explicitly limited.

## Validation truth

Completed in the connector environment:

- source-level review of the proposed TypeScript/TSX syntax;
- localization dictionary structure test added;
- existing backend contracts and Wails method names intentionally reused;
- no workflow change requested.

Not yet verified and must not be claimed as passed:

- full frontend TypeScript typecheck;
- frontend unit tests and production build;
- Go formatting, vet, unit/race tests, and builds;
- Wails development startup;
- Windows amd64 package build;
- real Chromium start/stop/cleanup after the M5.5 changes;
- manual Chinese UI smoke testing and window-size review;
- complete regression of recovery, portability, template, batch, proxy, and Evidence behavior.

## Exact next task

1. commit the active M5.5 implementation batch to PR #59;
2. inspect the complete PR diff for TypeScript interface, route, CSS, security, and localization regressions;
3. continue component-level Chinese localization for recovery, adapter, credential, runtime, diagnostics, Evidence, portability, templates, and batch child workspaces;
4. run frontend typecheck/tests/build and fix every issue found;
5. run Go and Wails validation plus Windows manual smoke testing;
6. update this file with actual results and do not merge PR #59 until executable validation is complete.
