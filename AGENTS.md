# Veilium development rules

Before changing the repository, read `docs/DEVELOPMENT.md`. It is the **single current source of truth** for product direction, architecture boundaries, current status, priorities, validation debt, and the exact next task.

Do not create competing current `STATUS`, `ROADMAP`, `PHASE`, `PLAN`, `ARCHITECTURE`, `PRODUCT`, handoff, or closing-review documents. Git history, merged PRs, issues, and commits are the archive.

## Execution

- Development is autonomous by default. Do not ask whether to continue for ordinary implementation choices, build/test fixes, local refactors, UI details, patch rebases, or a single optional capability downgrade.
- Prefer the smallest safe, reversible, evidence-backed implementation.
- Product-code changes must update `docs/DEVELOPMENT.md` in the same change set with the real validation result, remaining limitations, and next task.
- `main` is the only long-lived branch. Keep at most one active feature branch for the current major line when practical; merge it, then delete it.
- Do not create activation, handoff, closing-review, process-only, temporary-autofix, or diagnostic branches/PRs.

## Evidence and capability claims

- Reference projects may inform requirements and architecture, not copied implementation.
- A UI field or launch argument is not proof that a browser capability is applied.
- New fingerprint behavior requires an explicit Provider/version capability contract and real-browser evidence on the exact selected binary.
- Unsupported, ambiguous, tampered, or unverifiable combinations fail closed or remain explicitly limited.

## Security and data

- Secrets stay in the operating-system vault; persistent Profile data stores references only.
- Never log proxy passwords, cookies, tokens, decrypted browser data, or private runtime configurations.
- Local APIs/control surfaces remain loopback-only unless separately approved.
- Do not weaken Chromium Sandbox, runtime ownership, package-integrity checks, rollback, recovery, or Evidence to make tests pass.
- Persisted-data changes require compatibility, migration, failure, and rollback analysis.

## Owner gates

Stop for owner input only when the next action would materially change product direction, irreversibly affect user data without an established recovery path, broaden secret exposure/public remote control/telemetry/cloud dependence, weaken a security or Evidence boundary, or require a license/binary-redistribution/branding/commercial decision.
