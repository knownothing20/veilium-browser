# Verified local kernel registry

Veilium treats browser binaries as explicit local dependencies rather than silently downloading or trusting an arbitrary executable path.

## Import flow

1. The user selects a local Chromium executable in the Wails desktop application.
2. Veilium validates the provider/version capability contract.
3. The source must be a regular file; symbolic links are rejected.
4. The file is copied to the private application kernel directory through a temporary file.
5. Veilium records the destination SHA-256, byte size, provider, version and verification time.
6. Re-importing the same digest/provider/version is idempotent.

The original source path is not persisted. Only the managed destination is stored.

## Integrity states

- `verified`: current digest and size match the imported record.
- `modified`: the managed file changed, became a symlink, or is no longer a regular file.
- `missing`: the managed file no longer exists.

Registered kernels are re-verified before profile save and launch-plan generation. Modified or missing kernels are blocked.

## Deletion safety

A kernel cannot be removed while a profile references its registry ID. Removal first moves its managed directory to a private quarantine path, persists metadata deletion, and then removes the quarantined files. Persistence failure rolls the move back.

## Deliberate limits

This feature does not download kernels, validate publisher signatures, execute browser processes or claim that any third-party kernel is trustworthy. Future remote catalogs require signed manifests, pinned provenance and licensing review.
