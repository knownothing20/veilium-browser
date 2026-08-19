# Current Project Status

Last updated: 2026-08-19
Current state: Phase 5 product work is consolidated on main; documentation is simplified; post-merge validation remains
Current plan: docs/PRISM_COMPARISON_OPTIMIZATION_PLAN.md
Current task: Complete post-merge desktop/real-Chromium regression validation, repair any regressions, then automatically begin the Multi-Provider reviewed kernel foundation

## What is now on main

- lifecycle journal, locks, cancellation, inventory, snapshot/restore, archive, recoverable trash, and rollback;
- exact reviewed stock Chromium package identity and Provider Contract v2;
- OS Credential Vault, authenticated proxy bridges, Xray/sing-box supervised subsets, and proxy diagnostics;
- browser identity, managed-window/consistency, and Network Evidence;
- portable non-secret Profile definitions, import preview, dependency remapping, templates, and identity-transfer modes;
- bounded multi-Profile metadata/lifecycle/export/health/storage operations and redacted operation reports;
- Simplified Chinese task-oriented browser-environment workspace.

## Validation already recorded before consolidation

The former PR #59 recorded successful frontend typecheck/tests/build, Go fmt/vet/unit/race tests, Go build, Windows Wails build, Wails development startup, and governance validation.

## Validation debt carried into main

The following were not completed before the owner explicitly requested consolidation to `main`, so they remain the first task and must not be retroactively claimed as passed:

1. real Chromium start, readiness, stop, and child-process cleanup after the Chinese workspace changes;
2. manual primary-flow review at 1366×768 and 1920×1080;
3. end-to-end smoke tests for kernel import/install, proxy diagnostics, credentials, recovery, portability, templates, batch management, storage, and Evidence;
4. confirmation that management-language presentation does not mutate Profile language, timezone, platform, or fingerprint values;
5. persistence checks after application restart.

## After validation

If no blocking regression remains, continue without another owner checkpoint to:

1. Multi-Provider reviewed kernel foundation;
2. Veilium Fingerprint Chromium V1;
3. Evidence V2 / restart-stability / seed-separation matrix;
4. Identity Templates and Root Seed;
5. Network Identity Advisor;
6. Environment Readiness and UI integration.

## Non-negotiable boundaries

- existing stock Chromium remains a separate Provider and is never silently upgraded to a fingerprint Provider;
- unsupported capabilities remain unsupported/unverified until real-browser evidence exists;
- secrets remain in the OS vault;
- Chromium Sandbox, exact package integrity, process ownership, rollback, and lifecycle recovery are not weakened to make tests pass;
- no proxy rotation, account farming, telemetry, cloud control, or public remote-control surface is introduced by the current plan.
