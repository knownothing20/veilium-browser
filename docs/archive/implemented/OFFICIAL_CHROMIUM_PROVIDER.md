# Reviewed official Chromium Provider

Veilium exposes one exact official Chromium Snapshot as its first browser Provider with `reviewed` trust. This path is optional, user initiated, Windows amd64 only, and does not resolve a moving latest build.

## Exact reviewed identity

| Field | Pinned value |
| --- | --- |
| Provider ID | `official-chromium-snapshot-win64` |
| Provider revision | `1` |
| Chromium version | `152.0.7960.0` |
| Snapshot revision | `1664436` |
| Platform | `windows` |
| Architecture | `amd64` |
| Archive | `chrome-win.zip` |
| Archive size | `343585547` bytes |
| Archive SHA-256 | `d224019b7cbc115951b0f5dce8cf232c37244881a3eb969c010e457aa369332f` |
| Archive entries | `261` files |
| Expanded package size | `814120936` bytes |
| Package Tree SHA-256 | `312cb62d6bfab56ecfa52c4e8047dd33c05a1c17c7e44bc2afd9be436854a8dc` |
| Executable | `chrome-win/chrome.exe` |
| Executable size | `2926080` bytes |
| Executable SHA-256 | `5093988c8fdf969494f921deb32c177dbe5ed88cc101346852d93e760041e5c9` |
| License | Chromium `BSD-3-Clause`; bundled third-party notices remain available through `chrome://credits/` |

The official HTTPS archive URL, source page, license URL, review time, limitations, and every value above are embedded in `internal/kernelrelease/releases.json`. Normal installation and launch never query `LAST_CHANGE`, a release channel, or another moving version source.

## User-controlled installation

1. Open **Kernel registry** in the Wails desktop application.
2. Review the version, Snapshot revision, archive size, complete-package identity, platform boundary, and limitations.
3. Explicitly acknowledge the Chromium license, third-party notices, Snapshot limitations, and download action.
4. Press **Download, verify, and install**.
5. Veilium downloads only the embedded archive URL.
6. The archive byte size and SHA-256 must match before extraction.
7. Every ZIP entry is inspected. Absolute paths, traversal, non-canonical paths, links, special files, duplicate targets, and unexpected layout fail closed.
8. Veilium extracts the complete `chrome-win` package to a private temporary directory, verifies `chrome.exe`, then calculates the deterministic identity of all 261 files.
9. The verified package is atomically activated in Veilium's private kernel directory and registered as the exact reviewed Provider.

Repeated installation is idempotent when the exact healthy package is already registered. Veilium does not bundle the archive, download it in the background, or update it silently.

## Complete-package integrity

Chromium cannot be represented safely by a copied `chrome.exe` alone. The reviewed identity therefore includes the complete extracted package.

The Package Tree digest is calculated by sorting canonical package-relative file paths and hashing this sequence for each regular file:

```text
path + NUL + decimal byte size + NUL + file SHA-256 + newline
```

Any changed, missing, added, linked, or non-regular dependency changes the package state to `modified` or `missing`. A reviewed Provider cannot be imported through the generic single-executable path. Registered packages are re-verified before profile save and launch planning.

## Protected Evidence chain

Windows Required CI performs the product installation flow against the same pinned archive and passes the resulting managed `chrome.exe` to:

1. M4.2 real-browser identity Evidence;
2. M4.3 managed-window and consistency Evidence;
3. M4.4 controlled Network Evidence;
4. a dependency-tamper regression that must downgrade the package to `modified`.

The successful Evidence packet records the exact Provider revision, browser and Snapshot versions, archive digest, executable digest, Package Tree digest, file count, and expanded size. CI failure diagnostics record the browser stage and bounded Chromium logs without weakening any Evidence assertion.

## Compatibility boundary

Compatibility entries remain keyed by Provider ID and revision, browser version, operating system, architecture, binary-identity digest, and capability ID. Entries using this Provider are rejected unless their Provider revision, version, operating system, architecture, and reviewed trust match the embedded release exactly.

The protected Network Evidence fixture proves the collection path for the pinned Windows amd64 binary and a controlled direct route. It does not claim compatibility for arbitrary proxies, networks, other Chromium builds, Linux, macOS, arm64, or future Snapshot revisions.

## Deliberate limitations

- Chromium Snapshots are best-effort builds from arbitrary source revisions and are not stable-channel Chrome releases.
- Reviewed trust applies only to the exact archive, complete package, executable, Windows amd64 platform, and recorded Evidence chain above.
- Stock Chromium advanced fingerprint overrides remain `unsupported`; the reviewed Provider supports generic managed launch and runtime Evidence only.
- SHA-256 and official-bucket provenance do not provide publisher signatures, transparency logs, or reproducible-build guarantees.
- Veilium records the user's acknowledgement action but does not provide legal advice.
- A future replacement requires a new reviewed identity and cannot silently inherit this Provider claim.
