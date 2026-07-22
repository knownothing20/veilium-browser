# Profile Portability and Multi-Profile Tools

Status: Phase 5 implementation in PR #59

This guide describes the user-visible M5.3 and M5.4 desktop workflows. These features manage Profile definitions and lifecycle metadata only. They do not export browser data, cookies, credentials, binaries, runtime logs, or Evidence.

## Portable Profile export

Open **Local recovery** and use **Portable Profiles and templates** for a single Profile, or open **Multi-Profile tools** for a bounded multi-Profile export.

### New identity

Use **New identity** for normal transfer. The artifact excludes the source fingerprint seed. Import creates a new local Profile ID, managed directory, and seed.

### Preserve identity

Use **Preserve identity** only when the same identity will not run simultaneously elsewhere. The destination still receives a new local Profile ID and managed directory. Provider trust, compatibility, health, and Evidence are not transferred.

### Bulk export

1. Select available, stopped, unlocked Profiles.
2. Choose an existing destination folder.
3. Select the identity mode.
4. Start the export.

Veilium writes one deterministic `.veilium-profile.json` file per successful Profile. It never overwrites an existing file. A partial result lists each succeeded, skipped, cancelled, or failed Profile separately.

## Import

1. Pick one portable Profile JSON file.
2. Review exclusions, limitations, identity mode, and dependency matches.
3. Select a currently verified local Kernel and any required adapter.
4. Select an existing local operating-system-vault credential when the route requires one.
5. Create the imported Profile.

Import never overwrites an existing Profile. The new Profile remains `draft` until current local validation passes. Browser and Network Evidence are not imported. Current local validation remains authoritative.

## Templates

A template stores reusable non-secret defaults without a reusable fingerprint seed. Applying a template always creates a new Profile ID, managed directory, and seed. The result remains `draft` until current local validation passes. Templates never contain browser data or credential values.

## Bounded lifecycle actions

The **Bulk archive, unarchive, or trash** surface accepts a fixed set of stopped, unlocked Profiles and executes one authoritative M5.1/M5.2 lifecycle operation for each item.

- **Archive** accepts `available` and `draft` Profiles and preserves the exact origin state for later unarchive.
- **Unarchive** accepts only `archived` Profiles and restores the recorded `available` or `draft` origin state.
- **Move to recoverable trash** accepts `available`, `draft`, or `archived` Profiles, requires an exact confirmation phrase, and retains the browser data under Veilium private trash with a bounded retention deadline.
- Bulk permanent deletion is intentionally unavailable. Permanent deletion remains an explicit single-Profile action with exact confirmation.
- Every child operation keeps its own journal record, lock, cancellation boundary, result, and recovery status. A repeated request reuses the same deterministic child operations.
- Cancellation stops the next Profile from starting. A partial selection remains a truthful partial result rather than being reported as complete.

## Multi-Profile metadata and health

The fixed selection accepts only stopped Profiles without a lifecycle lock.

- **Bulk metadata** replaces a group and adds or removes bounded tags.
- **Bulk health refresh** performs read-only lifecycle, Kernel, route, fingerprint, consistency, and managed-data checks.
- Cancellation prevents the next Profile from starting. Each Profile keeps its own result.

Bulk start, scheduling, proxy rotation, and unattended automation are not included.

## Storage inventory and manual plans

The storage view counts opaque managed Profile files and reports missing, incomplete, orphaned, or unsafe entries. It also shows verified snapshot and recoverable-trash totals.

Repair plans are recommendations only:

- review a verified snapshot for missing Profile data;
- inspect missing data when no verified snapshot exists;
- review ownership before acting on an orphan directory;
- manually inspect unsafe links, reparse entries, or special files;
- rerun the bounded inventory after resolving incomplete-scan limitations.

Veilium does not automatically restore, move, quarantine, repair, or delete anything from these plans.

## Safety

- Do not point Veilium at a daily Chrome or Edge user-data directory.
- Do not treat a portable artifact as a browser-data backup.
- Keep credential values in the destination operating-system vault.
- Review every preserve-identity warning.
- Use Local recovery snapshot and trash functions for same-machine recovery, not portable definitions.
- Use bulk trash only when every selected Profile is intended to leave ordinary use; restore and permanent deletion remain deliberate follow-up actions.
