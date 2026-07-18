# Veilium Product Charter

## Purpose

Veilium is a local-first, multi-profile privacy browser workspace for isolated browser identities, controlled network routing, testing, QA, account separation, and authorized automation.

The product must make browser capabilities explicit and reviewable. A setting is not considered supported merely because it appears in the UI; support requires a provider/version contract and evidence from the selected browser runtime.

## Target users

- individuals managing separate browser identities on one device;
- developers and QA teams testing browser, network, and account-isolation behavior;
- operators who need stable per-profile kernels, data directories, and proxy routes;
- automation developers who need a local, permission-controlled browser launch surface.

## Product principles

1. **Identity consistency over maximum randomness.** A profile should describe one plausible, stable environment.
2. **Evidence over claims.** Public capability claims require integration evidence against a real selected kernel.
3. **Local-first and least privilege.** Sensitive data and control surfaces stay local by default.
4. **Fail closed.** Unsupported, ambiguous, or unverifiable combinations must be rejected rather than silently weakened.
5. **Replaceable providers.** Browser kernels and proxy runtimes are versioned providers, not hidden assumptions.
6. **Portable profile lifecycle.** Long-term product value includes safe backup, migration, and schema evolution.
7. **Controlled automation.** Automation must use explicit authorization, bounded sessions, cancellation, and local auditability.
8. **Clean-room implementation.** Reference projects may inform requirements and architecture, but Veilium code is implemented independently.

## Differentiation goal

Veilium aims to combine:

- the product completeness expected from a modern multi-profile browser workspace;
- the local operations, proxy, migration, and automation discipline of mature local tools;
- understandable fingerprint configuration;
- Veilium's own stronger capability contracts, runtime verification, secret isolation, and reviewable security boundaries.

It is not sufficient to have the largest number of settings or proxy protocols. Veilium should be better because supported behavior is coherent, testable, recoverable, and maintainable.

## Non-goals

Veilium is not intended to:

- bypass platform rules, anti-abuse systems, law enforcement, or access controls;
- promise account survival, CAPTCHA bypass, or detection-evasion rates;
- provide an unauthenticated remote-control service;
- silently download, update, or replace sensitive runtimes;
- copy source code from Donut Browser, Ant Browser, VirtualBrowser, or their browser kernels;
- expand protocol or feature count without a product need, test plan, and maintenance owner.

## Product success criteria

A production-ready Veilium release should demonstrate:

- stable isolated profiles with documented kernel support;
- consistent identity and network behavior verified in real browser sessions;
- safe profile backup, restore, and schema migration;
- controlled API and automation access;
- reproducible cross-platform build and release evidence;
- no plaintext fallback for secrets;
- clear unsupported-state reporting rather than optimistic capability claims.

## Change control

Changes to this charter require a dedicated issue and pull request. The change must explain the user problem, alternatives considered, security and licensing impact, and resulting roadmap changes.

Priority conflicts are resolved in this order:

1. safety, legality, and clean-room constraints;
2. this product charter;
3. the active phase and its exit criteria;
4. current milestone scope;
5. implementation convenience.
