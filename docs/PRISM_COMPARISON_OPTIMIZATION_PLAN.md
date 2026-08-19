# Veilium Fingerprint Platform Optimization Plan

Status: Active development plan
Last updated: 2026-08-19
Execution mode: Autonomous by default; owner gate only for major product/licensing decisions

## 1. Objective

Prism Browser demonstrates a complete fingerprint-browser loop: custom Chromium, profile-to-kernel arguments, stable seeded surfaces, hardware templates, proxy-aware identity suggestions, and fingerprint matrices.

Veilium already has the stronger surrounding platform: Provider contracts, exact package integrity, OS Credential Vault, proxy bridges/adapters, Browser Supervisor, real-browser/Network Evidence, Compatibility, Lifecycle, Recovery, Portability, and bounded multi-Profile operations.

The goal is **not** to copy Prism or replace Go/Wails. The goal is to add a separate Veilium Fingerprint Chromium Provider and reuse Veilium's existing security/evidence/runtime architecture.

## 2. Confirmed code gaps

### Single reviewed Provider assumption

`internal/kernelrelease` and `internal/fingerprint` currently assume one reviewed official Chromium release. The catalog must support multiple exact Providers before a fingerprint Provider can coexist safely with the stock Provider.

### Incomplete Profile-to-kernel mapping

`FingerprintConfig` contains more identity fields than the current stock Chromium can apply. In particular, Device Memory is validated but not mapped to a real fingerprint-kernel control, and window size must not be treated as Screen Identity.

### Arbitrary identity composition

Users can still combine platform, CPU, memory, GPU, screen, language, timezone, and surface modes independently. Veilium needs a small template-first identity model with an advanced escape hatch.

### Proxy and identity are not yet connected

Proxy Diagnostics and Network Evidence are strong, but there is no explicit observation/recommendation layer that turns an exit location into reviewable timezone/language/geolocation suggestions.

### Evidence lacks longitudinal fingerprint tests

Current Evidence already covers top-level/iframe/worker and Canvas/WebGL/Audio/ClientRects digests. It still needs Device Memory, richer GPU data, restart stability, seed separation, freshness, and Provider tamper downgrade as one matrix.

## 3. Stage 0 — Post-merge regression validation

Priority: P0

Because the Phase 5 product branch was consolidated before its remaining manual smoke tests were completed, first run and repair:

- real Chromium start/readiness/stop/process cleanup;
- 1366×768 and 1920×1080 Chinese primary flows;
- kernel import/install, credentials, proxy diagnostics, recovery, portability, templates, batch, storage, and Evidence;
- application restart persistence;
- proof that management-language changes do not mutate Profile identity values.

Exit: no blocking regression remains. Then proceed automatically to Stage 1.

## 4. Stage 1 — Multi-Provider reviewed kernel foundation

Priority: P1

Refactor the reviewed release model so these can coexist:

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
- existing stock Provider ID/digests/records remain valid;
- Kernel Registry, installer, portable dependency matching, frontend descriptors, and Evidence bindings understand multiple Providers;
- no new fingerprint capability is claimed in this stage.

Exit: a second test Provider can exist without changing stock Chromium behavior or trust.

## 5. Stage 2 — Veilium Fingerprint Chromium V1

Priority: P1

Add a separate exact Windows amd64 fingerprint Provider. Start with a deliberately small capability set:

- Platform / browser brand;
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

Defer Fonts catalog simulation, Speech voices, WebGPU templates, macOS reviewed Provider, and parallel maintenance of multiple Chromium major versions until the core is stable.

Rules:

- pin an exact Chromium baseline; do not resolve moving latest;
- define versioned provider arguments/contracts;
- archive/executable/package-tree/provenance checks are mandatory;
- stock and fingerprint Providers remain visibly separate;
- custom kernels cannot be manually promoted to reviewed;
- every claimed capability needs exact-binary real-browser evidence;
- a capability that cannot be verified stays unsupported/unverified while other validated capabilities continue.

## 6. Stage 3 — Evidence V2 and fingerprint matrix

Priority: P1

Extend the existing Evidence system instead of creating a parallel tester.

Add observations for:

- `navigator.deviceMemory`;
- Accept-Language;
- GPU vendor/renderer and relevant WebGL parameters;
- Provider capability revision and identity-derivation version;
- Fonts/Speech/WebGPU only when the Provider later supports them.

Required matrices:

