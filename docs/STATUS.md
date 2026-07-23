# Current Project Status

Last updated: 2026-07-23
Application version: 0.15.0-dev
Main baseline SHA: ffcf25d94cd821c82f07cc49fc61130d3e02fcdb
Current phase: Phase 5
Current milestone: Consolidated M5.3, M5.4, and M5.5 product completion
Current task: Execute full local validation and repair any build or runtime issues on PR #59 branch `agent/handoff-m5-3`
Current implementation stage: M5.3/M5.4 services and desktop surfaces are implemented; the first complete source-level M5.5 Chinese browser workspace implementation is committed and awaits executable validation

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

## M5.5 source implementation committed

The owner directed implementation of `docs/CHINESE_BROWSER_WORKSPACE_DEVELOPMENT_PLAN.md` in PR #59. The branch now contains:

- typed `zh-CN` messages, an English fallback dictionary, dictionary-shape test, Chinese status/size/date helpers, and `zh-CN` document metadata;
- Chinese CJK typography, larger readable table and action sizes, responsive sidebar behavior, and product-specific UI overrides;
- a five-item task-oriented primary navigation: browser environments, proxy/network, data/recovery, batch management, and settings;
- advanced navigation for runtime sessions, browser kernels, and credentials;
- browser environments as the default page with metrics, search, group filter, first-use guidance, visible “打开浏览器”/“关闭浏览器”, edit, and a progressive “更多” menu;
- an internal SVG icon set replacing Unicode characters as the only operation signal in primary workflows;
- a five-section Chinese create/edit environment flow covering basic information, browser/kernel, identity, network, and review, with Provider details progressively disclosed;
- official and custom browser-kernel management localized in Chinese;
- proxy adapter installation/import, credential storage, runtime sessions, launch details, proxy diagnostics, browser Evidence, Network Evidence, and identity consistency surfaces localized in Chinese;
- data/recovery, local snapshots, restore, archive, trash, permanent-delete confirmation, portability/import/export, and template flows localized in Chinese;
- the floating Multi-Profile dock removed from application rendering and integrated as a normal full-page batch-management workspace;
- bulk operation history, recoverable lifecycle, metadata, health, portable export, managed storage, fixed locations, and template maintenance localized in Chinese;
- optional managed-window frontend fields aligned with the existing consistency UI so TypeScript can represent those values without changing translation or persistence semantics.

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

- source-level review of changed TypeScript/TSX interfaces, imports, enum values, routes, Wails method references, and component composition;
- localization dictionary structure test added;
- existing backend contracts and Wails method names intentionally reused;
- no GitHub workflow change requested or executed;
- PR remains open and unmerged.

Not yet verified and must not be claimed as passed:

- full frontend TypeScript typecheck;
- frontend unit tests and production build;
- Go formatting, vet, unit/race tests, and builds;
- Wails development startup;
- Windows amd64 package build;
- real Chromium start/stop/cleanup after the M5.5 changes;
- manual Chinese UI smoke testing at 1366×768 and 1920×1080;
- complete regression of recovery, portability, template, batch, proxy, and Evidence behavior;
- final proof that changing application-language presentation does not alter any Profile identity value.

## Exact next task

1. fetch the current branch in the owner's Windows development environment;
2. run `cd frontend && npm run typecheck && npm test && npm run build` and repair every reported issue;
3. run repository Go formatting, vet, tests, race tests, and builds without enabling GitHub Actions;
4. run `wails dev`, exercise the primary Chinese browser-environment journey, and fix layout or runtime integration issues;
5. build Windows amd64 and manually test kernel installation/import, environment creation, proxy diagnostics, real Chromium start/stop, recovery, portability, templates, batch management, and persistence after restart;
6. update this file and PR #59 with actual command output and manual-test results;
7. do not merge PR #59 until executable validation and final scope/security review are complete.
