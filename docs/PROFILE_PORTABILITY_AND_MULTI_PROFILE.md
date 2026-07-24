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

The **Template maintenance** surface allows safe edits to:

- the template name;
- the default new-Profile name;
- group, notes, and bounded tags.

An update preserves the template ID and creation timestamp while advancing its update timestamp. It also preserves the reviewed Kernel and adapter requirements, route defaults, credential requirement, and Provider-compatible fingerprint settings. The seed remains blank and the template remains `new-identity` only. Case-insensitive duplicate template names, invalid text, and over-bound tag sets fail before the catalog is replaced. Create a new template from a validated Profile when dependency or route defaults must change.

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

## Operation history and redacted reports

The **Phase 5 operation history** is projected directly from the authoritative M5.1 lifecycle journal. Each row shows the fixed Profile selection, current stage, terminal or running status, per-item outcome counts, and whether safe cancellation is available.

Use **Export redacted report** to save a point-in-time `.veilium-operation-report.json` file for one operation. The report:

- includes the operation type, selected Profile IDs, status, stage, timestamps, item outcomes, progress totals, limitations, recovery actions, application version, and platform;
- includes a deterministic SHA-256 over the report payload so later edits are detectable;
- excludes the idempotency key, local staging and quarantine references, browser contents, credential values, proxy secrets, runtime logs, and Evidence payloads;
- replaces absolute or path-like output and recovery references with a redacted marker;
- never overwrites an existing destination file.

An operation report is diagnostic information only. It is not a portable Profile, browser-data backup, health certificate, Provider-trust record, compatibility proof, or Evidence export. A report exported while an operation is still running is a point-in-time snapshot and may differ from the final journal state.

## Storage inventory and manual plans

The storage view counts opaque managed Profile files and reports missing, incomplete, orphaned, or unsafe entries. It also shows verified snapshot and recoverable-trash totals.

Repair plans are recommendations only:

- review a verified snapshot for missing Profile data;
- inspect missing data when no verified snapshot exists;
- review ownership before acting on an orphan directory;
- manually inspect unsafe links, reparse entries, or special files;
- rerun the bounded inventory after resolving incomplete-scan limitations.

Veilium does not automatically restore, move, quarantine, repair, or delete anything from these plans.

## Managed storage locations

The **Storage locations** surface shows the fixed local paths Veilium derives from its configured data root, including Profile browser data, browser Kernels, adapter packages and runtime state, logs, lifecycle records, snapshots, recoverable trash, credential metadata, and portable templates.

- It reports the data-root volume and, on Windows, whether that volume matches the detected system volume.
- Each expected location is reported as present, not yet created, an unsafe link, an unexpected entry type, or unavailable for inspection.
- A missing optional location normally means that feature has not created its directory or catalog yet; it is not created merely by opening this view.
- **Copy path** copies the displayed fixed path for local diagnostics.
- The view does not enumerate arbitrary directories, open a filesystem browser, move data, change the data root, create links, clean storage, repair findings, or delete files.
- These absolute local paths remain installation-specific and are excluded from portable Profile artifacts and redacted operation reports.

Use this view before installing large Chromium packages or creating many Profiles to confirm which drive will hold Veilium-managed data. Changing the data root is not part of this Phase 5 surface.

## Safety

- Do not point Veilium at a daily Chrome or Edge user-data directory.
- Do not treat a portable artifact as a browser-data backup.
- Keep credential values in the destination operating-system vault.
- Review every preserve-identity warning.
- Use Local recovery snapshot and trash functions for same-machine recovery, not portable definitions.
- Use bulk trash only when every selected Profile is intended to leave ordinary use; restore and permanent deletion remain deliberate follow-up actions.
