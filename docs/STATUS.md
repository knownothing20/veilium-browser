# Current Project Status

Last updated: 2026-07-19
Application version: 0.15.0-dev
Main baseline SHA: 7d48e6a409d9a559eae2408c808adfdc6051dd47
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: Corrective — First Exact Reviewed Provider Path
Current task: Implement Issue #30 after activation PR merge

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

## Active prohibitions

Do not begin Phase 5, add another Provider/platform, build or patch Chromium, broaden fingerprint controls, expand proxies, add cookies/extensions/migration/API/MCP/sync/release work, or include unrelated refactors.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

## Handoff

After this activation PR merges, Issue #30 is the only authorized task. After Issue #30 merges, return Phase 4 to `Closing` and rerun the exit review.