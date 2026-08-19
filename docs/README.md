# Veilium documentation

The `docs/` root contains only current sources of truth and the active development plan.

## Current documents

- `PRODUCT.md` — product purpose, principles, non-goals, and success criteria.
- `ARCHITECTURE.md` — durable runtime, security, Provider, Evidence, and data boundaries.
- `ROADMAP.md` — prioritized work from the consolidated baseline forward.
- `STATUS.md` — current truth, validation debt, and exact next task.
- `PRISM_COMPARISON_OPTIMIZATION_PLAN.md` — active implementation plan for Multi-Provider and Fingerprint Chromium work.
- `DEVELOPMENT_PROCESS.md` — autonomous execution, branching, review, and documentation rules.

## Archive

`archive/implemented/` contains completed subsystem design/implementation notes that remain useful for maintenance.

`archive/history/` contains old phase plans, closing reviews, and plans that have already been implemented.

Archive files are historical references only. If an archive document conflicts with the current root documents or current code, the current root documents and code win.

## Documentation rule

Do not add new phase-handoff, activation, closing-review, or temporary planning files to the root. Update the active plan, roadmap, or status instead. When a substantial plan is completed, move it to the archive and keep the root small.
