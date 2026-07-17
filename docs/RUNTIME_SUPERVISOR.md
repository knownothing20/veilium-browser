# Runtime supervisor

Veilium starts browser processes only after a profile resolves to a registered kernel whose managed copy still passes integrity verification.

## Start sequence

1. Load the profile and require a registered kernel ID.
2. Re-verify the managed kernel SHA-256 and size.
3. Require the profile user-data directory to be the Veilium-managed path for that profile.
4. Create the private user-data directory when it does not exist and reject symlinked paths.
5. Remove only a stale regular `DevToolsActivePort` file; reject symlinked or abnormal replacements.
6. Build a launch plan with `--remote-debugging-address=127.0.0.1` and `--remote-debugging-port=0`.
7. Reject plans that require an unavailable proxy bridge.
8. Start the browser under a platform process-tree container and write stdout/stderr to a private log.
9. Wait for Chromium to write `DevToolsActivePort`, then parse the assigned port and browser path.
10. Poll `http://127.0.0.1:<port>/json/version` and accept only a loopback WebSocket debugger URL on that exact port.

Using Chromium's assigned port removes the previous gap between Veilium releasing a reserved socket and Chromium attempting to bind it.

## Process-tree ownership

### Linux and macOS

The browser starts in a new process group. Interrupt and force-stop operations target the entire group rather than only the original browser parent process.

### Windows

Veilium creates a Job Object configured with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`, starts the browser in a new process group, and assigns the process to the Job immediately. Force-stop terminates the Job, and closing the Job after the parent exits removes remaining descendants.

There is still a very small interval between Windows process creation and Job assignment because Go's standard `os/exec` API does not expose the suspended primary-thread handle. The automated test delays child creation until assignment and proves that descendants created after assignment are terminated. A future native suspended-process launcher can remove this final micro-window if real Chromium testing shows it is material.

## Stop and shutdown

Veilium first requests graceful interruption. Unsupported or unresponsive processes fall back to process-tree termination after a bounded timeout. The Wails shutdown hook attempts to stop every active browser before the desktop application exits.

Profiles cannot be edited or deleted while their browser session is active. This prevents a running process from silently diverging from the profile configuration shown in the workspace.

## Automated runtime tests

CI now verifies:

- `DevToolsActivePort` stale-file removal, parsing and symlink rejection;
- a real supervised helper process that writes `DevToolsActivePort` and serves `/json/version`;
- dynamic-port discovery and exact loopback WebSocket validation;
- Unix child-process cleanup through process groups;
- Windows child-process cleanup through a Job Object;
- duplicate-start, stop, readiness-failure and proxy-bridge rejection behavior.

The helper process is not a fingerprint Chromium binary. A separate approved test-kernel matrix is still required before claiming compatibility with a specific third-party kernel.

## Security boundaries

- CDP is loopback-only and redirects are rejected during readiness checks.
- The debugger WebSocket must remain on the Chromium-discovered loopback port.
- Environment keys and null bytes in executable paths, arguments and environment values are rejected.
- Only regular managed kernel files may execute.
- Runtime logs use private file permissions and unique per-start names.
- Profile data paths outside Veilium's managed profile root are not executable.

## Deliberate limits

This feature does not provide authenticated proxy bridges, Xray/sing-box processes, automatic crash restart, remote CDP access, extension management, cookie import, or real-window dimension synchronization. Those remain separate reviewed features.
