# Runtime supervisor

Veilium starts browser processes only after a profile resolves to a registered kernel whose managed copy still passes integrity verification.

## Start sequence

1. Load the profile and require a registered kernel ID.
2. Re-verify the managed kernel SHA-256 and size.
3. Require the profile user-data directory to be the Veilium-managed path for that profile.
4. Create the private user-data directory when it does not exist and reject symlinked paths.
5. Allocate an ephemeral IPv4 loopback port.
6. Build a launch plan containing the exact `127.0.0.1` CDP address and allocated port.
7. Reject plans that require a proxy bridge that has not been implemented.
8. Start the browser with stdout and stderr written to a private per-start log file.
9. Poll `http://127.0.0.1:<port>/json/version` until ready or until the readiness timeout expires.
10. Accept only a loopback WebSocket debugger URL using the same allocated port.

Runtime sessions move through `starting`, `ready`, `stopping`, `exited`, and `failed` states. PID, CDP endpoint, browser version, log path, timestamps, exit code, and the latest error are kept in memory for the current desktop run.

## Stop and shutdown

Veilium first sends an interrupt signal. Platforms that do not support that signal, or processes that do not exit within the stop timeout, fall back to process termination. The Wails shutdown hook attempts to stop every active browser before the desktop application exits.

Profiles cannot be edited or deleted while their browser session is active. This prevents a running process from silently diverging from the profile configuration shown in the workspace.

## Security boundaries

- CDP is loopback-only and redirects are rejected during readiness checks.
- The debugger WebSocket must remain on the exact allocated loopback port.
- Environment keys and null bytes in executable paths, arguments, and environment values are rejected.
- Only regular managed kernel files may execute.
- Runtime logs use private file permissions and unique per-start names.
- Profile data paths outside Veilium's managed profile root are not executable.

## Deliberate limits

This feature does not provide authenticated proxy bridges, Xray/sing-box processes, automatic crash restart, remote CDP access, extension management, cookie import, or real-window dimension synchronization. Those remain separate reviewed features.
