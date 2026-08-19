# Veilium Roadmap

Last updated: 2026-08-19
Current development plan: docs/PRISM_COMPARISON_OPTIMIZATION_PLAN.md

## Consolidated baseline

The former Phase 1–5 work is now treated as one completed baseline on `main`: core contracts, Wails desktop, kernel/adapter management, proxy runtime, Credential Vault, real-browser and Network Evidence, lifecycle/recovery, portable Profiles/templates, bounded multi-Profile operations, storage tools, and the Chinese browser workspace.

Old phase documents are archived and no longer control implementation.

## Priority order

| Priority | Work | Status | Outcome |
| --- | --- | --- | --- |
| P0 | Post-merge regression validation | Current | Verify the just-consolidated desktop and real Chromium flow; repair regressions before deeper kernel work |
| P1 | Multi-Provider reviewed kernel foundation | Next | Remove the single-reviewed-Provider assumption without changing current stock Chromium behavior |
| P1 | Veilium Fingerprint Chromium V1 | Planned | Add a separate exact Windows amd64 fingerprint Provider with a minimal verified capability set |
| P1 | Evidence V2 and fingerprint matrix | Planned | Add device/GPU observations, restart stability, seed separation, tamper downgrade, and freshness |
| P2 | Identity Templates and Root Seed | Planned | Replace arbitrary field combinations with a small set of coherent device identities plus advanced mode |
| P2 | Network Identity Advisor | Planned | Turn proxy observations into reviewable language/timezone/geolocation recommendations |
| P2 | Environment Readiness | Planned | Combine lifecycle, kernel, Provider, credential, proxy, Evidence, and freshness into one backend truth |
| P3 | Cookie/Extension product surfaces | Deferred | Add only after the fingerprint core is stable |
| P3 | Controlled automation, MCP, signing, updater, SBOM | Deferred | Separate product/release work after the core browser identity loop is complete |

## Development rule

Work follows the table in dependency order. Internal implementation choices do not need owner approval. The owner is needed only for:

- final merge decisions when requested;
- a major product-direction change;
- irreversible data/security changes;
- project licensing, binary redistribution, branding, or commercial/open-source boundaries.

Do not create new phase/handoff documents for each row. Update this roadmap, `STATUS.md`, and the active plan.
