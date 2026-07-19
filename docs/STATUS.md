# Current Project Status

Last updated: 2026-07-20
Application version: 0.15.0-dev
Main baseline SHA: 63c05cc7f52233939b94a1d7b88efd79ed6b6c3c
Current phase: Phase 4
Current phase document: docs/PHASE_04.md
Current milestone: Corrective — First Exact Reviewed Provider Path
Current task: Finalize Draft PR #34 and confirm protected checks on the closing head

## Operational rule

Read this file after `AGENTS.md`, `docs/PRODUCT.md`, and `docs/ROADMAP.md`. Work only on Issue #30.

Phase 4 is `Active` only for the narrow reviewed-Provider corrective task. Phase 5 planning and implementation remain blocked.

## Closing review result

Issue #28 concluded **Blocked** because M4.1–M4.4 were complete but the Provider catalog contained no exact `reviewed` browser Provider. Hosted Chrome/Chromium CI proved the Evidence harness and could not be silently promoted.

## Authorized corrective scope

- one pinned official Chromium Snapshot for Windows x64;
- exact Revision, source URL, archive size/SHA-256, browser version, executable path/size/SHA-256, complete Package Tree, license/provenance, limitations, and review time;
- fail-closed install/import and verification;
- no moving latest lookup, bundling, or silent update;
- exact M4.2 identity, M4.3 window/consistency, and M4.4 Network Evidence on the same binary;
- compatibility output restricted to the exact Provider revision/version/platform/binary combination;
- rollback, missing, modified, incompatible, traversal, link, and dependency-tamper tests;
- return to Phase 4 Closing Review after merge.

## PR #34 implementation status

- the exact reviewed Provider contract and immutable release manifest are present;
- the installer verifies the archive, `chrome.exe`, and all 261 extracted package files before atomic activation;
- reviewed Providers cannot use generic single-file import;
- the desktop Kernel registry exposes an explicit license acknowledgement and Windows amd64 installation action;
- Windows Required CI has installed the fixed package and passed identity/window Evidence, controlled Network Evidence, and dependency-tamper fail-closed verification on the same managed `chrome.exe`;
- frontend typecheck/tests/build, Go quality, official adapter checks, Linux browser/adapter tests, and Windows/Linux Wails builds have passed on the implementation head;
- compatibility validation rejects the reviewed Provider outside its exact embedded revision, browser version, operating system, architecture, and trust boundary;
- Provider, licensing, installation, package-integrity, compatibility, and current-status documentation are included;
- the PR remains Draft until every protected check passes on its final closing head.

## Active prohibitions

Do not begin Phase 5, add another Provider/platform, build or patch Chromium, broaden fingerprint controls, expand proxies, add cookies/extensions/migration/API/MCP/sync/release work, or include unrelated refactors.

## Required validation

```bash
python scripts/check_project_governance.py
make check
```

Protected Windows CI must additionally use the pinned product installer and the same resulting binary for identity, window/consistency, Network Evidence, and package-tamper gates.

## Handoff

After PR #34 merges, return Phase 4 to `Closing` and rerun the closing review. Phase 5 remains blocked.
