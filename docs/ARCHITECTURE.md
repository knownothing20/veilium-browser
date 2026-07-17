# Architecture

## Design principles

1. **Clean-room implementation.** Reference projects inform requirements, not copied source.
2. **Provider contracts over guessed flags.** Every fingerprint option must be supported by the chosen provider and major version.
3. **Identity consistency over maximum randomness.** Stable, coherent profiles are safer than changing every surface at every launch.
4. **Local-first and least privilege.** API binds to loopback and uses bearer authentication by default.
5. **Replaceable runtimes.** Chromium kernels and proxy bridges are adapters, not application foundations.

## Layer model

```text
Desktop UI (future Wails + React)
             |
Local REST / future MCP
             |
Profile service and policy validation
             |
Launch planner ---- Proxy route resolver
             |
Kernel provider ---- Runtime supervisor (future)
```

## Packages

- `internal/domain`: stable data contracts.
- `internal/fingerprint`: provider capability catalog, validation, and provider-specific arguments.
- `internal/proxy`: native-versus-bridge routing decision.
- `internal/launch`: produces a redacted, reviewable launch plan.
- `internal/profile`: atomic local persistence.
- `internal/api`: loopback-only authenticated API.
- `cmd/veilium`: service entry point.

## Security boundaries

- Browser debugging is always bound to `127.0.0.1` in generated plans.
- API requests are limited to 1 MiB and reject unknown JSON fields.
- API token comparison is constant-time.
- Inline proxy passwords are rejected. Profiles carry only a future operating-system-vault `credentialRef`.
- Remote API binding requires an explicit future opt-in and must include additional controls before production use.

## Deliberately not implemented in Phase 1

- Browser binary download or execution.
- Xray, sing-box, or Mihomo process supervision.
- Cookie import/export.
- Cloud sync.
- MCP page-control tools.
- Desktop UI.

These features require separate threat models, licensing review, and platform-specific tests.
