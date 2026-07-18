# Phase 4 Capability and Evidence Contracts

Status: Approved planning contract
Phase: Phase 4
Contract revision: 1

## Purpose

This document freezes the logical contracts that Phase 4 implementation must satisfy. It defines required meanings, compatibility behavior, privacy boundaries, and ownership. It does not prescribe exact Go type names, file layouts, or database formats where implementation flexibility remains safe.

The active phase scope and milestone order remain governed by `docs/PHASE_04.md`.

## Contract principles

1. Provider identity, capability claims, and runtime evidence are separate records.
2. A capability declaration is not evidence that a real browser applies it.
3. Reviewed status belongs to an exact provider/version/platform/binary identity.
4. Custom or legacy providers cannot inherit reviewed status through names, URLs, or user-entered metadata.
5. Compatibility migrations preserve original records until a replacement record is durably validated.
6. Evidence data is local, redacted, versioned, and independently deletable.
7. Product health status is derived; it is never a user-editable claim.

## Required status vocabularies

### Provider trust status

Allowed meanings:

- `reviewed` — exact provider definition, license, provenance, binary identity, platform support, and review metadata are present;
- `custom` — locally imported provider without Veilium-reviewed capability claims;
- `legacy` — record created before Provider Contract v2 and interpreted through compatibility rules;
- `disabled` — provider is retained for recovery or audit but cannot start new sessions;
- `invalid` — required identity, integrity, or contract data is missing or contradictory.

Implementation may use different serialized names only through an explicit versioned mapping.

### Capability status

Allowed meanings:

- `verified` — required evidence passed for the exact reviewed combination;
- `partial` — some evidence passed but one or more documented limits remain;
- `unsupported` — provider contract explicitly does not support the capability;
- `unverified` — not enough reviewed real-browser evidence exists;
- `failed` — evidence was attempted and contradicted the claim or exposed an unsafe mismatch.

`verified` and `partial` are evidence-derived states. User input cannot assign them.

### Evidence run status

Allowed meanings:

- `pending`;
- `running`;
- `passed`;
- `partial`;
- `failed`;
- `cancelled`;
- `incomplete`.

An interrupted or unavailable probe must not become `passed`.

### Profile health status

Allowed meanings:

- `healthy` — all required checks for the selected reviewed provider and route passed;
- `degraded` — required safe behavior remains usable, but documented partial or stale evidence exists;
- `blocked` — launch must not proceed because provider, capability, integrity, or consistency requirements fail;
- `unknown` — required evidence has not been collected or is no longer applicable.

Profile health is derived from provider, configuration, integrity, evidence, and freshness policy.

## Logical record: ProviderDefinition

A versioned provider definition must contain at least:

- contract schema version;
- stable provider ID;
- display name and optional description;
- trust status;
- upstream project identity and canonical source;
- license identifier and optional review note;
- supported operating systems;
- supported architectures;
- supported browser version range or exact versions;
- expected executable name or identity rule;
- binary provenance requirements;
- capability declarations;
- known limitations;
- creation and review timestamps;
- predecessor, replacement, or rollback compatibility references where applicable.

For a reviewed provider, upstream source, license, platform, architecture, version support, and binary provenance are mandatory.

A provider definition must not contain credentials, cookies, browser history, proxy secrets, or private runtime configurations.

## Logical record: ProviderBinaryIdentity

The exact binary identity for a reviewed provider must contain enough information to distinguish the reviewed artifact from a similarly named binary:

- provider ID;
- provider-definition revision;
- browser version;
- operating system and architecture;
- executable size;
- executable cryptographic digest;
- source or installer provenance;
- verification timestamp;
- integrity state;
- optional archive identity when installation uses an archive;
- optional native version output digest or normalized version evidence.

User-entered source URLs, display names, or license strings cannot create a reviewed identity.

When the managed executable no longer matches its identity, new launches using reviewed claims must fail closed. The original record remains available for diagnosis and recovery.

## Logical record: CapabilityDeclaration

Each provider capability declaration must contain:

- capability ID;
- provider ID and provider-definition revision;
- supported browser version/platform constraints;
- declared status before evidence, normally `unsupported` or `unverified`;
- required evidence surfaces;
- required evidence freshness policy;
- configuration constraints and incompatible combinations;
- known limitations;
- fallback behavior, which must never silently weaken a reviewed claim.

Initial capability IDs should map to existing product concepts rather than creating duplicate synonyms. Candidate areas include:

- platform;
- browser brand;
- language and language list;
- timezone;
- screen/window/viewport/DPR;
- hardware concurrency;
- WebRTC policy;
- Canvas;
- WebGL and GPU metadata;
- Audio;
- ClientRects.

The M4.1 implementation issue decides exact identifiers and mapping to existing fields, but must document all mappings and compatibility behavior.

## Logical record: EvidenceRun

An evidence run must contain at least:

- evidence schema version;
- evidence run ID;
- profile ID;
- provider ID and provider-definition revision;
- exact provider binary identity reference;
- browser version, operating system, and architecture;
- route kind and redacted route identity;
- evidence harness revision;
- start and completion timestamps;
- run status;
- ordered observation results;
- derived compatibility and profile-health summary;
- limitations and unavailable probes;
- retention metadata.

The route identity must be sufficient to distinguish direct, built-in bridge, Xray, and sing-box execution without exposing usernames, passwords, UUIDs, tokens, full share links, or private generated configuration.

## Logical record: EvidenceObservation

An evidence observation must contain:

