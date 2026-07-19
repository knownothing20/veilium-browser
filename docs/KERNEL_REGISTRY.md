# Verified local kernel registry

Veilium treats browser kernels as explicit managed dependencies rather than silently trusting arbitrary executable paths. The registry supports one exact reviewed official Chromium package and a separate generic custom-executable path.

## Three separate states

Provider Contract v2 keeps these concepts separate:

1. **Binary and package integrity** — whether the managed executable and, when applicable, its complete package still match their recorded identities.
2. **Provider trust** — whether the Provider is `reviewed`, `custom`, `legacy`, `disabled`, or `invalid`.
3. **Capability status** — whether one capability is `verified`, `partial`, `unsupported`, `unverified`, or `failed` for the exact Provider/version/platform/binary identity.

An integrity status of `verified` does not by itself grant reviewed Provider trust or advanced fingerprint support.

## Reviewed package installation

`official-chromium-snapshot-win64` is the only reviewed browser Provider. It is restricted to Chromium `152.0.7960.0`, Snapshot revision `1664436`, Windows amd64, and the immutable release identity documented in [`OFFICIAL_CHROMIUM_PROVIDER.md`](OFFICIAL_CHROMIUM_PROVIDER.md).

The reviewed path cannot use the generic single-file importer. It must use the pinned installer, which:

1. requires explicit license and third-party-notice acknowledgement;
2. downloads only the embedded official HTTPS archive URL;
3. verifies exact archive size and SHA-256;
4. rejects traversal, links, special entries, duplicate targets, and unexpected archive layout;
5. extracts and verifies the complete `chrome-win` package;
6. verifies the `chrome.exe` path, size, and SHA-256;
7. verifies all 261 files through the deterministic Package Tree identity;
8. atomically activates the package in Veilium's private kernel directory;
9. registers the exact Snapshot, archive, executable, and package metadata.

The source archive is not bundled, moving-latest resolution is forbidden, and a healthy repeated installation is idempotent.

## Custom executable import

1. The user selects a local Chromium executable in the Wails desktop application.
2. Veilium resolves the versioned custom or legacy Provider contract.
3. The source must be a regular file; symbolic links are rejected.
4. The file is copied to the private application kernel directory through a temporary file.
5. Veilium records the destination SHA-256, byte size, Provider, version, and verification time.
6. Re-importing the same digest/Provider/version is idempotent.
7. The Provider contract and stored binary fields derive a versioned `ProviderBinaryIdentity` for policy and diagnostics.

The original source path is not persisted. Only the managed destination is stored. Custom and legacy imports cannot inherit the reviewed official package identity.

## Provider compatibility

- `official-chromium-snapshot-win64` is reviewed only for its exact pinned Windows amd64 package and protected Evidence chain. Generic managed launch and Evidence are supported; stock Chromium advanced fingerprint overrides remain unsupported.
- `custom-chromium` is the generic local-import path. It can launch with generic settings but has no Veilium-reviewed advanced fingerprint claims.
- `native-chromium` and `patched-chromium` remain legacy compatibility IDs. Historical declarations are not silently upgraded.
- unknown, disabled, invalid, contradictory, modified, missing, or incompatible states fail closed.

Existing `kernels.json` records remain readable because Provider trust is derived from the versioned Provider catalog rather than by rewriting old records in place.

## Integrity states

- `verified`: the executable and, for package records, the complete package match all recorded identities.
- `modified`: the executable or any package dependency changed, a file was added or removed, or a path became a link or another unsupported file type.
- `missing`: the executable or required package content no longer exists.

Registered kernels are re-verified before profile save and launch-plan generation. Modified or missing kernels are blocked. The reviewed package tamper regression changes a dependency rather than `chrome.exe` and must still produce `modified`.

## Replacement and rollback policy

A Provider replacement must explicitly name its predecessor. A reviewed replacement cannot silently change source, license, platform, archive, executable, package tree, or Evidence while retaining the same identity.

Installer failure leaves the previous healthy record and package untouched. Temporary downloads, extraction directories, and partially activated packages are removed. Failed verification disables new reviewed claims while preserving metadata needed for diagnosis and recovery.

## Deletion safety

A kernel cannot be removed while a profile references its registry ID. Removal first moves its managed directory to a private quarantine path, persists metadata deletion, and then removes the quarantined files. Persistence failure rolls the move back.
