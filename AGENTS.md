# Veilium development rules

## Required reading

Before changing the repository, read only the current sources of truth in this order:

1. `docs/PRODUCT.md` — product purpose and non-goals;
2. `docs/ROADMAP.md` — prioritized development sequence;
3. `docs/STATUS.md` — current task, blockers, and validation debt;
4. the current plan named by `docs/STATUS.md`;
5. `docs/ARCHITECTURE.md` — durable technical boundaries;
6. `docs/DEVELOPMENT_PROCESS.md` — execution and review rules.

Files under `docs/archive/` are historical references only. They must not override the current sources above.

## Source-of-truth order

When instructions conflict, follow this order:

1. safety, law, licensing, and clean-room restrictions;
2. `docs/PRODUCT.md`;
3. `docs/ROADMAP.md`;
4. `docs/STATUS.md`;
5. the active development plan;
6. `docs/ARCHITECTURE.md`;
7. implementation details and historical archive material.

## Default autonomous execution

- Continue through ordinary technical decisions without asking the owner whether to continue.
- Fix build, test, type, runtime, compatibility, and local architecture problems autonomously.
- Prefer the smallest safe, reversible, evidence-backed implementation.
- A single capability failure should normally downgrade that capability to `unsupported` or `unverified`, not stop unrelated work.
- Ask the owner only for a major product-direction change, an irreversible data/security change, or a licensing/distribution decision.

## Branch and pull-request discipline

- `main` is the single canonical branch.
- Use at most one active feature branch for the current major task unless a split is required for reviewability.
- Do not create phase-handoff, activation, closing-review, process-only, temporary-autofix, or diagnostic branches/PRs.
- Use short-lived `agent/<topic>` branches, merge them, then delete the branch.
- Product-code PRs must update `docs/STATUS.md` in the same change set.
- Do not create a new phase document for routine development; update the current roadmap/status/plan instead.

## Clean-room and capability evidence

- Reference projects may inform requirements and architecture, not copied implementation.
- New fingerprint behavior requires an explicit provider/version capability contract and real-browser evidence.
- A UI field or launch argument is not proof that a browser capability is applied.
- Unsupported, ambiguous, tampered, or unverifiable combinations must fail closed or remain explicitly limited.

## Security and data

- Keep secrets in the operating-system vault; persistent profile data stores references only.
- Never log proxy passwords, cookies, tokens, decrypted browser data, or private runtime configurations.
- Local APIs bind to loopback and require authentication by default.
- Do not weaken Chromium Sandbox, runtime ownership, package-integrity checks, rollback, or Evidence to make tests pass.
- Persisted-data changes require compatibility, migration, failure, and rollback analysis.

## Completion standard

A task is complete when relevant tests and real-runtime checks pass, security/licensing boundaries remain intact, `docs/STATUS.md` records the truth, and completed implementation notes are archived rather than left as competing current plans.
