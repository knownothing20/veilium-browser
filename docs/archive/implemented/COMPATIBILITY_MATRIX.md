# Compatibility Matrix

## Status

Veilium generates compatibility records from accepted Evidence for exact browser combinations. The repository now contains one reviewed browser Provider: the pinned official Chromium Snapshot documented in [`OFFICIAL_CHROMIUM_PROVIDER.md`](OFFICIAL_CHROMIUM_PROVIDER.md).

This document is not a manually maintained list of optimistic support claims. A passing local observation cannot broaden Provider trust, platform support, or capability coverage.

## Exact combination key

Each compatibility entry is keyed by:

- Provider ID;
- Provider revision;
- browser version;
- operating system;
- architecture;
- binary-identity digest;
- capability ID.

Network-related entries additionally record:

- ProbeSet ID and revision;
- accepted Network Evidence IDs;
- review time;
- Evidence expiration;
- limitations.

The generated matrix keeps only the newest applicable entry for an exact combination. A later run for the same combination does not create an ambiguous duplicate row.

## Reviewed Chromium boundary

`official-chromium-snapshot-win64` may appear as reviewed only when all of these fields match the embedded release:

| Field | Required value |
| --- | --- |
| Provider revision | `1` |
| Chromium version | `152.0.7960.0` |
| Operating system | `windows` |
| Architecture | `amd64` |
| Snapshot revision | `1664436` in the underlying managed package identity |
| Archive SHA-256 | `d224019b7cbc115951b0f5dce8cf232c37244881a3eb969c010e457aa369332f` in the underlying managed package identity |
| Package Tree SHA-256 | `312cb62d6bfab56ecfa52c4e8047dd33c05a1c17c7e44bc2afd9be436854a8dc` in the underlying managed package identity |
| Executable SHA-256 | `5093988c8fdf969494f921deb32c177dbe5ed88cc101346852d93e760041e5c9` in the underlying managed package identity |

The compatibility contract validates the exact Provider revision, browser version, operating system, architecture, and reviewed trust against the embedded release. The binary-identity digest remains part of every row, so records for different managed identities cannot collapse into one claim.

Linux, macOS, arm64, nearby Chromium versions, another Snapshot, a custom executable, or a modified package cannot inherit this reviewed identity.

## Status meanings

| Status | Meaning |
| --- | --- |
| `verified` | The exact reviewed Provider combination has accepted, current Evidence for the capability. |
| `partial` | The exact reviewed Provider combination has accepted Evidence with explicit limitations or incomplete coverage. |
| `unsupported` | The Provider contract declares that the exact capability is unsupported. |
| `failed` | Applicable Evidence explicitly failed for the exact combination. |
| `stale` | Previously accepted Evidence expired or no longer matches the current exact inputs. |
| `untested` | No accepted reviewed Evidence exists for the exact combination. |

## Trust boundary

Only a Provider with `reviewed` trust can produce `verified` or `partial` compatibility.

Custom and legacy Providers remain `untested` in the reviewed matrix even when a local observation succeeds. Their reports remain useful for local diagnostics and health, but they do not establish a public Provider compatibility promise.

Network Evidence alone never changes Provider trust. Reviewed trust also does not convert stock Chromium's unsupported advanced fingerprint controls into supported capabilities.

## Network capability rows

M4.4 generates three capability rows from each newest exact Network Evidence combination:

- `network.route` from the browser-observed Exit-IP result;
- `network.webrtc` from the reconciled WebRTC/STUN result;
- `network.dns` from the delegated-DNS result.

A missing observation does not become verified. It remains partial or untested according to Provider trust, accepted Evidence, and recorded limitations. A failure is never hidden by a passing result from another probe.

## Protected CI Evidence

Windows Required CI installs the exact pinned Snapshot through the product installer and passes that same managed `chrome.exe` to:

1. real-browser identity Evidence;
2. managed-window and consistency Evidence;
3. controlled Network Evidence;
4. complete-package dependency tamper verification.

The controlled Network Evidence fixture uses a synthetic Profile, a controlled direct route, and a local synthetic Exit-IP endpoint. It proves that the exact reviewed binary can complete the M4.4 collection path. It does not establish compatibility for arbitrary proxy providers, routes, networks, STUN services, delegated DNS zones, or user Profiles.

Linux Required CI continues to validate the generic Evidence harness and official adapter smoke tests with the hosted/container browser. It does not create Linux reviewed Chromium compatibility.

## Generated output

The desktop service generates the machine-readable `CompatibilityMatrix` contract from private accepted records. A production review process may serialize that contract after confirming the exact Provider, binary, consistency, route, ProbeSet, Evidence, expiration, and limitations.

Repository documentation and CI artifacts intentionally avoid local user Evidence, route secrets, browser paths from user machines, Profile identifiers, credentials, and probe tokens.

## Review procedure

Before publishing a reviewed compatibility row, reviewers must confirm:

1. Provider ID and revision match the reviewed contract;
2. browser version, operating system, and architecture match the embedded release;
3. the binary-identity digest belongs to the accepted base browser Evidence;
4. the managed package identity matches the exact archive, executable, and Package Tree pins;
5. consistency input and route digest are applicable;
6. ProbeSet ID and revision are approved;
7. referenced Network Evidence is current and accepted;
8. all limitations are included;
9. the status does not exceed the weakest applicable observation;
10. no local secrets or Profile identifiers are published.

Expired Evidence becomes `stale`. Re-running a test does not automatically publish a reviewed claim without the review step.
