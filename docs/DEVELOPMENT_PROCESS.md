# Veilium Development Process

Last updated: 2026-08-19

## Goal

Keep development fast and reviewable without recreating the old phase/handoff branch explosion.

## One canonical line

- `main` is the only long-lived canonical branch.
- Keep at most one active feature branch for the current major development line whenever practical.
- Use `agent/<topic>` for a feature branch, merge it, then delete it.
- Do not create activation, handoff, closing-review, process-only, temporary-autofix, or diagnostic branches/PRs.
- Do not create a new phase document for routine work.

## Default execution mode

Development is autonomous by default:

1. read the current sources of truth;
2. implement the current task;
3. run static/unit/integration checks;
4. run real-runtime checks where the claim depends on Chromium or the OS;
5. diagnose and repair failures;
6. record truthful limitations;
7. continue to the next dependency-ordered task when acceptance criteria are satisfied.

Do not pause for ordinary implementation choices, naming, file layout, local refactors, test fixes, small Chromium-version adjustments, or a single capability downgrade.

## Owner gates

Stop for owner input only when the next action would:

- materially change product direction;
- irreversibly delete or transform user data without an established recovery path;
- broaden secret exposure, remote control, telemetry, or cloud behavior;
- weaken Sandbox, package integrity, Provider trust, or Evidence;
- require a project license, binary-redistribution, trademark, or commercial-boundary decision.

## Pull requests

- Keep one PR focused on one major development line, not one PR per sub-capability.
- Use dependency-ordered commits inside the PR.
- Split only when a PR becomes materially unsafe to review or roll back.
- Product-code PRs update `docs/STATUS.md` with what changed, what was actually validated, remaining limitations, and the next task.
- Never claim a test or manual flow passed unless it actually ran.

## Validation

Run the narrowest useful checks during iteration, then the full relevant matrix before declaring the task complete. Capability claims that depend on Chromium require the exact selected binary and real-browser Evidence.

A failed optional capability may remain `unsupported`/`unverified` while unrelated validated work continues. A failure that affects Provider integrity, secrets, Sandbox, persisted data, or recovery blocks release.

## Documentation lifecycle

- Root `docs/` contains only current sources and the active development plan.
- Completed subsystem documentation moves to `docs/archive/implemented/`.
- Old phase/plan/review documents move to `docs/archive/history/`.
- Documents fully superseded and no longer useful are deleted; Git history remains the audit trail.
- Archive material is reference-only and never overrides current code or current root docs.