1. **Context consistency** — top-level, iframe, worker;
2. **Restart stability** — same Profile stays stable across restart;
3. **Seed separation** — different seeds separate the surfaces that are supposed to differ;
4. **Template coherence** — platform/CPU/memory/GPU/screen/language/timezone combinations remain plausible;
5. **Network coherence** — route, exit IP, DNS, WebRTC, and Profile policy agree;
6. **Tamper downgrade** — protected Kernel modifications remove reviewed/compatible state;
7. **Evidence freshness** — Provider, Kernel, Profile identity, or Route changes make old Evidence stale.

Third-party fingerprint websites are supplemental observations, never the sole acceptance gate.

## 7. Stage 4 — Identity Templates and Root Seed

Priority: P2

Use a template-first product model:

- Windows · host hardware;
- Windows · standard office device;
- Windows · mainstream desktop;
- Windows · high-performance desktop;
- Advanced · custom.

A versioned Root Seed derives stable sub-identities for Canvas, Audio, ClientRects, and WebGL/GPU. Later versions may include Fonts, Speech, and WebGPU.

Rules:

- same Profile + same derivation version remains stable;
- Clone creates a new Root Seed by default;
- preserve-identity remains explicit and advanced;
- template updates never silently mutate existing Profiles;
- templates create configuration only and cannot create Provider trust or Evidence.

## 8. Stage 5 — Network Identity Advisor

Priority: P2

Create a bounded observation layer from Proxy Diagnostics / Network Evidence:

```text
Exit IP / country / region / city
        +
Timezone / locale / geolocation candidates
        +
confidence / conflicts / observed-at / expiry
        ↓
User-reviewable recommendation
```

The user must explicitly apply recommendations. Proxy changes invalidate stale observations but do not silently rewrite Profile identity.

Do not add proxy rotation, pools, scheduled switching, unattended identity mutation, or account-farming workflows.

## 9. Stage 6 — Environment Readiness

Priority: P2

Move product health truth to the backend. `EnvironmentReadiness` should combine:

- Lifecycle;
- Kernel integrity;
- Provider trust;
- Capability compatibility;
- Credential/Adapter availability;
- Proxy Diagnostics;
- Identity Evidence;
- Network Evidence;
- Evidence freshness.

Expose a small product state such as `ready`, `warning`, `blocked`, `recovery-required`, or `unverified`, with detailed reasons available through progressive disclosure.

Provider labels and trust descriptors come from backend contracts, not hard-coded frontend Provider IDs. Default Provider selection should prefer an installed, verified, platform-compatible trusted Provider. Default timezone should come from the host or explicit user/network choice, not a hard-coded country.

## 10. Code map

| Area | Main work |
| --- | --- |
| `internal/kernelrelease` | Multi-Provider exact release catalog |
| `internal/kernelinstaller` | Provider-aware install/verify/rollback |
| `internal/kernel` | Multi-Provider registry and in-use protection |
| `internal/fingerprint` | Provider capabilities, validation, complete argument mapping |
| `internal/domain` | Root Seed, derivation version, template reference |
| `internal/launch` | Provider-aware launch plan |
| `internal/evidence` | Evidence V2, restart/seed/context matrices |
| `internal/networkevidence` | Route-bound browser network evidence |
| `internal/proxydiagnostics` | Network identity observations |
| `internal/desktop` | EnvironmentReadiness authority |
| `frontend` | Template-first editor, recommendations, readiness presentation |

## 11. Owner gates

Normal technical work proceeds autonomously. Stop for owner input only when:

- the feasible Chromium baseline would materially change product positioning;
- progress requires weakening Sandbox, package integrity, secret isolation, recovery, or Evidence;
- a change is irreversible for user data without an established recovery path;
- the design would introduce public remote control, telemetry, or cloud dependence;
- project licensing, fingerprint-kernel binary redistribution, branding, or commercial/open-source boundaries must be decided.

## 12. Non-goals

- no claim of CAPTCHA/risk-control bypass or account-survival rate;
- no per-launch random identity;
- no account farming, proxy rotation, or unattended bulk browser start;
- no automatic promotion of custom Chromium to reviewed;
- no V1 requirement to support all desktop OSes or all Chromium versions;
- no weakening of fail-closed behavior to improve third-party detection-site scores;
- no direct copying of Prism application code or unreviewed patches.

## 13. Completion definition

The fingerprint platform is ready for product review when the exact fingerprint Provider can be installed, verified, launched, stopped, tamper-downgraded, and recovered; every public capability is backed by real-browser evidence; same-Profile restart stability and different-seed separation pass; Network/Identity coherence is visible; and the existing stock Provider, Vault, Proxy, Supervisor, Lifecycle, Recovery, and Portability behavior remain intact.
