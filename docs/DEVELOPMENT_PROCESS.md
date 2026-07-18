# Development Process

## Sources of truth

Project decisions are governed in this order:

1. `docs/PRODUCT.md` — product purpose, principles, and non-goals.
2. `docs/ROADMAP.md` — six-phase sequence and phase status.
3. the active `docs/PHASE_XX.md` — approved scope and exit criteria.
4. `docs/STATUS.md` — current milestone, task, blockers, and handoff.
5. module documents — implementation contracts and operational details.
6. issue and pull-request descriptions — the scope of one change.

A lower-level document may add detail but may not contradict a higher-level document.

## Required reading order

Before planning or changing the repository, every developer or agent must read:

1. `AGENTS.md`;
2. `docs/PRODUCT.md`;
3. `docs/ROADMAP.md`;
4. `docs/STATUS.md`;
5. the current phase document named by `docs/STATUS.md`;
6. only then, the relevant module documents and code.

## Unit of work

One issue represents one reviewable problem. One pull request should normally resolve one issue and one milestone task.

Every work item must state:

- the user or engineering problem;
- the current phase and milestone;
- scope and explicit non-scope;
- affected contracts and files;
- security, privacy, and licensing impact;
- validation and acceptance criteria;
- documentation and status updates.

Unplanned improvements discovered during implementation are recorded as separate issues. They are not added to the current pull request unless they are required for correctness or safety and the pull-request scope is explicitly updated.

## Standard workflow

1. Read the sources of truth.
2. Select the single current task from `docs/STATUS.md`.
3. Create or confirm a scoped issue.
4. Create a branch from the latest default branch using `agent/<description>` or an equivalent phase-prefixed name.
5. Open a Draft PR early when implementation begins.
6. Implement only the approved scope.
7. Add or update tests before claiming completion.
8. Update relevant documentation and `docs/STATUS.md` in the same PR.
9. Run local checks and allow required CI checks to complete.
10. Resolve review comments and confirm the diff still matches the issue.
11. Merge only when acceptance criteria and governance checks pass.
12. Update the handoff so the next session has one explicit next task.

## Scope control

A pull request must not:

- introduce a feature outside the active phase;
- start work from a later phase before the current phase exit gate closes;
- add unrelated cleanup, refactors, dependencies, protocols, or UI redesigns;
- change product goals implicitly through implementation;
- claim provider or fingerprint behavior without the required evidence;
- weaken security boundaries to make implementation easier.

When a requested change conflicts with the active plan, stop implementation and open a planning issue or planning PR. Do not silently reinterpret the roadmap.

## Documentation ownership

- `PRODUCT.md` changes only when product intent changes.
- `ROADMAP.md` changes when phase status, order, or approved phase goals change.
- `PHASE_XX.md` changes when the active phase scope or acceptance criteria change.
- `STATUS.md` changes with every product-code PR and whenever the current task, blocker, version, or handoff changes.
- module documents change with the contracts they describe.
- architecture decisions with long-term consequences should be recorded in a dedicated decision document when needed.

Documentation must describe the merged implementation, not an intended future state presented as already complete.

## Validation levels

Use the strongest applicable evidence:

1. formatting and static checks;
2. unit tests;
3. component and integration tests;
4. real-binary or real-browser runtime tests;
5. cross-platform CI;
6. security and failure-path tests;
7. migration, rollback, and long-running tests when state or release behavior changes.

A UI control, launch argument, or mocked test alone does not prove that a selected browser binary applies a fingerprint setting.

## Phase activation and closure

A planned phase becomes active only through a dedicated planning PR that defines its user outcome, ordered milestones, non-goals, dependencies, evidence plan, and exit criteria.

A phase closes only through a dedicated closure PR that:

- verifies every exit criterion;
- runs the complete required test matrix;
- records unresolved risks and deferred work;
- marks the phase document `Done`;
- updates `ROADMAP.md` and `STATUS.md`;
- identifies the first planning task for the next phase.

## Urgent fixes

A severe correctness or security fix may interrupt the current task when the issue explains why it cannot wait. The fix must remain narrow, include regression evidence, update status, and avoid opportunistic feature work.

## Required repository settings

The repository owner should configure default-branch protection to require:

- pull requests instead of direct pushes;
- required CI and governance checks;
- resolved review conversations;
- no force pushes or branch deletion;
- the same rules for administrators where supported.

These settings are repository configuration and are not enforced solely by files in the repository.
