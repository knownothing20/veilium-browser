# Phase 4 — Planning Gate

Status: Planning
Phase: Phase 4
Owner decision required: Yes
Product implementation allowed: No

## Purpose

This document is the controlled planning surface for the next product phase. It deliberately does not freeze a feature list yet. The product owner and development agent will review the product direction after the governance baseline is merged, then replace the planning placeholders with an ordered, testable implementation plan.

## Why this gate exists

Veilium has completed substantial browser-runtime and proxy foundations. Continuing directly into more implementation without freezing the next user outcome would create a high risk of expanding whichever subsystem was most recently touched rather than building the most important product capability.

No Phase 4 product feature should be implemented until this document is changed from `Planning` to `Active` in a dedicated planning pull request.

## Inputs to the planning discussion

The discussion should consider, without automatically committing to, the remaining topics already identified in the repository:

- verified browser-kernel provider strategy;
- real browser evidence for fingerprint capabilities;
- window, screen, viewport, and identity consistency;
- live browser network-leak validation;
- profile data, extension, cookie, and migration lifecycle;
- proxy-pool operations versus further protocol expansion;
- sequencing of Launch API, unified CDP, MCP, sync, and release hardening.

These are candidate problem areas, not an approved task list.

## Required decisions before activation

The activation pull request must define:

1. **User outcome** — the concrete capability a user receives at the end of Phase 4.
2. **Scope** — the ordered milestones included in the phase.
3. **Non-scope** — adjacent work explicitly deferred.
4. **Provider and dependency policy** — accepted runtimes, licenses, versions, and trust boundaries.
5. **Data-contract impact** — schema additions, compatibility requirements, and migration rules.
6. **Evidence plan** — unit, integration, real-runtime, cross-platform, and security tests.
7. **Exit criteria** — measurable conditions required to close the phase.
8. **Rollback and recovery** — how failed upgrades or incompatible providers are handled.

## Activation checklist

- [ ] Product owner approves the user outcome.
- [ ] Milestones are ordered by dependency.
- [ ] Every milestone has acceptance criteria.
- [ ] Non-goals prevent unrelated expansion.
- [ ] Security and licensing impact is documented.
- [ ] Required tests and supported platforms are explicit.
- [ ] `docs/ROADMAP.md` and `docs/STATUS.md` are updated in the same pull request.
- [ ] Status above is changed from `Planning` to `Active`.

## Development rule while planning

Only governance documentation, urgent narrowly scoped fixes, and work necessary to validate planning assumptions may be merged. Assumption-validation work must be explicitly approved in its issue and must not be presented as a completed product capability.

## Closure rule after activation

When the approved Phase 4 exit criteria are met, create a dedicated phase-closure pull request. That pull request must run the complete validation matrix, summarize unresolved risks, freeze this document as `Done`, update the roadmap, and identify the first planning task for Phase 5.