- observation ID or capability ID;
- observation context, such as top-level page, iframe, worker, browser process, or controlled network probe;
- expected value or policy reference;
- observed redacted value or digest where raw values are sensitive or unnecessarily identifying;
- observation status;
- readable reason;
- collection timestamp;
- collector revision;
- optional limitation code.

Evidence observations must not store:

- page contents unrelated to the controlled evidence page;
- arbitrary browsing URLs or history;
- cookies or storage values;
- authentication tokens;
- proxy credentials;
- decrypted private adapter configuration;
- downloaded user files.

## Logical record: CompatibilityEntry

A compatibility entry is generated from reviewed provider definitions and accepted evidence. It must contain:

- provider ID and revision;
- exact or bounded browser version;
- operating system and architecture;
- capability ID;
- compatibility status;
- evidence run references;
- last reviewed timestamp;
- known limitations;
- evidence freshness or expiration state.

A generated compatibility document may summarize these records, but the source records remain versioned and reviewable.

## Logical record: ProfileHealthSummary

A profile health summary must contain:

- profile ID;
- provider and binary identity references;
- configuration consistency result;
- binary integrity result;
- latest applicable evidence run reference;
- evidence freshness state;
- network route result where required;
- derived health status;
- blocking reasons;
- degraded reasons;
- generated timestamp.

It is a cacheable derived result, not authoritative user data. It must be recomputable from source records.

## Existing-data compatibility

### Existing kernel records

Existing kernel records remain readable. Compatibility handling must:

- preserve original provider string, version, executable path, digest, size, and timestamps;
- map known legacy IDs through an explicit compatibility table;
- classify legacy records as `legacy` or `custom` unless an exact reviewed binary match is independently established;
- never promote a legacy record to `reviewed` based only on `native-chromium`, `patched-chromium`, a file name, or a declared version;
- keep modified and missing states fail closed;
- provide an actionable compatibility or recovery message.

### Existing profiles

Existing profiles remain readable. Compatibility handling must:

- preserve fingerprint and route configuration;
- resolve the referenced kernel through the compatibility layer;
- mark advanced claims `unverified` when reviewed evidence is unavailable;
- reject new launches only when a required safety or integrity condition fails;
- avoid deleting or silently rewriting the profile during compatibility evaluation.

### Existing provider capability booleans

Existing boolean capabilities are legacy declarations. They may guide migration but cannot directly become `verified`.

The implementation must define a deterministic mapping such as:

- false or unavailable capability → `unsupported`;
- true legacy capability without reviewed evidence → `unverified`;
- reviewed exact-binary evidence → evidence-derived `verified`, `partial`, or `failed`.

The exact mapping must be covered by tests and module documentation.

## Migration rules

1. Migration is versioned and idempotent.
2. Original data remains recoverable until the new record set is durably written and validated.
3. A failed migration returns a recovery report and does not partially promote trust status.
4. Unknown fields are not silently discarded when safe preservation is possible.
5. Downgrade and rollback behavior must be documented before persisted schema changes merge.
6. A migration may classify data more conservatively, but never more optimistically without evidence.
7. Evidence records have an independent schema version from provider and profile records.

## Provider update and rollback rules

- provider updates are explicit user or administrator actions;
- an update creates or selects a new provider definition/binary identity rather than mutating the previous reviewed identity in place;
- profiles do not silently switch reviewed provider versions;
- failed verification prevents selection for new reviewed launches;
- previous reviewed versions remain usable only while exact integrity, compatibility, and policy requirements still pass;
- rollback never rewrites evidence from a different provider revision as applicable;
- removed or disabled providers remain referenced by historical evidence without exposing deleted secrets.

## Evidence freshness

Capability evidence can become stale when any applicable input changes, including:

- provider-definition revision;
- exact provider binary identity;
- browser version;
- operating system or architecture;
- evidence harness revision;
- profile configuration relevant to the capability;
- network route relevant to a network observation;
- freshness duration defined by policy.

Stale evidence becomes `unknown` or `degraded` according to policy; it cannot remain `verified` without an applicable reviewed evidence run.

## Security and privacy boundaries

- all evidence services bind to loopback unless a separately reviewed controlled probe is explicitly required;
- controlled external probes receive only the minimum network request needed for the test;
- evidence files use private permissions and bounded size;
- evidence collection supports cancellation and cleanup;
- secret-bearing command lines and generated adapter configurations are not recorded;
- logs use redacted provider, profile, and route identifiers;
- evidence export is outside Phase 4 unless required for repository compatibility fixtures and contains only synthetic data.

## Contract ownership by milestone

### M4.1 owns

- provider trust status;
- provider definition;
- binary identity;
- capability declaration;
- legacy compatibility and migration boundaries;
- provider disable and rollback policy.

### M4.2 owns

- evidence run and observation records;
- evidence collection privacy and retention;
- comparison and evidence-derived capability status.

### M4.3 owns

- consistency rules;
- profile health derivation for identity and window behavior;
- evidence freshness after profile changes.

### M4.4 owns

- route and controlled network observation fields;
- network health derivation;
- compatibility matrix generation.

A milestone may add fields needed by its scope, but must not change the meaning of an earlier frozen field without a planning update and compatibility analysis.

## Required documentation with implementation

Each implementation PR that changes these contracts must update:

- the relevant module document;
- `docs/STATUS.md`;
- migration and rollback notes when persisted data changes;
- tests proving compatibility and failure behavior;
- this document when a logical contract meaning changes.

Implementation details that do not change logical meaning belong in module documents or code comments rather than expanding this planning contract.