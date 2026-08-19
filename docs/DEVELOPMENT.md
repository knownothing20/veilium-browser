# Veilium Development

Last updated: 2026-08-19
Current state: Consolidated Phase 1–5 product baseline is on `main`; post-merge real-browser/product regression validation remains
Current task: Complete the P0 validation debt below, repair any regressions, then continue automatically into the Multi-Provider foundation

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

## 4. P0 — validation debt that must be completed first

PR #59 was merged at the owner's request before its remaining manual/runtime checks were complete. These checks must not be retroactively marked as passed.

Run and repair:

1. real Chromium start, readiness, stop, and child-process cleanup after the Chinese workspace changes;
2. manual primary-flow review at 1366×768 and 1920×1080;
3. end-to-end smoke tests for Kernel import/install, credentials, Proxy Diagnostics, Recovery, Portability, Templates, Batch Management, Storage, and Evidence;
4. persistence after application restart;
5. proof that management-language presentation does not mutate Profile language, timezone, platform, or fingerprint values.

Previously recorded successful checks from the former PR #59 include frontend typecheck/tests/build, Go fmt/vet/unit/race tests, Go build, Windows Wails build, Wails development startup, and governance validation.

**Exit:** no blocking regression remains. Then continue automatically to P1.

## 5. P1 — Multi-Provider reviewed Kernel foundation

The current release/fingerprint code still assumes a single reviewed official Chromium path. Refactor it so these can coexist:

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

**Exit:** a second test Provider can coexist without changing stock Provider trust or behavior.

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
