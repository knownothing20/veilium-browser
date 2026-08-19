# Phase 2 — Desktop shell

## Goal

Turn the Phase 1 policy and persistence core into a usable local desktop workspace without prematurely launching unverified browser binaries.

## Technology decision

- Wails v2.12.0: current stable Wails release when this phase was prepared.
- React 19 + TypeScript + Vite.
- Standard CSS with no component-library dependency.
- Existing Go core remains the source of truth.

Wails v3 is intentionally not used because it is still pre-release. The frontend can run in browser-preview mode with temporary demo data, while the packaged desktop application binds directly to Go methods.

## Delivered interactions

- Dashboard with profile readiness and route metrics.
- Searchable profile registry with groups and tags.
- Create and edit profile drawer.
- Clone and delete actions.
- Capability-driven fingerprint fields.
- Kernel registry explaining verified and unavailable controls.
- Deterministic launch-plan preview.
- Read-only application and security settings.

## Security boundaries

Phase 2 still does **not**:

- start Chromium processes;
- download browser kernels;
- execute Xray, sing-box or Mihomo;
- persist proxy passwords;
- expose a new network listener;
- import cookies or extensions.

The UI produces configuration and launch plans only. Process execution remains blocked until signed kernel manifests, OS credential storage and runtime supervision are implemented.

## Local development

```bash
# Install stable Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0

# Run desktop development mode
wails dev

# Frontend only
cd frontend
npm install
npm run dev
```

The browser-only frontend falls back to in-memory sample data. It never writes those samples to the real profile store.

## CI

The stacked Phase 2 PR runs:

- Go formatting, vet, race tests and builds;
- React type checking, tests and production build;
- Windows Go tests;
- full Wails Windows build;
- full Wails Linux build using WebKitGTK 4.1.

The first successful CI run also produces `frontend/package-lock.json` as a short-lived artifact so it can be committed in a follow-up change and switch installation from `npm install` to `npm ci`.
