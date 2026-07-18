# Architecture

## Design principles

1. **Clean-room implementation.** Reference projects inform requirements, not copied source.
2. **Provider contracts over guessed flags.** Every fingerprint option must be supported by the chosen provider and major version.
3. **Identity consistency over maximum randomness.** Stable, coherent profiles are safer than changing every surface at every launch.
4. **Local-first and least privilege.** Control surfaces bind to loopback and require authentication by default.
5. **Replaceable runtimes.** Chromium kernels and proxy runtimes are adapters, not application foundations.
6. **Fail-closed execution.** Missing integrity, unsupported fields, unavailable secrets, and failed readiness checks block launch.
7. **Evidence-backed capabilities.** UI availability and launch arguments are not sufficient proof of real browser behavior.

## Current layer model

```text
Wails + React desktop workspace
            |
Direct Go bindings / authenticated loopback REST
            |
Profile service, policy validation and persistence
            |
Launch planner -------- Network route resolver
      |                         |
Kernel registry          Credential vault
      |                         |
Browser supervisor       Proxy bridge / adapter runtime
      |                         |
Managed Chromium         Xray / sing-box providers
            \             /
             Runtime status, diagnostics and cleanup
```

A future MCP or broader automation surface must use the same policy and authorization layer rather than bypassing it.

## Package responsibilities

- `internal/domain`: stable profile, fingerprint, proxy, and launch contracts.
- `internal/fingerprint`: provider capability catalog, validation, and provider-specific arguments.
- `internal/profile`: atomic local profile persistence.
- `internal/kernel`: managed Chromium imports, integrity records, and in-use protection.
- `internal/launch`: redacted, reviewable browser launch plans.
- `internal/supervisor`: browser process ownership, readiness, runtime status, logs, and cleanup.
- `internal/credential`: metadata plus operating-system-backed secret storage.
- `internal/proxy`: route classification and native-versus-bridge decisions.
- `internal/proxybridge`: authenticated HTTP, HTTPS, and SOCKS5 loopback bridges.
- `internal/proxydiagnostics`: connectivity, timing, exit-IP, DNS-route, and WebRTC-policy analysis.
- `internal/adapter`: managed Xray and sing-box executable records.
- `internal/adapterruntime`: provider registry and supervised adapter lifecycle.
- `internal/xrayprovider`: constrained reviewed Xray configurations.
- `internal/singboxprovider`: constrained reviewed sing-box configurations.
- `internal/adapterrelease`: embedded official release pins.
- `internal/adaptervalidation`: native version and configuration checks.
- `internal/adapterinstaller`: explicit pinned download, verification, safe extraction, and import.
- `internal/desktop`: application service composition and desktop-facing operations.
- `internal/api`: loopback-only authenticated REST service.
- `cmd/veilium`: headless service entry point.
- `frontend`: desktop workspace and local browser-preview mode.

## Persistent-data boundaries

- Profile metadata is stored locally through atomic file replacement.
- Browser user data uses a Veilium-managed directory per profile.
- Kernel and adapter stores keep managed copies plus size and SHA-256 records.
- Secret values remain in the operating-system credential provider; persistent profile data stores references only.
- Private per-session proxy configurations and logs live in restricted runtime directories and are cleaned up with their sessions.

Persisted-contract changes must include compatibility, migration, failure, and rollback analysis before implementation.

## Runtime boundaries

- A browser starts only from a registered and currently verified kernel.
- CDP uses a Chromium-assigned port discovered through a validated `DevToolsActivePort` file.
- Debugging endpoints and local bridges bind to loopback.
- Xray and sing-box configurations are generated per session and expose only a local SOCKS5 endpoint to Chromium.
- Browser and adapter processes are owned through Unix process groups or Windows Job Objects.
- Application shutdown stops active sessions and removes private runtime material.

## Security boundaries

- API requests are bounded and unknown JSON fields are rejected.
- Authentication tokens are compared in constant time.
- Inline proxy credentials are rejected.
- Secrets must not appear in Chromium arguments, Bootstrap payloads, logs, or profile files.
- Downloaded official adapters are restricted to embedded release pins and exact archive and executable hashes.
- Automatic downloads and updates remain disabled unless separately designed and approved.
- Remote binding, telemetry, deployment, cloud sync, and automation permissions require explicit future threat models.

## Capability evidence boundary

A capability may be declared only when all applicable layers agree:

1. provider and Chromium-version contract;
2. profile validation;
3. generated launch configuration;
4. selected binary integrity and identity;
5. integration or real-runtime evidence;
6. clear UI reporting of supported and unsupported states.

The active roadmap and phase document determine when additional capability evidence is developed. Architecture does not set product priority by itself.

## Deferred architecture surfaces

The following remain deferred until approved by the active phase plan:

- broader browser fingerprint evidence and consistency testing;
- extension, cookie, and complete profile lifecycle management;
- stable Launch API and unified CDP abstraction;
- MCP and tool-level authorization;
- encrypted export, backup, or sync;
- signed application releases, updates, SBOM, provenance, and reproducible builds.

See `docs/PRODUCT.md`, `docs/ROADMAP.md`, `docs/STATUS.md`, and the active phase document for scope and priority.
