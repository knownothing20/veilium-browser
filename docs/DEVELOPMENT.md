# Veilium Development

Last updated: 2026-08-19
Current state: P0 regression closure and the Multi-Provider reviewed-Kernel foundation are complete on the active feature branch; the production catalog still exposes only the exact stock Chromium Provider
Current task: Specify and build a separately licensed, provenance-recorded, exact Windows amd64 Veilium Fingerprint Chromium package before adding any Fingerprint Provider release or capability claim

> **This file is the single current source of truth for Veilium development.**
> Do not create separate current roadmap, status, phase, architecture, product-plan, handoff, or review documents. Git history is the archive.

## 1. Product goal

Veilium is a local-first, multi-profile privacy browser workspace for isolated browser identities, controlled network routing, testing, QA, account separation, and authorized automation.

The product should combine:

- understandable browser-environment management;
- explicit Provider/version capability contracts;
- exact Kernel and Adapter integrity;
- OS-backed secret storage;
- controlled Proxy/Xray/sing-box routing;
- supervised Chromium execution;
- real-browser identity and network Evidence;
- recoverable Profile lifecycle, backup, restore, portability, and bounded batch operations;
- a separate evidence-backed Fingerprint Chromium Provider rather than optimistic UI-only spoofing.

Core principles:

1. identity consistency over maximum randomness;
2. Evidence over claims;
3. local-first and least privilege;
4. unsupported or unverifiable behavior fails closed or remains explicitly limited;
5. reviewed runtimes use exact Provider/version/package identities;
6. secrets stay in the operating-system vault;
7. storage-changing operations remain recoverable;
8. reference projects may inform requirements, not copied implementation.

Veilium is not intended to promise CAPTCHA/risk-control bypass, account-survival rates, fraud, unauthorized access, platform-rule evasion, account farming, or unauthenticated remote control.

## 2. Current baseline on `main`

The consolidated baseline includes:

- Go + Wails + React desktop workspace;
- isolated Profiles and Veilium-managed browser user-data directories;
- Provider Contract v2 with reviewed/custom/legacy/disabled/invalid trust states;
- one exact reviewed Windows amd64 **stock Chromium Snapshot** Provider;
- archive/executable/complete-package integrity verification;
- Kernel Registry and supervised browser runtime;
- dynamic loopback CDP readiness and process-tree cleanup;
- OS Credential Vault with no plaintext fallback;
- HTTP/HTTPS/SOCKS5 authenticated loopback Proxy Bridge;
- supervised Xray and sing-box supported subsets;
- Proxy Diagnostics, browser identity Evidence, managed-window consistency, Network Evidence, and Compatibility;
- lifecycle journal, locks, cancellation, inventory, snapshot/restore, archive, recoverable trash, rollback, and reconciliation;
- portable non-secret Profile definitions, dependency remapping, templates, and identity-transfer modes;
- bounded multi-Profile metadata/lifecycle/export/health/storage operations and redacted operation reports;
- Simplified Chinese task-oriented desktop workspace.

Important truth: the current reviewed Chromium Provider is stock Chromium. **Advanced fingerprint overrides are not claimed for it.**

## 3. Architecture that must remain stable

```text
Wails + React desktop
        |
Desktop service / authenticated loopback API
        |
Profile + policy + lifecycle + portability
        |
Launch planner -------- Network route resolver
      |                         |
Provider contract         Credential vault
      |                         |
Kernel registry          Proxy bridge / adapters
      |                         |
Browser supervisor       Xray / sing-box runtime
      \                         /
       Identity Evidence + Network Evidence
                    |
        Compatibility / Environment Readiness
```

Stable responsibilities:

- `internal/domain` — Profile, fingerprint, proxy, kernel, launch contracts;
- `internal/fingerprint` — Provider capabilities, validation, Provider-specific args;
- `internal/kernel*` — Kernel release pins, import/install, integrity, in-use protection;
- `internal/launch` — reviewable launch plans;
- `internal/supervisor` — browser process ownership, CDP readiness, logs, cleanup;
- `internal/credential` — OS-backed secret storage;
- `internal/proxy*`, `internal/adapter*`, Xray/sing-box providers — controlled network routing;
- `internal/evidence`, `internal/consistency`, `internal/networkevidence`, `internal/compatibility` — runtime verification;
- lifecycle/recovery/portable-profile packages — recoverable data operations;
- `internal/desktop` — backend product authority;
- `frontend` — presentation only; it must not invent trust/readiness state.

