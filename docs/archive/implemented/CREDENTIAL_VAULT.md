# Operating-system credential vault

Veilium stores proxy passwords in the user's operating-system credential store. It does not write secrets to `profiles.json`, `credentials.json`, application logs, Wails bootstrap data, or browser-preview state.

## Platform providers

Veilium uses `github.com/zalando/go-keyring` as a small cross-platform adapter:

- Windows: Windows Credential Manager;
- macOS: Keychain through the system `security` utility;
- Linux and BSD: the Secret Service D-Bus interface, normally supplied by GNOME Keyring or another compatible service.

If the platform keyring is unavailable or locked, the operation fails. There is no plaintext-file fallback.

## Data split

`credentials.json` contains only reviewable metadata:

```json
{
  "id": "cred_...",
  "name": "US proxy",
  "username": "alice",
  "createdAt": "...",
  "updatedAt": "..."
}
```

The password is stored under the application service `Veilium Browser` and an opaque account key derived from the credential ID. The real proxy username remains metadata so users can identify records and so a future authenticated proxy bridge can combine it with the secret internally.

## Transaction behavior

Credential creation, rotation, and deletion coordinate the system keyring with metadata persistence:

- creation removes the newly written keyring item if metadata persistence fails;
- rotation restores the previous keyring value if metadata persistence fails;
- deletion reads the previous value first and restores it if metadata persistence fails;
- stale metadata whose keyring value is already missing can still be deleted;
- keyring errors never trigger an insecure fallback.

Metadata is written through a private temporary file and replaced with mode `0600` inside the private Veilium data directory.

## UI boundary

The desktop UI may submit a new password but never receives an existing password. Editing a record with a blank password changes only the display name or username. Password rotation requires a new value.

Browser preview mode cannot create, update, or delete credentials. This prevents secrets from being retained in ordinary browser JavaScript state.

## Profile references

Profiles store only `proxy.credentialRef`. Creating or updating a profile rejects:

- unknown credential IDs;
- inline credentials in a proxy URL;
- a credential attached to `direct://`;
- credential-backed routes without a supported proxy scheme.

A credential cannot be deleted while any profile references it.

## Deliberate limits

This phase does not pass secrets to Chromium and does not start an authenticated proxy bridge. Credential-backed profiles remain blocked by the runtime planner until the separate loopback bridge phase is implemented and reviewed.
