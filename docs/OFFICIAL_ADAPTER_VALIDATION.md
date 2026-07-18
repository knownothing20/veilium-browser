# Official adapter validation

Veilium keeps a small embedded release manifest for reviewed Xray and sing-box builds. The manifest is not an auto-update feed. It is an immutable trust record used to identify an imported executable and to reproduce CI validation.

## Current pins

- Xray `v26.3.27`: official Linux amd64 and Windows amd64 release assets.
- sing-box `v1.13.12`: official Linux amd64 and Windows amd64 release assets.

Each platform entry pins all of the following:

- official GitHub repository, tag, asset name, and HTTPS release URL;
- archive byte size and SHA-256;
- executable path inside the archive;
- extracted executable byte size and SHA-256;
- version command and native configuration-check command.

The release asset digest returned by GitHub's API was compared with an independently calculated download digest before the manifest was committed.

## Import identity

Veilium always calculates the SHA-256 and byte size of the executable selected by the user. A record is marked `official` only when the declared adapter kind and version plus the executable digest and size exactly match one embedded platform pin.

Typing an official source URL or license value does not grant official status. When an executable matches, Veilium replaces user-declared provenance with the canonical asset URL and license stored in the manifest. Other binaries remain supported as `custom local` adapters but cannot pass the official check.

## Native configuration check

The desktop **Official check** action:

1. repeats managed-file integrity verification;
2. requires the current operating system and architecture to match the pin;
3. runs the binary's native version command and verifies the pinned version;
4. generates the reviewed Veilium protocol samples through the production providers;
5. writes each sample to a private temporary file;
6. invokes Xray `run -test -config <file>` or sing-box `check -c <file>`;
7. removes all temporary files and returns only a non-secret check report.

Adapter command output from configuration checks is deliberately not returned to the desktop UI because upstream errors may repeat configuration values. Only the version output is retained, with a strict size bound.

## CI reproduction

The repository fetch script reads the embedded manifest, downloads only the exact pinned asset, verifies archive size and SHA-256, safely extracts only the expected regular executable, and verifies the executable size and SHA-256.

Linux and Windows CI run the native configuration checks against the real pinned binaries. Linux CI additionally launches each official adapter with a local direct SOCKS5 configuration and requires a hosted headless Chromium to render a token from a local HTTP server through that SOCKS5 endpoint. A negative control first confirms Chromium cannot reach the token when the proxy port is unavailable.

This local smoke test validates the browser-to-adapter handoff and process compatibility without depending on or probing a public proxy service.

## Deliberate limits

- The manifest does not automatically download or update binaries in the desktop application.
- GitHub release metadata and SHA-256 pins are not a substitute for publisher code signing or reproducible builds.
- Only Linux and Windows amd64 are pinned in this version.
- A real external VLESS, VMess, Trojan, Shadowsocks, Hysteria2, TUIC, or AnyTLS endpoint is still required for endpoint-specific acceptance testing.
