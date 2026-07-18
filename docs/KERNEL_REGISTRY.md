# Verified local kernel registry

Veilium treats browser binaries as explicit local dependencies rather than silently downloading or trusting an arbitrary executable path.

## Three separate states

Provider Contract v2 keeps these concepts separate:

1. **Binary integrity** — whether the managed executable still matches its imported SHA-256 and byte size.
2. **Provider trust** — whether the provider is `reviewed`, `custom`, `legacy`, `disabled`, or `invalid`.
3. **Capability status** — whether one capability is `verified`, `partial`, `unsupported`, `unverified`, or `failed` for the exact provider/version/platform identity.

An integrity status of `verified` does not mean that a provider or fingerprint capability is reviewed. A custom or legacy binary can be integrity-verified while every advanced fingerprint capability remains unverified or unsupported.

## Import flow

1. The user selects a local Chromium executable in the Wails desktop application.
2. Veilium resolves the versioned provider contract.
3. The source must be a regular file; symbolic links are rejected.
4. The file is copied to the private application kernel directory through a temporary file.
5. Veilium records the destination SHA-256, byte size, provider, version and verification time.
6. Re-importing the same digest/provider/version is idempotent.
7. The provider contract and the stored binary fields derive a versioned `ProviderBinaryIdentity` view for policy and diagnostics.

The original source path is not persisted. Only the managed destination is stored.

## Provider compatibility

- `custom-chromium` is the generic local-import path. It can launch with generic settings but has no Veilium-reviewed advanced fingerprint claims.
- `native-chromium` is retained as a legacy compatibility ID and points users toward the generic custom path.
- `patched-chromium` is retained as a legacy compatibility ID. Former boolean capability declarations are not silently upgraded to reviewed status.
- unknown, disabled, invalid, contradictory, modified, or missing states fail closed.

Existing `kernels.json` records remain readable because provider trust is derived from the versioned provider catalog rather than by rewriting old records in place. A future reviewed provider must use an exact source, license, platform, architecture, version, executable identity, provenance rule, and evidence chain.

## Integrity states

- `verified`: current digest and size match the imported record.
- `modified`: the managed file changed, became a symlink, or is no longer a regular file.
- `missing`: the managed file no longer exists.

Registered kernels are re-verified before profile save and launch-plan generation. Modified or missing kernels are blocked. Integrity verification alone never grants reviewed trust.

## Replacement and rollback policy

A provider replacement must explicitly name its predecessor. A reviewed replacement cannot silently change upstream source or license while retaining the same reviewed identity. Failed verification disables new reviewed claims but preserves profile and kernel records for diagnosis and recovery.

## Deletion safety

A kernel cannot be removed while a profile references its registry ID. Removal first moves its managed directory to a private quarantine path, persists metadata deletion, and then removes the quarantined files. Persistence failure rolls the move back.

## Deliberate limits

M4.1 defines and enforces the provider contract boundary. It does not select a production reviewed browser provider, download kernels, validate publisher signatures, collect real-browser evidence, or claim that any third-party kernel is trustworthy. Those claims require the later Phase 4 evidence milestones.
