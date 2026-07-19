# Identity and Window Consistency

Status: M4.3 implementation contract
Rules revision: `m4.3-v1`

## Purpose

M4.3 turns profile configuration, Provider Contract v2, exact managed-kernel identity, controlled browser observations, and the real browser window into one versioned consistency result.

The result is derived. Users cannot assign `healthy`, `degraded`, `blocked`, or `unknown` directly, and matching configuration values alone do not establish reviewed capability support.

## Profile window fields

A profile may define:

- `windowWidth`;
- `windowHeight`;
- `deviceScaleFactor`.

The explicit values form a `WindowPlan` used by launch arguments, the managed window controller, evidence freshness, and health evaluation.

For existing profiles, zero values preserve compatibility:

- when width and height are both zero, the screen dimensions are used as `legacy-screen-fallback`;
- when the scale factor is zero, it falls back to `1`;
- the profile is not rewritten automatically;
- the health result records the compatibility fallback as degraded.

A partial explicit window, such as a width without a height, is rejected.

## Pre-launch consistency

Profile create, update, launch-plan generation, and runtime launch run the same versioned preflight rules.

Preflight currently checks:

- Provider trust and launchability;
- declared profile platform against the current host when the Provider cannot safely override it;
- screen dimensions and supported bounds;
- explicit or compatibility WindowPlan bounds;
- window dimensions not exceeding the declared screen;
- device-scale-factor bounds;
- required Provider capability state for configured overrides;
- contradictory or impossible combinations.

Failed required checks block the operation with readable reasons. The original profile remains unchanged.

## Managed browser window lifecycle

The desktop Service wraps its existing Runtime Supervisor during initialization.

For every Direct, built-in bridge, Xray, and sing-box launch path:

1. the normal Planner produces a `LaunchPlan` containing a `WindowPlan`;
2. the existing Supervisor starts Chromium and validates dynamic loopback CDP readiness;
3. the wrapper applies the captured WindowPlan through the bounded window controller;
4. the controller reads the actual window bounds after applying them;
5. the observed state is stored only for the active local session;
6. the ready Session is returned to the caller.

If window application fails, the wrapper stops the newly started browser. Existing advanced-route error handling then closes the associated proxy runtime or authentication bridge.

Stopping a profile or shutting down the application deletes the cached observed window state.

## Window-controller security boundary

The window controller is not a general CDP API.

It:

- accepts only a selected loopback CDP port;
- accepts only a loopback Browser WebSocket on that same port;
- requires a Browser WebSocket path under `/devtools/browser/`;
- obtains only a bounded page Target ID from `/json/list`;
- allows only:
  - `Browser.getWindowForTarget`;
  - `Browser.setWindowBounds`;
  - `Browser.getWindowBounds`;
- limits request time, WebSocket message size, skipped event count, target ID length, dimensions, and scale factor;
- does not navigate pages, read page content, access cookies, or expose a remote-control gateway.

## Evidence freshness

New Evidence Runs store:

- `consistencyInputDigest`;
- `consistencyRulesRevision`.

The digest includes the relevant profile identity and fingerprint configuration, effective WindowPlan, Provider ID/revision/trust, managed kernel and exact binary integrity identity, runtime OS/architecture, evidence-harness revision, and consistency-rules revision.

Any relevant change makes previous Evidence stale. Older M4.2 reports remain readable, but reports without M4.3 metadata cannot remain fresh.

## Health states

### `healthy`

Used only when:

- required preflight checks pass;
- the Provider is reviewed for the applicable exact identity;
- applicable real-browser Evidence is fresh;
- required identity and window observations pass;
- no degraded limitation remains.

### `degraded`

The environment remains launchable, but a documented limitation exists, for example:

- custom or legacy Provider trust;
- legacy screen-to-window fallback;
- partial Provider support;
- expired Evidence where safe launch behavior still remains available;
- a non-blocking observation limitation.

### `blocked`

Launch or continued trusted use must not proceed because a required check failed, such as:

- impossible window geometry;
- host/platform contradiction;
- unsupported required capability;
- modified or invalid managed binary;
- fresh Evidence contradicting required identity or window behavior.

### `unknown`

Required applicable Evidence is missing, incomplete, or stale and no stronger degraded rule applies.

## Window and observation tolerances

The rules deliberately use bounded tolerances rather than exact equality for platform decoration and rounding:

- controlled outer-window width and height: up to 2 pixels;
- observed screen dimensions: up to 1 pixel;
- device-pixel ratio: up to `0.05`;
- viewport must not exceed the inner window by more than 2 pixels;
- visual viewport scale must remain between `0.5` and `4.0`.

These tolerances do not allow an impossible viewport, inner window larger than the outer window, invalid scale, or window larger than the declared screen.

## Desktop product surface

The Profile row provides a Consistency action that:

- derives the current result from the profile, Provider, exact kernel identity, and latest applicable Evidence;
- shows effective window, DPR, evidence freshness, blocking reasons, degraded reasons, and each check;
- lets the user explicitly save window width, window height, and DPR;
- lets the user return all three fields to zero to use the legacy screen fallback;
- uses the normal profile update path, so running profiles and invalid values are rejected.

No result silently edits a profile.

## Testing

Unit and integration coverage includes:

- legacy compatibility;
- Provider/host platform contradiction;
- window larger than screen;
- stale and fresh Evidence;
- health derivation;
- bounded window-controller commands and loopback validation;
- Supervisor wrapper success, failure cleanup, and state deletion;
- desktop health service;
- frontend typecheck and production build;
- real Chromium managed-window application and Evidence collection in the existing Windows and Linux Required jobs.

The CI fixture proves only the exact hosted test environment. It does not grant general reviewed Provider status.

## Non-scope

M4.3 does not add:

- external exit-IP evidence;
- WebRTC/STUN leak probes;
- delegated-domain DNS probes;
- automatic profile correction;
- new fingerprint fields without Provider and Evidence contracts;
- public CDP, Launch API, MCP, sync, cookie, extension, migration, or proxy-pool functionality.
