# Veilium development rules

## Required reading

Before planning or modifying the repository, read in this order:

1. `docs/PRODUCT.md`;
2. `docs/ROADMAP.md`;
3. `docs/STATUS.md`;
4. the current phase document named by `docs/STATUS.md`;
5. `docs/DEVELOPMENT_PROCESS.md`;
6. relevant module documents and code.

Do not begin by independently redesigning the project from code or README alone.

## Source-of-truth order

When instructions conflict, follow this order:

1. safety, law, licensing, and clean-room restrictions;
2. `docs/PRODUCT.md`;
3. `docs/ROADMAP.md`;
4. the active `docs/PHASE_XX.md`;
5. `docs/STATUS.md`;
6. module documents;
7. issue and pull-request scope.

Stop and open a planning change when a requested implementation conflicts with a higher-level source. Do not silently reinterpret the plan.

## Scope and workflow

- Never commit directly to `main`; use reviewed branches and Draft PRs.
- Work only on the single current task or an explicitly approved issue.
- Keep one pull request focused on one reviewable problem.
- Do not add unrelated features, refactors, protocols, dependencies, or UI redesigns while completing another task.
- Do not implement work from a later phase before the current phase exit gate closes.
- Product-code pull requests must update `docs/STATUS.md` with completed work, validation, risks, and the exact next task.
- Phase scope changes require updates to the phase document, roadmap, and status in one planning PR.
- A phase becomes active or closes only through the process defined in `docs/DEVELOPMENT_PROCESS.md`.

## Clean-room and capability evidence

- Do not copy source code from Donut Browser, Ant Browser, VirtualBrowser, or their browser kernels.
- Reference projects may inform product requirements and architectural lessons only.
- New fingerprint fields require a provider/version capability contract and tests.
- Do not claim a setting is applied until an integration test verifies the selected browser binary.
- Unsupported or ambiguous combinations must fail closed and be reported clearly.

## Security and data

- Local APIs must bind to loopback and require authentication by default.
- Never log proxy passwords, cookies, tokens, decrypted browser data, or private runtime configurations.
- Keep secrets in the operating-system vault and pass only references through persistent profile data.
- Do not add workflow write permissions, automatic merging, deployment, remote binding, telemetry, or background downloads without explicit review.
- Keep platform-specific runtime code behind interfaces and test the portable policy layer on Linux and Windows.
- Changes to persisted data require compatibility, migration, failure, and rollback analysis.

## Completion standard

A task is not complete until:

- its acceptance criteria are satisfied;
- relevant tests and failure paths pass;
- security and licensing impact is documented;
- implementation documents describe the merged behavior accurately;
- `docs/STATUS.md` identifies one explicit next task;
- the final diff still matches the issue's scope and non-scope.
