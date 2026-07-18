# Optional official adapter installer

Veilium can install the exact Xray and sing-box builds already pinned by the embedded official-release manifest. The installer is disabled by default: it performs no background network requests, version checks, downloads, or updates.

## User-controlled flow

1. Open **Proxy adapters** in the Wails desktop application.
2. Review the pinned project, version, platform asset, archive size, and SPDX license identifier.
3. Check the explicit license acknowledgement.
4. Press **Download, verify, and install**.
5. Veilium downloads only the exact HTTPS asset URL stored in the embedded manifest.
6. The archive and extracted executable must match their pinned byte sizes and SHA-256 values.
7. The executable is imported through the normal private adapter store and must receive the embedded official identity.

A healthy matching official adapter is returned without another network request, making repeated installation requests idempotent.

## Download boundary

- The initial URL must be the embedded `github.com/<owner>/<repo>/releases/download/<tag>/<asset>` URL.
- Redirects are bounded and accepted only on approved GitHub release/CDN hosts over HTTPS.
- Authorization and cookie headers are removed on redirects.
- The response is bounded by the exact pinned archive size.
- Context cancellation and the desktop timeout abort the request.
- Partial downloads and all temporary installer files are removed.

## Archive extraction boundary

The installer supports the pinned ZIP and tar.gz assets only. Every archive entry is inspected before the expected executable is accepted.

It rejects:

- absolute paths, path traversal, backslashes, null bytes, and non-canonical paths;
- symbolic links, hard links, devices, sockets, named pipes, and other special entries;
- duplicate or missing executable entries;
- executable size or digest mismatches.

Only the expected regular executable is written to a private temporary directory. The complete archive is never unpacked into the application directory.

## Deliberate limits

- There is no automatic update mechanism.
- The installer does not accept user-supplied download URLs.
- Only platforms and architectures present in the embedded manifest are offered.
- SHA-256 and GitHub release provenance do not replace publisher signatures, transparency logs, or reproducible builds.
- License acknowledgement records the current user action only; Veilium does not provide legal advice.
