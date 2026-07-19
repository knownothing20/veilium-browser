# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: 63c05cc7f52233939b94a1d7b88efd79ed6b6c3c
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: Corrective — First Exact Reviewed Provider Path
Current task: Complete PR #34 exact reviewed Chromium installation and protected Evidence exit gates

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. Work only on Issue #30.

Phase 4 is `Active` only for the narrow reviewed-Provider corrective task. Phase 5 planning and implementation remain blocked.

## Closing review result

Issue #28 concluded **Blocked** because M4.1–M4.4 are complete but the Provider catalog contains no exact `reviewed` Provider. Hosted Chrome/Chromium CI proves the Evidence harness and cannot be silently promoted.

## Authorized corrective scope

- one pinned official Chromium Snapshot for Windows x64;
- exact Revision, source URL, archive size/SHA-256, browser version, executable path/size/SHA-256, license/provenance, limitations, and review time;
- fail-closed install/import and verification;
- no moving latest lookup, bundling, or silent update;
- exact M4.2 identity, M4.3 window/consistency, and M4.4 Network Evidence on the same binary;
- exact compatibility output and rollback/failure tests;
- return to Phase 4 Closing Review after merge.

## PR #34 implementation status

- the exact Provider contract, release pin, complete-package Store, installer, desktop binding, installation card, and tamper tests are present;
- frontend typecheck/tests/build and Windows/Linux Wails builds pass;
- the protected Windows package-tree identity and same-binary Evidence chain remain blocked and must pass before merge;
- the PR remains Draft and must not broaden beyond Issue #30.

## Active prohibitions

Do not begin Phase 5, add another Provider/platform, build or patch Chromium, broaden fingerprint controls, expand proxies, add cookies/extensions/migration/API/MCP/sync/release work, or include unrelated refactors.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

## Handoff

After PR #34 merges, return Phase 4 to `Closing` and rerun the closing review. Phase 5 remains blocked.