Persistent data classes stay separate: Profile metadata, browser user data, vault secrets, Kernel/Adapter records, Evidence, lifecycle journals, snapshots/trash, runtime logs, and staging data.

## 4. P0 — validation debt completed on 2026-08-19

PR #59 was merged at the owner's request before its remaining manual/runtime checks were complete. The checks below were not retroactively marked as passed: they were rerun against the current feature-branch source and closed with new local evidence on 2026-08-19.

Execute P0 in this order. A later gate must not be used to conceal a failure in an earlier one.

### P0-A — fail-closed product hardening

Close defects that can make metadata or UI state look healthier than the runtime:

- a managed Kernel must pass a bounded offline launch probe before registration and after restart before it is treated as launchable;
- native HTTP/HTTPS/SOCKS5 routes must pass a bounded endpoint preflight before browser start and backend health must report an unreachable endpoint as blocked;
- Profile creation must surface required name, Kernel, route, and Adapter failures in the visible editor rather than relying on hidden browser-form validation;
- Windows system-proxy discovery, when exposed, is read-only and credential-free, converts only safely representable static routes to an explicit Profile URL, and rejects PAC-only or ambiguous per-scheme settings;
- concurrent runtime cleanup must be idempotent and return cleanup failures instead of racing private-directory removal.

Delivered:

- managed reviewed packages run a bounded product-like loopback CDP launch probe before registration and after restart;
- native HTTP/HTTPS/SOCKS5 endpoints run a bounded preflight before health can be ready or a browser can start;
- Profile-editor validation is visible and system-proxy application clears stale Credential and Adapter references;
- the Windows system-proxy API is read-only, never returns credentials, accepts only unambiguous static HTTP/HTTPS/SOCKS routes, and rejects PAC-only or unsafe configuration;
- Adapter runtime cleanup is serialized, idempotent, and returns cleanup failures;
- reviewed Chromium runtime ACLs preserve Sandbox execution while preventing content writes that would silently mutate the integrity-recorded package; failed install/probe paths release the temporary ACL before cleanup;
- Evidence records use a deterministic `evidence-mismatch` failure code when evaluated observations disagree with the expected identity, while unavailable optional worker observations remain explicitly partial instead of falsely passing or blocking unrelated checks.

### P0-B — deterministic local acceptance

Automate backend and presentation invariants before relying on Windows input injection:

- test the Profile-editor state transitions, including system-proxy application clearing stale Credential and Adapter references;
- render the primary workspace at 1366×768 and 1920×1080 using an isolated test data root and capture reviewable evidence;
- prove that presentation-language changes do not mutate Profile language, timezone, platform, Seed, or fingerprint settings;
- prove Profile, Kernel, lifecycle, portable-template, and Evidence metadata survive an application/service restart;
- keep test-only control surfaces disabled in normal builds and loopback-only when enabled.

Windows GUI automation may supplement these checks, but an input-injection permission failure is not permission to bypass the desktop security boundary.

Completed evidence:

- frontend state-transition tests cover visible Profile validation and the system-proxy reset invariant;
- the primary workspace rendered at 1366×768 and 1920×1080 with exact viewport dimensions, no horizontal overflow, and no browser console warnings or errors;
- all eight main workspace destinations opened in the local browser preview, and required Profile name validation blocked an empty submission with the visible in-editor alert `请先填写环境名称。`;
- restart tests preserve Profile identity, Kernel binding, lifecycle metadata, portable templates, and Evidence metadata;
- presentation-language tests continue to prove that UI language does not mutate Profile language, timezone, platform, Seed, or fingerprint settings;
- the production Wails window was built and opened with an isolated `VEILIUM_DATA_DIR`; the WebView workspace rendered and remained alive without input-injection bypasses.

### P0-C — exact real-browser and product regression

Run and repair:

1. real Chromium start, readiness, stop, and child-process cleanup after the Chinese workspace changes;
2. end-to-end smoke tests for Kernel import/install, credentials, Proxy Diagnostics, Recovery, Portability, Templates, Batch Management, Storage, and Evidence;
3. persistence after application restart;
4. proof that management-language presentation does not mutate Profile language, timezone, platform, or fingerprint values.

