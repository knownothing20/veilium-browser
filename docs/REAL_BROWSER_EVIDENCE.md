# Controlled real-browser evidence

Veilium M4.2 collects structured observations from a browser session that Veilium already manages. The evidence path is local, explicit, bounded, and independent from the browser profile.

## What the user does

1. Import a Chromium executable into the managed kernel registry.
2. Assign that managed kernel to a profile.
3. Start the profile and wait until its runtime state is `ready`.
4. Open the evidence action from the profile row.
5. Choose **Run evidence**.
6. Review the resulting local report or delete it independently.

Historical reports remain viewable while the browser is stopped. A new run requires the same ready managed session and exact managed kernel identity.

## Controlled execution chain

1. The desktop service re-verifies the managed kernel and resolves the active Provider Contract v2 definition.
2. The evidence manager binds the run to the current profile, provider revision, binary identity, operating system, architecture, process, and dynamically discovered loopback CDP port.
3. A temporary HTTP collector binds to `127.0.0.1` on an operating-system-assigned port.
4. The collector creates a cryptographically random one-time path token and a per-page CSP nonce.
5. The bounded CDP target client asks only the selected loopback CDP endpoint to open that controlled loopback URL.
6. The page collects allowlisted observations from:
   - the top-level controlled page;
   - one same-origin iframe;
   - one same-origin Web Worker.
7. The page makes one JSON submission to the tokenized loopback endpoint.
8. The manager closes the temporary target and collector, evaluates the observations, and writes one terminal report.

Veilium does not expose a generic CDP proxy or a remote evidence API.

## Allowlisted observations

The initial evidence schema can contain:

- user agent;
- UA Client Hints platform and brand/version entries when available;
- `navigator.platform`;
- primary language and language list;
- IANA timezone reported by `Intl`;
- hardware concurrency;
- screen width, height, available size, color depth, and pixel depth;
- outer window, inner window, visual viewport, scale, and device-pixel ratio;
- local WebRTC availability, ICE candidate type/protocol indicators, mDNS use, and gathering state;
- SHA-256 digests of fixed Canvas, WebGL, Audio, and ClientRects samples only when the provider contract makes those surfaces relevant.

Worker contexts intentionally report unavailable DOM-only fields rather than fabricating values.

## What is never collected

Evidence reports do not contain:

- cookies;
- LocalStorage or IndexedDB values;
- authentication tokens or credentials;
- browsing history or arbitrary URLs;
- arbitrary page contents or screenshots;
- downloads;
- proxy usernames, passwords, UUIDs, share links, or generated private adapter configuration;
- microphone, camera, geolocation, clipboard, USB, serial, Bluetooth, or payment data;
- external exit-IP, STUN, or delegated DNS results in M4.2.

The controlled page uses a restrictive Content Security Policy, no-store response headers, loopback Host/Origin validation, bounded request bodies, strict JSON decoding, and one-time submission semantics.

## Status meanings

### Run status

- `passed` — all required observations passed and the exact provider identity is reviewed;
- `partial` — useful observations were collected, but review status or documented limitations prevent a full pass;
- `failed` — observations contradicted required expectations or the controlled chain failed unsafely;
- `cancelled` — the user or application shutdown cancelled the run;
- `incomplete` — timeout, browser exit, or unavailable required contexts prevented completion.

A custom or legacy provider cannot become reviewed merely because observed values match. Its report remains partial, failed, or incomplete according to the observations and limitations.

### Observation status

Each observation records its context and can be `passed`, `partial`, `failed`, `unavailable`, or `skipped`. Provider capability state and browser observation state remain separate concepts.

## Storage and retention

- reports use a versioned JSON schema;
- the evidence directory uses private permissions;
- each report is written atomically and is write-once;
- individual reports are limited to 1 MiB;
- identifiers, strings, list sizes, numerical ranges, report counts, and submitted bodies are bounded;
- the default retention period is 30 days;
- the default maximum is 100 reports;
- old or expired reports are pruned according to store policy;
- reports can be deleted without modifying the profile or browser user-data directory.

Evidence storage rejects symlinked roots, symlinked report files, non-regular files, duplicate run IDs, oversized files, malformed JSON, unsupported schema versions, and expired records.

## Failure and cleanup behavior

Only one evidence run may be active for a profile. The manager handles:

- collector start failure;
- target creation failure;
- malformed or oversized submissions;
- duplicate submissions;
- collection timeout;
- explicit cancellation;
- browser process or session replacement;
- target close failure;
- collector close failure;
- comparison failure;
- report persistence failure;
- application shutdown.

Failures do not edit or delete the profile. Target/collector cleanup limitations are recorded in the terminal report when possible.

## Test boundary

Unit and integration tests cover contracts, private storage, retention, symlink rejection, Collector HTTP security, Target client loopback restrictions, comparison semantics, cancellation, timeout, browser exit, cleanup, and desktop bindings.

The required Windows job launches the hosted Chrome binary. The required Linux job compiles the Evidence integration test as a static binary and runs it with repository Chromium inside a clean official Debian container, avoiding host enterprise-policy differences. Both dynamically discover loopback CDP, open only the controlled evidence page, and require valid top-level, iframe, and worker submissions. This proves the M4.2 collection chain for those exact CI fixtures; it does not by itself grant production reviewed-provider status.

## Deferred work

M4.2 does not implement:

- final screen/window/viewport/DPR correction or consistency policy — M4.3;
- external exit-IP, WebRTC/STUN leak, or delegated-domain DNS evidence — M4.4;
- Cookie, extension, full migration, public Launch API, MCP, synchronization, or release work — later phases.
