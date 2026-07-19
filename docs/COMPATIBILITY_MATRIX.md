# Compatibility Matrix

## Status

This document defines how Veilium generates and interprets exact-combination compatibility records. It is not a manually maintained list of optimistic support claims.

At the time M4.4 is implemented, no custom or legacy browser Provider is promoted to reviewed trust merely because a local Network Evidence run succeeds.

## Exact combination key

Each compatibility entry is keyed by:

- Provider ID;
- Provider revision;
- browser version;
- operating system;
- architecture;
- managed binary identity digest;
- capability ID.

Network-related entries additionally record:

- ProbeSet ID and revision;
- accepted Network Evidence IDs;
- review time;
- Evidence expiration;
- limitations.

The generated matrix keeps only the newest applicable entry for an exact combination. A later run for the same combination does not create an ambiguous duplicate row.

## Status meanings

| Status | Meaning |
| --- | --- |
| `verified` | Exact reviewed Provider combination has accepted, current evidence for the capability. |
| `partial` | Exact reviewed Provider combination has accepted evidence with explicit limitations or incomplete coverage. |
| `unsupported` | The Provider contract declares that the exact capability is unsupported. |
| `failed` | Applicable evidence explicitly failed for the exact combination. |
| `stale` | Previously accepted evidence expired or no longer matches the current exact inputs. |
| `untested` | No accepted reviewed evidence exists for the exact combination. |

## Trust boundary

Only a Provider with `reviewed` trust can produce `verified` or `partial` compatibility.

Custom and legacy Providers remain `untested` in the reviewed matrix even when a local observation succeeds. Their reports remain useful for local diagnostics and health, but they do not establish a public Provider compatibility promise.

Network Evidence alone never changes Provider trust.

## Network capability rows

M4.4 generates three capability rows from each newest exact Network Evidence combination:

- `network.route` from the browser-observed Exit-IP result;
- `network.webrtc` from the reconciled WebRTC/STUN result;
- `network.dns` from the delegated-DNS result.

A missing observation does not become verified. It remains partial or untested according to Provider trust and accepted evidence policy.

## Generated output

The desktop service can generate the machine-readable `CompatibilityMatrix` contract from private accepted records. A production release process may serialize that contract to a generated JSON artifact after review.

This repository document intentionally does not embed local user Evidence, route digests, browser paths, Profile identifiers, or probe tokens.

## Controlled CI fixture

Windows and Linux Required CI jobs exercise a real browser against a local synthetic Exit-IP endpoint. That test validates the collection mechanism for the exact hosted environment only.

The controlled fixture does not create a production `verified` row because:

- it is a synthetic test Profile;
- it does not represent a production reviewed Provider;
- it does not validate arbitrary proxy providers or networks;
- it does not broaden STUN or delegated DNS coverage beyond their controlled tests.

## Review procedure

Before publishing a reviewed compatibility row, reviewers must confirm:

1. Provider ID and revision match the reviewed contract;
2. browser version and managed binary digest are exact;
3. operating system and architecture match the Evidence;
4. consistency input and route digest are applicable;
5. ProbeSet ID and revision are approved;
6. referenced Network Evidence is current and accepted;
7. all limitations are included;
8. the status does not exceed the weakest applicable observation;
9. no local secrets or Profile identifiers are published.

Expired evidence becomes `stale`. Re-running a test does not automatically publish a reviewed claim without the review step.