Use the pinned stock Chromium archive only through the existing integrity-checked installer and an isolated Veilium data root. It is an application-managed runtime, not a Windows system installation. Existing Microsoft Edge may be used for controlled custom-Kernel test fixtures, but it must not inherit reviewed Provider status and copying only its executable is not a valid package installation.

Before CPU- or memory-intensive build, race, package, or real-browser stages, measure host load. Queue the stage only when CPU and memory are simultaneously above 95%; remeasure before resuming instead of starting competing heavy work.

New exact-browser evidence on 2026-08-19:

- Provider `official-chromium-snapshot-win64`, revision `1`, Chromium `152.0.7960.0`, snapshot `1664436`;
- archive size `343585547`, SHA-256 `d224019b7cbc115951b0f5dce8cf232c37244881a3eb969c010e457aa369332f`;
- executable SHA-256 `5093988c8fdf969494f921deb32c177dbe5ed88cc101346852d93e760041e5c9`;
- complete package tree: 261 files, 814120936 bytes, SHA-256 `312cb62d6bfab56ecfa52c4e8047dd33c05a1c17c7e44bc2afd9be436854a8dc`;
- the integrity-checked installer imported that exact archive, the bounded CDP probe reached ready state, the managed 1280×800 window started, Evidence evaluated honestly as partial for unsupported stock-browser surfaces, the process tree stopped, and post-run package verification retained the exact tree identity;
- the host screen observed by the real-browser test was 2880×1800. Stock Chromium has no reviewed Screen-Identity override, so a mismatching Profile screen remains a failure rather than being disguised as window sizing.

Final regression evidence on the same source state:

- `gofmt` over all changed Go sources;
- `go vet ./...`;
- `go test -count=1 ./...`;
- `go test -race -count=1 ./...` using the already-installed local MinGW toolchain from a temporary no-space path;
- `go build ./...`;
- the opt-in read-only Windows static-proxy integration test completed without exposing credentials or changing system settings;
- frontend TypeScript typecheck, 6 Vitest files / 20 tests, and a 76-module Vite production build;
- an initial clean Wails build plus the final `wails build -platform windows/amd64 -o veilium-browser-final.exe` rebuild with Wails 2.12.0;
- `python scripts/check_project_governance.py`.

Every recorded resource gate was below the queue condition; CPU and memory were never simultaneously above 95%.

**Exit:** no blocking regression remains. Then continue automatically to P1.

## 5. P1 — Multi-Provider reviewed Kernel foundation completed on 2026-08-19

The release, Kernel, installer, portability, Evidence, desktop, and frontend contracts now support these identities coexisting without changing the stock Provider:

```text
Official Stock Chromium
Veilium Fingerprint Chromium
Custom Chromium
Legacy Chromium
```

Requirements:

- compound identity: Provider ID + revision + version + OS + arch;
- strict manifest decoding and duplicate rejection;
- per-Provider provenance and archive-layout validation;
- existing stock Provider IDs, digests, Profiles, portable definitions, and Evidence remain valid;
- Kernel Registry, installer, portable dependency matching, frontend descriptors, and Evidence bindings support multiple Providers;
- stock Chromium behavior and capability claims do not change;
- no new fingerprint capability is claimed in this stage.

Implementation order:

1. generalize strict manifest validation and release lookup around the complete compound identity without changing the existing stock manifest values;
2. add a second embedded test Provider fixture to prove duplicate rejection, independent provenance/layout policy, installer lookup, Kernel matching, portable dependency matching, frontend descriptors, and Evidence binding;
3. retain the existing stock Provider ID and exact package identities as compatibility tests;
4. expose only backend-derived Provider labels, release availability, and trust state to the frontend;
5. run tamper and cross-Provider mismatch tests before adding any Fingerprint Chromium release.

**Exit:** a second test Provider can coexist without changing stock Provider trust or behavior.

Delivered and verified:

- the compound identity is Provider ID + Provider revision + browser version + platform + architecture;
- strict multi-release manifests reject duplicate compound identities and apply Provider-scoped provenance, download-host, archive-layout, executable-path, size, and platform policy;
- Kernel Registry records, launch-probe tokens, package import/verify, installer requests, portable dependency matching, local recovery, Evidence binding, and frontend install requests carry Provider revision;
- legacy persisted stock records with no revision migrate in memory to the exact current stock contract, while new exact matching fails closed across Provider or revision boundaries;
- frontend Provider descriptors are derived from backend capability contracts rather than hard-coded Provider IDs;
- an embedded test-only second Provider proves coexistence, independent provenance/layout, installer lookup, portable matching, descriptor derivation, duplicate rejection, cross-revision rejection, and Evidence mismatch handling;
- the production release manifest still contains only `official-chromium-snapshot-win64`; no Fingerprint Provider binary or advanced capability has been added or claimed.

