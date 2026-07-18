# Milestone Template

Status: Draft
Phase: Phase X
Milestone: MX.Y
Owner decision required: Yes/No

## User outcome

Describe the concrete capability or reliability improvement visible at milestone completion.

## Problem

Explain the current limitation and why it belongs in the active phase.

## Scope

- included item;
- included item.

## Non-scope

- explicitly deferred adjacent work;
- unrelated cleanup or expansion.

## Dependencies

List required earlier milestones, providers, contracts, decisions, and platform support.

## Contract and data impact

Document API, schema, persistence, provider, runtime, UI, migration, and compatibility changes.

## Security, privacy, and licensing

Describe secret handling, local/remote boundaries, downloaded artifacts, third-party licenses, abuse risks, and failure behavior.

## Implementation sequence

1. contract and validation changes;
2. portable implementation;
3. platform-specific implementation;
4. UI or API surface;
5. migration and compatibility work;
6. documentation and release notes.

## Acceptance criteria

- [ ] measurable behavior;
- [ ] unsupported states fail closed;
- [ ] required documentation is updated;
- [ ] no unplanned scope is included.

## Evidence plan

| Evidence level | Required checks |
| --- | --- |
| Static | formatting, lint, vet, typecheck |
| Unit | contract and failure-path tests |
| Integration | component boundaries and persistence |
| Runtime | real binary or browser behavior when applicable |
| Platform | explicitly supported operating systems |
| Recovery | migration, rollback, cancellation, and cleanup |

## Rollback and compatibility

Explain how users recover from failure and how old data or providers are handled.

## Files likely involved

- `path/to/module`;
- `path/to/tests`;
- `docs/relevant-document.md`.

## Completion handoff

State exactly what the next milestone may begin after this one is merged.
