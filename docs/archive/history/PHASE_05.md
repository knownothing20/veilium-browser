# Phase 5 — Profile Lifecycle, Day-to-Day Operations, and Desktop Productization

Status: Active
Phase: Phase 5
Owner decision required: No
Product implementation allowed: Yes

## Activation and authority

Phase 5 was approved through Activation Review Issue #40 and activated through PR #47. M5.1 and M5.2 completed their required implementation and closing reviews.

The repository owner subsequently directed that the approved Chinese browser workspace plan be implemented on the only remaining Phase 5 development path. The current authority is therefore:

- PR #59 — `feat: complete Phase 5 product`;
- branch `agent/handoff-m5-3`;
- `docs/STATUS.md`;
- `docs/PHASE_05_CONTRACTS.md` for lifecycle and artifact contracts;
- `docs/PROFILE_PORTABILITY_AND_MULTI_PROFILE.md` for M5.3/M5.4 product behavior;
- `docs/CHINESE_BROWSER_WORKSPACE_DEVELOPMENT_PLAN.md` for M5.5 desktop productization.

No additional development branch, pull request, temporary issue, or workflow is authorized while PR #59 remains the active completion path.

## Milestone record

1. **M5.1 — Lifecycle Contract, Inventory, and Operation Journal — Done**
   - Issue #45;
   - PR #52, squash-merged as `51c469e51ec4cab4ade99efd83c2e6c26145f266`;
   - Closing Review #53 — PASS.
2. **M5.2 — Safe Local Recovery — Done**
   - Issue #54;
   - PR #56, merged to main;
   - Closing Review #57 — PASS.
3. **M5.3 — Portable Profile Definitions and Templates — Implemented in PR #59, executable validation pending.**
4. **M5.4 — Bounded Multi-profile Operations and Storage Management — Implemented in PR #59, executable validation pending.**
5. **M5.5 — Chinese Browser Workspace Productization — Active in PR #59.**

## Phase 5 user outcome

A user can create, start, stop, inspect, preserve, recover, export, import, template, archive, and manage Veilium browser environments through a local-first desktop product that remains truthful about unsupported, limited, unsafe, or unverifiable state.

The default experience is organized around browser environments and the action “打开浏览器”, not around backend modules. Advanced Provider, Kernel, proxy runtime, lifecycle, operation journal, compatibility, and Evidence details remain reviewable through advanced pages and technical details.

## M5.5 authorized scope

M5.5 may implement only the productization work defined by `docs/CHINESE_BROWSER_WORKSPACE_DEVELOPMENT_PLAN.md`, including:

- a typed Simplified Chinese localization foundation and English fallback dictionary;
- `zh-CN` document metadata, Chinese CJK typography, readable sizes, and desktop window adaptation;
- a primary navigation containing browser environments, proxy/network, data/recovery, batch management, and settings;
- advanced access to runtime sessions, browser kernels, credentials, diagnostics, compatibility, and Evidence;
- browser environments as the default page with visible create, open, close, edit, search, filter, status, and repair entry points;
- a guided Chinese create/edit environment flow that reuses existing Profile contracts and backend validation;
- integration of the Phase 5 Multi-Profile floating dock into the normal page hierarchy;
- localized status, date, size, confirmation, empty, loading, and high-frequency error presentation;
- lightweight shared icons, feedback surfaces, and progressive disclosure;
- frontend component refactoring required to make the approved UI testable and maintainable;
- structured error projection only where free-form errors cannot be mapped safely, without exposing secrets.

## M5.5 constraints

- no existing persisted Profile, Kernel, Adapter, Credential, Lifecycle, Snapshot, Portable Definition, Template, Runtime, Compatibility, or Evidence schema may be changed merely to store translated text;
- internal enum and reason-code values remain stable and are translated only at the presentation boundary;
- the management interface language must remain separate from each Profile's browser language, timezone, Accept-Language, platform, and fingerprint configuration;
- frontend controls may explain backend policy but may not replace or bypass backend authorization and preflight;
- hiding technical detail must not hide a launch blocker, recovery requirement, trust limitation, or unsafe state;
- no new Provider or fingerprint capability may be claimed from UI availability;
- all secrets remain in the operating-system credential vault and must not enter Bootstrap data, logs, errors, reports, exports, or translations;
- the Wails window remains an environment manager and does not become an embedded Chromium tabbed browser.

## Frozen security and product boundaries

- local APIs and control surfaces remain loopback-only and authenticated where applicable;
- browser data remains opaque and separate from portable Profile metadata;
- custom and legacy Providers remain unpromoted;
- reviewed browser trust remains restricted to the approved Windows amd64 package;
- artifacts, templates, localization, and UI summaries cannot create reviewed trust, compatibility, health, or applicable Evidence;
- destructive work remains serialized and recoverable boundaries preserve the only healthy copy until verification;
- bulk permanent deletion remains prohibited;
- unsupported, unsafe, contradictory, missing, modified, or unverifiable state fails closed or remains explicitly limited;
- no general filesystem browser, remote API, MCP, cloud sync, proxy rotation, scheduling, bulk browser start, or unrelated automation is authorized.

## Required validation

PR #59 must not merge until applicable validation has actually run:

- repository governance and documentation consistency;
- Go formatting, vet, unit/race tests, and builds;
- frontend typecheck, unit tests, and production build;
- Wails development startup and Windows amd64 build;
- Linux build checks where currently claimed;
- real browser start, readiness, stop, process-tree cleanup, and retained Phase 4 Evidence regressions;
- proxy adapters, credential references, diagnostics, local recovery, portability, templates, bulk operations, storage inventory, and report export smoke tests;
- localization dictionary shape tests and primary-flow component tests;
- manual Simplified Chinese smoke testing at 1366×768 and 1920×1080;
- explicit verification that application-language changes do not mutate Profile identity values;
- final changed-file, security, privacy, licensing, failure-path, and scope review.

GitHub Actions remain paused. Static review may identify issues but cannot replace executable validation or justify a success claim.

## Implementation control

- PR #59 and `agent/handoff-m5-3` are the single implementation path;
- product-code commits must update `docs/STATUS.md` with completed work, validation truth, risks, and the exact next task;
- changes should remain in reviewable, dependency-ordered groups;
- no direct commit to `main`;
- no claim of completion until code, documentation, build output, and manual behavior agree.