The exact next gate is not another UI placeholder. It is a project/license decision plus a reproducible build and review package for a separate Fingerprint Chromium binary, including source baseline, patch set, build recipe, redistribution/license record, archive/executable/tree digests, Provider revision, runtime-control contract, tamper downgrade, and exact-binary real-browser Evidence.

## 6. P1 — Veilium Fingerprint Chromium V1

Add a separate exact Windows amd64 Fingerprint Chromium Provider. Do not replace or silently upgrade the stock Provider.

V1 target capabilities:

- Platform / Browser Brand;
- Language / Accept-Language;
- Timezone;
- Screen / available screen / device scale;
- Hardware Concurrency;
- Device Memory;
- root-seeded Canvas;
- root-seeded Audio;
- root-seeded ClientRects / DOMRect;
- WebGL / GPU identity;
- WebRTC proxy-only policy.

Defer until the core is stable:

- complete Fonts catalog simulation;
- Speech voice catalog;
- WebGPU templates;
- macOS reviewed Fingerprint Provider;
- parallel maintenance of multiple Chromium major versions.

Rules:

- pin an exact Chromium baseline; never resolve moving latest;
- Provider arguments/contracts are versioned;
- archive, executable, package-tree, provenance, and license records are mandatory;
- `FingerprintConfig` fields must map to real Kernel controls; validation alone is not implementation;
- window size must not be treated as Screen Identity;
- custom Kernels cannot be manually promoted to reviewed;
- every public capability requires exact-binary real-browser Evidence;
- failed optional capabilities remain `unsupported`/`unverified` while unrelated verified work continues.

The Go/Wails foundation may be completed without a Fingerprint Chromium binary. The Fingerprint Provider itself is complete only when an exact locally built or reviewed package, provenance record, license record, package-tree identity, runtime control contract, and real-browser Evidence are all present. Until then its release is absent and every advanced capability remains unsupported or unverified; placeholder digests, stock-Chromium aliases, and UI-only claims are forbidden.

## 7. P1 — Evidence V2 and fingerprint matrix

Extend the existing Evidence system; do not create a parallel tester.

Add observations for:

- `navigator.deviceMemory`;
- Accept-Language;
- GPU vendor/renderer and relevant WebGL parameters;
- Provider capability revision;
- identity-derivation version;
- Fonts/Speech/WebGPU only when later supported.

Required matrices:

1. Context Consistency — top-level / iframe / worker;
2. Restart Stability — same Profile remains stable across restart;
3. Seed Separation — different Seeds separate intended surfaces;
4. Template Coherence — Platform/CPU/Memory/GPU/Screen/Language/Timezone remain plausible;
5. Network Coherence — Route/Exit IP/DNS/WebRTC agree with Profile policy;
6. Tamper Downgrade — protected Kernel modification removes reviewed/compatible state;
7. Evidence Freshness — Provider/Kernel/Profile identity/Route changes invalidate stale Evidence.

Third-party fingerprint websites are supplemental observations, never the sole acceptance gate.

## 8. P2 — Identity Templates and Root Seed

Move ordinary users to a template-first model:

- Windows · host hardware;
- Windows · standard office device;
- Windows · mainstream desktop;
- Windows · high-performance desktop;
- Advanced · custom.

A versioned Root Seed derives stable sub-identities for Canvas, Audio, ClientRects, and WebGL/GPU.

Rules:

- same Profile + same derivation version stays stable;
- Clone creates a new Root Seed by default;
- preserve-identity is explicit and advanced;
- Template updates never silently mutate existing Profiles;
- Templates create configuration only; they do not create Provider trust or Evidence.

## 9. P2 — Network Identity Advisor

Turn Proxy Diagnostics / Network Evidence into an explicit observation layer:

```text
Exit IP / country / region / city
        +
Timezone / locale / geolocation candidates
        +
confidence / conflicts / observed-at / expiry
        ↓
User-reviewable recommendation
```

