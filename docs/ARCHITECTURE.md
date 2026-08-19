# Veilium Architecture

Last updated: 2026-08-19

## Design principles

1. **Local first and least privilege.** Sensitive browser, credential, and control data stay local by default.
2. **Provider contracts over guessed behavior.** A browser capability is supported only for an explicit Provider/version combination.
3. **Evidence over UI claims.** A setting or launch argument is not proof that Chromium actually exposes the requested identity.
4. **Identity consistency over maximum randomness.** A Profile represents one stable, plausible environment.
5. **Exact runtime identity.** Reviewed kernels and adapters are pinned and verified by archive/executable/package identity.
6. **Replaceable runtimes.** Chromium, Xray, and sing-box are managed providers, not hidden assumptions.
7. **Fail closed.** Missing secrets, tampered packages, unsupported settings, and unverifiable states are rejected or clearly limited.
8. **Recoverable lifecycle.** Storage-changing operations use locks, journals, staging, verification, rollback, and reconciliation.

## Runtime layers

```text
Wails + React desktop workspace
            |
Desktop service / authenticated loopback API
            |
Profile + policy + lifecycle + portability
            |
Launch planner -------- Network route resolver
      |                         |
Provider contract         Credential vault
      |                         |
Kernel registry          Proxy bridge / adapters
      |                         |
Browser supervisor       Xray / sing-box runtime
      \                         /
       Real browser identity + Network Evidence
                    |
        Compatibility / Readiness
```

## Stable package responsibilities

- `internal/domain` — Profile, fingerprint, proxy, kernel, and launch contracts.
- `internal/fingerprint` — Provider capability catalog, validation, and provider-specific arguments.
- `internal/kernel` / `internal/kernelrelease` / `internal/kernelinstaller` — managed Chromium identity, release pins, import/install, integrity, and in-use protection.
- `internal/launch` — redacted, reviewable launch plans.
- `internal/supervisor` — process ownership, CDP readiness, runtime state, logs, and cleanup.
- `internal/credential` — metadata plus OS-backed secret storage.
- `internal/proxy` / `internal/proxybridge` / `internal/proxydiagnostics` — route resolution, authenticated loopback bridges, and connectivity diagnostics.
- `internal/adapter*`, `internal/xrayprovider`, `internal/singboxprovider` — managed proxy-adapter providers and supervised runtimes.
- `internal/evidence`, `internal/consistency`, `internal/networkevidence`, `internal/compatibility` — real-browser identity, context consistency, network observations, and exact-combination compatibility.
- `internal/lifecycle`, `internal/localrecovery`, `internal/portableprofile` and desktop services — recoverable Profile lifecycle, snapshots, restore, portable definitions, templates, and bounded multi-Profile operations.
- `frontend` — task-oriented desktop product surface; it must not invent backend trust or readiness state.

## Persistent-data boundaries

These remain separate:

- Profile metadata and non-secret configuration;
- Veilium-managed browser user-data directories;
- OS-vault secret values;
- Kernel and adapter package records and managed binaries;
- Evidence, compatibility, lifecycle journals, snapshots, trash, runtime logs, and staging data.

A portable Profile definition is not browser data, a local snapshot is not a cross-device identity guarantee, and archived Evidence cannot create current Provider trust.

## Security boundaries

- Local APIs and debugger-facing control use loopback-only endpoints.
- Inline proxy credentials are rejected; secrets are resolved from the OS vault at runtime.
- Logs, launch plans, Bootstrap payloads, reports, and portable exports must not expose secret values.
- Browser and adapter child processes remain owned by Veilium and are cleaned up on shutdown.
- Automatic remote control, telemetry, cloud sync, and background runtime updates require separate product/security approval.

## Current browser-provider boundary

The existing reviewed Provider is an exact Windows amd64 stock Chromium Snapshot. It proves managed launch, package identity, and the current Evidence chain, but it does **not** claim advanced fingerprint overrides.

The next architecture extension is a **separate Veilium Fingerprint Chromium Provider**. Stock and fingerprint Providers must coexist. New capability claims require exact Provider/revision/version/platform bindings and real-browser evidence.

## Future identity flow

```text
Identity Template + Root Seed
            |
Fingerprint Provider Contract
            |
Versioned launch configuration
            |
Exact Fingerprint Chromium package
            |
Top-level / iframe / worker Evidence
            |
Restart stability + seed separation
            |
Environment Readiness
```

This extension must reuse the current Kernel Registry, Supervisor, Evidence, Proxy, Security, and Lifecycle layers rather than bypassing them.
