# Phase 4 — Verified Browser Capability and Evidence

Status: Active
Phase: Phase 4
Owner decision required: No
Product implementation allowed: Yes

## User outcome

At the end of Phase 4, a user can select a reviewed browser-kernel provider, create or edit a profile using only capabilities supported for that provider and version, launch the profile, and receive a local evidence report showing whether the declared browser identity and network route were actually observed in the real browser session.

Veilium must distinguish four capability states:

- **Verified** — tested against the exact reviewed provider/version on a supported platform;
- **Partially verified** — relevant evidence exists, but documented limitations remain;
- **Unsupported** — the provider/version does not implement or cannot safely expose the capability;
- **Unverified custom provider** — the binary may launch through generic behavior, but Veilium makes no advanced fingerprint claim.

A launch argument, UI control, mocked test, file name, or upstream README claim alone is not evidence of support.

## Product principles for this phase

1. Evidence is collected from the running browser, not inferred from configuration alone.
2. Identity coherence is more important than adding the largest number of editable fields.
3. Reviewed-provider status requires exact provenance, license, binary identity, capability contracts, and runtime evidence.
4. Custom providers remain usable for generic local launch, but cannot inherit reviewed claims.
5. Unsupported, contradictory, modified, missing, or unverifiable states fail closed.
6. Evidence records contain no cookies, tokens, browsing content, proxy secrets, or private runtime configurations.
7. Provider updates are explicit and reversible; no reviewed binary is silently replaced.

## Supported-platform policy

- **Windows** is the first required real-browser validation platform.
- **Linux** is required when the reviewed provider declares Linux support.
- **macOS** remains unclaimed until a real macOS validation path and exact reviewed binary exist.
- Portable policy code may be tested on additional platforms, but portable compilation does not establish product support.

## Provider policy

Phase 4 uses a two-tier provider model.

### Reviewed providers

A reviewed provider must define:

- stable provider ID and provider-contract schema version;
- upstream project and source URL;
- license identifier and review notes;
- supported operating systems and architectures;
- supported browser version range;
- exact executable identity and integrity metadata;
- per-capability state and required evidence level;
- known limitations;
- predecessor or rollback-compatible provider definitions where applicable.

Reviewed-provider status is granted only to an exact provider/version/platform/binary combination. A matching name or user-supplied source URL cannot manufacture reviewed status.

### Custom local providers

Custom local providers may be imported and launched through generic capabilities when existing safety checks pass. They must be displayed as unverified and cannot claim advanced fingerprint behavior until a reviewed provider definition and real-browser evidence exist.

Reference projects and browser kernels may inform requirements and architecture only. Veilium remains a clean-room implementation and does not copy their source.

## Ordered milestones

The milestone order is dependency-controlled. Later milestone product work must not begin before the earlier milestone's acceptance criteria are met, except for a narrowly approved planning or test fixture needed to unblock the earlier milestone.

### M4.1 — Kernel Provider Contract v2

**Goal:** establish the trust, versioning, compatibility, and capability model before adding further fingerprint controls.

Scope:

- versioned provider definition and capability contracts;
- reviewed versus custom trust classification;
- source, license, platform, architecture, version, executable identity, and provenance fields;
- explicit capability states rather than optimistic booleans;
- compatibility rules for existing `native-chromium` and `patched-chromium` records;
- provider verify, disable, and rollback behavior;
- service and UI behavior that prevents unsupported settings from being saved or launched as verified;
- failure-path, compatibility, frontend, and cross-platform policy tests.

Acceptance criteria:

- one versioned provider-contract schema exists and is documented;
- existing records remain readable through an explicit compatibility layer;
- no legacy or custom record is silently promoted to reviewed status;
- unsupported and unverified combinations fail closed;
- at least one reviewed-provider candidate can be represented without provider-specific UI hard-coding;
- licensing and binary provenance are mandatory for reviewed providers;
- rollback and recovery behavior is tested.

### M4.2 — Real-Browser Evidence Harness

**Goal:** observe real browser behavior and produce structured local evidence.

Scope:

- local evidence page and collector;
- top-level page, same-origin iframe, worker, and relevant browser API observations;
- structured, redacted, versioned evidence reports;
- comparison of declared profile values, provider claims, and observed values;
- explicit evidence retention and deletion policy;
- exact-binary integration tests where an approved provider binary is available.

Initial evidence surfaces:

- user agent and UA Client Hints where available;
- platform and browser brand;
- language and language list;
- timezone;
- screen, available screen, outer/inner window, viewport, and device-pixel ratio;
- hardware concurrency;
- WebRTC policy observations;
- Canvas, WebGL, Audio, and ClientRects only when the provider contract claims support.

Acceptance criteria:

- each report identifies provider, version, platform, profile, evidence schema, test revision, and timestamp;
- a launch argument or mock alone cannot yield `Verified`;
- conflicting observations produce a failed or partial result with readable reasons;
- evidence collection failure does not mutate or delete the browser profile;
- evidence reports contain no secrets or browsing content;
- the harness runs locally and in CI wherever an approved exact binary is available.

### M4.3 — Identity and Window Consistency

**Goal:** make each profile describe one plausible and stable browser environment.

Scope:

- consistency rules for operating system, browser brand/version, language, timezone, CPU, screen, viewport, DPR, and GPU mode;
- real-window sizing and restoration behavior;
- prevention of impossible or contradictory combinations;
- deterministic profile stability across relaunches;
- profile health derived from evidence rather than configuration validation alone;
- clear UI reasons for invalid or partially verified states.