The user must explicitly apply recommendations. Proxy changes invalidate stale observations but never silently rewrite Profile identity.

Do not add Proxy rotation, pools, scheduled switching, unattended identity mutation, or account-farming workflows.

## 10. P2 — Environment Readiness

Move final environment health truth to the backend. A versioned `EnvironmentReadiness` should combine:

- Lifecycle;
- Kernel integrity;
- Provider trust;
- Capability compatibility;
- Credential/Adapter availability;
- Proxy Diagnostics;
- Identity Evidence;
- Network Evidence;
- Evidence freshness.

Expose a compact product state such as `ready`, `warning`, `blocked`, `recovery-required`, or `unverified`, with detailed reasons through progressive disclosure.

Provider labels and trust descriptors come from backend contracts, not hard-coded frontend IDs. Default Provider selection should prefer an installed, verified, platform-compatible trusted Provider. Default timezone should come from the host or explicit user/network choice, not a hard-coded country.

## 11. Later work — only after the fingerprint core is stable

Deferred product areas:

- Cookie view/import/export;
- Extension management;
- Controlled Automation;
- Scheduler / MCP;
- Release signing and updater;
- SBOM and reproducible builds;
- cloud or cross-device features only after a separate product/security decision.

Do not prioritize these merely to match Prism feature count.

## 12. Non-negotiable boundaries

- stock Chromium and Fingerprint Chromium remain separate Providers;
- unsupported behavior stays unsupported/unverified until Evidence exists;
- secrets remain in the OS vault and do not enter logs, Bootstrap, reports, portable exports, or Chromium args;
- local APIs/control surfaces remain loopback-only unless separately approved;
- Chromium Sandbox, package integrity, process ownership, rollback, and recovery are not weakened to make tests pass;
- no silent runtime download/update/replacement;
- no proxy rotation, account farming, telemetry, cloud control, or public unauthenticated remote-control surface;
- persisted-data changes require compatibility, migration, failure, and rollback analysis;
- reference projects may inform behavior and test strategy, but implementation remains clean-room and license-reviewed.

## 13. Default autonomous development

Ordinary technical work continues without asking the owner whether to proceed.

Do **not** stop for:

- naming, file layout, local refactors, internal API shape;
- build/type/test failures and their repairs;
- small Chromium baseline adjustments;
- patch rebases or replacement implementation details;
- UI component organization;
- test fixtures;
- documentation updates;
- a single optional capability being downgraded to unsupported/unverified.

Stop for owner input only when the next action would:

- materially change product direction;
- irreversibly affect user data without an established recovery path;
- broaden secret exposure, public remote control, telemetry, or cloud dependence;
- weaken Sandbox, package integrity, Provider trust, Evidence, or recovery;
- require a project-license, binary-redistribution, branding, trademark, or commercial/open-source decision.

## 14. Branch and PR discipline

- `main` is the only long-lived canonical branch;
- keep at most one active feature branch for the current major line whenever practical;
- use `agent/<topic>`, merge it, then delete it;
- do not create activation, handoff, closing-review, process-only, temporary-autofix, or diagnostic branches/PRs;
- do not create a new project-plan document for each subtask;
- product-code PRs update **this file** with what changed, what actually passed, remaining limitations, and the exact next task;
- split a PR only when review or rollback safety materially requires it.

## 15. Documentation rule

This file is the only mutable project-development truth.

- Do not create competing `STATUS`, `ROADMAP`, `PHASE`, `PLAN`, `ARCHITECTURE`, or `PRODUCT` files.
- Update this file in the same change set whenever product-code status, priorities, architecture boundaries, validation truth, or the next task changes.
- Root `README.md` is only a stable product entry page.
- `AGENTS.md` is only the stable execution/safety contract for development agents.
- Git history, merged PRs, issues, and commits are the historical archive. Do not duplicate history under `docs/archive/`.

## 16. Completion definition

The current fingerprint-platform line is ready for product review when:

- P0 regression validation is complete;
- a separate exact Fingerprint Chromium Provider can be installed, verified, launched, stopped, tamper-downgraded, and recovered;
- every public fingerprint capability is backed by real-browser Evidence;
- same-Profile restart stability and different-Seed separation pass;
- Network/Identity coherence is visible and reviewable;
- existing stock Provider, Vault, Proxy, Supervisor, Lifecycle, Recovery, Portability, and Chinese workspace behavior remain intact.
