# Network Evidence

## Purpose

M4.4 verifies network behavior from inside the selected managed browser session. It does not treat application-side connectivity checks, proxy configuration, or launch arguments as proof that the browser used the intended route.

Network Evidence is local-first, versioned, bounded, and tied to:

- the existing real-browser Evidence run;
- Profile identity;
- Provider ID and revision;
- exact browser version and managed binary identity;
- operating system and architecture;
- M4.3 consistency input digest;
- selected route kind and privacy-preserving route digest;
- ProbeSet ID and revision.

## Explicit ProbeSet requirement

Veilium does not ship a hidden default public probe. The user must explicitly save a ProbeSet containing one or more replaceable or self-hostable definitions.

Supported definitions are:

- `exit-ip`: HTTPS or loopback HTTP endpoint returning a bounded JSON object with an `ip` string;
- `webrtc-stun`: explicit `stun:host:port` endpoint;
- `delegated-dns`: explicit DNS zone plus HTTPS or loopback HTTP result endpoint.

Every definition includes a revision, timeout, privacy note, and response limit where applicable. Plain HTTP is accepted only for loopback test fixtures.

## Controlled collection flow

1. The Profile must have a ready managed browser session.
2. A current passed or partial M4.2/M4.3 Evidence run must match the Profile and kernel.
3. Veilium starts an ephemeral IPv4 loopback Collector.
4. A constrained CDP Target opens the Collector page.
5. The page runs only probes declared by the selected ProbeSet.
6. The browser submits one bounded, allowlisted JSON result.
7. Veilium reconciles Exit IP, WebRTC/STUN, and delegated DNS observations.
8. Target and Collector are closed.
9. A private write-once Network Evidence report is stored locally.

The Collector enforces loopback access, Host and Origin validation, strict content types, one-time submission, request-size limits, no-store headers, strict JSON decoding, and a ProbeSet-derived Content Security Policy.

## Observation rules

### Exit IP

The browser calls the configured endpoint through its current route. A successful observation contains exactly one normalized IP address.

An unavailable endpoint produces `unavailable`, never an optimistic pass.

### WebRTC/STUN

The browser uses a temporary `RTCPeerConnection`. Veilium stores only normalized summaries:

- candidate type: host, srflx, prflx, or relay;
- protocol: UDP or TCP;
- whether a host candidate used mDNS;
- normalized reflexive or relay public IPs.

Veilium does not store raw ICE candidates, SDP, local media, screenshots, cookies, storage, history, or arbitrary page content.

A STUN public IP that differs from the browser-observed Exit IP is `failed` with `webrtc-exit-ip-mismatch`. Matching public IPs pass. A result without a comparable public address is partial or unavailable.

### Delegated DNS

The browser requests a random single-purpose token below the configured DNS zone and then queries the configured result endpoint. Stored values are limited to:

- `seen:true` or `seen:false`;
- a normalized resolver IP when supplied;
- a bounded uppercase DNS response code.

A query that is not observed before the deadline is partial. Probe unavailability remains explicit.

## Route identity and secrets

Network reports store route kind, scheme, bridge kind, and a SHA-256 digest derived from the selected route references. They do not store the original proxy URL, inline credentials, vault secret, password, token, adapter private configuration, UUID, or share link.

Changing the selected route or credential reference changes the route digest and makes previous Network Evidence stale for current health evaluation.

## Profile health

Network Evidence extends, but does not replace, M4.3 consistency health.

A current explicit ProbeSet with no applicable Network Evidence degrades health. Missing or unavailable probes do not count as a leak, but they cannot count as verified success.

The following conditions block health:

- a failed network observation;
- WebRTC/STUN public IP mismatch;
- a future explicit route-leak observation marked failed.

The following conditions degrade health:

- no ProbeSet configured;
- no applicable run;
- expired or mismatched Profile, route, Provider, binary, consistency digest, or ProbeSet revision;
- partial, unavailable, or skipped observations;
- cleanup or probe limitations.

No report silently changes the Profile, proxy route, language, timezone, geolocation, or Provider trust.

## Storage and lifecycle

Network Evidence reports are stored in a private directory with atomic write-once files, bounded size, maximum-count retention, expiration pruning, explicit deletion, cancellation, timeout, browser-exit handling, and application-shutdown cancellation.

A failed new run does not delete older reports.

## CI evidence boundary

Required Windows and Linux jobs run a real hosted Chromium against a local synthetic Exit-IP fixture. This verifies the exact Collector → CDP Target → browser request → submission → cleanup path without public network dependency or user credentials.

That fixture proves only the tested browser, runner OS, architecture, and synthetic route. It is not evidence that every production Provider, proxy, STUN service, DNS resolver, operating system, or network is compatible.

STUN and delegated DNS contracts, privacy boundaries, reconciliation, failure behavior, and storage are covered by controlled tests. Production STUN and delegated DNS claims require accepted evidence from the exact configured ProbeSet and exact runtime combination.