Acceptance criteria:

- declared screen/window/viewport/DPR combinations pass approved real-browser checks;
- relaunches remain stable unless the profile or provider is explicitly changed;
- contradictory combinations are rejected before launch;
- health status is evidence-derived and understandable;
- failure paths preserve the existing profile and provide recovery guidance.

### M4.4 — Live Browser Network Evidence and Compatibility Matrix

**Goal:** verify the selected route from inside the launched browser and publish truthful support status.

Scope:

- browser-observed exit IP through the selected route;
- controlled WebRTC/STUN leak checks;
- delegated-domain DNS route checks through controlled or self-hostable probes;
- comparison of IP region, timezone, language, and optional geolocation policy;
- distinction among direct, HTTP/HTTPS/SOCKS5 bridge, reviewed Xray, and reviewed sing-box paths;
- generated provider/version/OS/capability compatibility matrix;
- documented limitations when external evidence is unavailable.

Acceptance criteria:

- supported route types are distinguishable in evidence reports;
- route mismatch or leak cannot be reported as healthy;
- probe endpoints are configurable or replaceable and receive no profile secrets;
- compatibility output distinguishes verified, partial, unsupported, and untested combinations;
- the repository contains a reviewed compatibility document generated from evidence records.

## Explicit non-scope

The following work is deferred unless strictly required to complete the approved Phase 4 evidence chain:

- cookie import, export, or editing;
- extension package management;
- complete profile backup, restore, or cross-device migration;
- proxy-pool tagging, batch testing, rotation, or additional protocols/transports;
- stable public Launch API or unified CDP gateway;
- MCP server or broad automation-script platform;
- cloud sync;
- automatic background kernel updates;
- claims about CAPTCHA bypass, account survival, or detection-evasion rates;
- broad UI redesign unrelated to provider, capability, evidence, or health states.

These remain candidates for Phase 5 or Phase 6 planning.

## Data-contract and compatibility policy

The detailed planning contract is defined in `docs/PHASE_04_CONTRACTS.md`.

Phase 4 may add versioned records for:

- provider definition and trust status;
- provider capability declaration;
- evidence run and individual observation result;
- compatibility status and documented limitation;
- profile health derived from reviewed evidence.

Compatibility rules:

- existing kernel and profile records remain readable;
- legacy provider IDs are mapped explicitly rather than rewritten optimistically;
- migration failures preserve original records and return a recovery report;
- evidence schema changes are versioned;
- provider updates do not silently replace an active reviewed version;
- evidence collection never mutates browser data.

## Evidence and validation plan

Each milestone uses the strongest applicable evidence:

1. schema and policy unit tests;
2. component and integration tests;
3. exact real-browser binary tests;
4. Windows and Linux CI where the reviewed provider supports them;
5. modified, missing, incompatible, unsupported, and ambiguous failure-path tests;
6. stable relaunch, process cleanup, and recovery tests;
7. privacy review of evidence records and logs.

The complete repository checks remain mandatory:

```bash
python scripts/check_project_governance.py
make check
```

Additional provider and evidence tests become required checks when their implementation lands.

## Rollback and recovery

- provider definitions and evidence schemas are versioned;
- a new provider version cannot silently replace the active reviewed definition;
- failed provider verification disables new launches that rely on the affected reviewed claim while preserving profile data;
- a previously verified provider version may remain selectable when its exact binary is intact and policy permits it;
- migration failures leave existing records readable and produce a recovery report;
- evidence collection failure does not change the profile, provider binary, credentials, cookies, or user-data directory.

## Phase exit criteria

Phase 4 may enter `Closing` only when:

- Provider Contract v2 and compatibility rules are implemented and frozen;
- at least one reviewed provider/version/platform path has exact real-browser evidence;
- custom providers are visibly unverified and cannot inherit reviewed claims;
- all user-visible fingerprint capability states come from the provider contract and evidence model;
- window, screen, viewport, and DPR consistency passes the approved real-browser matrix;
- browser-observed route, WebRTC, and DNS evidence exists for supported route types;
- conflicting, unsupported, modified, missing, and ambiguous states fail closed;
- a compatibility matrix documents verified, partial, unsupported, and untested combinations;
- governance, Go, frontend, desktop, provider, and real-browser tests pass;
- unresolved limitations and Phase 5 candidates are recorded in the closure PR.

Phase 4 becomes `Done` only through a dedicated closure pull request that verifies every exit criterion, runs the complete validation matrix, records unresolved risks, updates `docs/ROADMAP.md` and `docs/STATUS.md`, and identifies the first Phase 5 planning task.

## Activation record

- [x] Product owner approved the user outcome.
- [x] Milestones are ordered by dependency.
- [x] Every milestone has acceptance criteria.
- [x] Non-goals prevent unrelated expansion.
- [x] Security and licensing impact is documented.
- [x] Required tests and supported platforms are explicit.
- [x] Data compatibility, rollback, and recovery rules are explicit.
- [x] `docs/ROADMAP.md` and `docs/STATUS.md` are updated in the activation pull request.
- [x] Phase status is changed from `Planning` to `Active`.

## Current authorized work

The first authorized implementation task is Issue #17, **M4.1 Kernel Provider Contract v2**. No M4.2–M4.4 product implementation or deferred non-scope work should be added to that issue or its pull request.