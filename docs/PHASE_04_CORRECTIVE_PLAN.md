# Phase 4 Corrective Plan

Issue: #30
Blocked review: #28

## Decision

Create one reviewed Provider from one pinned official Windows x64 Chromium Snapshot.

## Required identity

Freeze one Revision, official archive URL, archive size and SHA-256, executable path, browser version, executable size and SHA-256, Provider revision, source, license/provenance, review time, and limitations.

Do not resolve `LAST_CHANGE` during normal use. Do not use a moving latest build, bundle the archive, silently update it, or generalize support to another revision or platform.

## Evidence

The same exact binary must complete M4.2 identity Evidence, M4.3 window and consistency checks, and M4.4 Network Evidence in protected Windows CI.

Reviewed trust confirms only the exact source and binary. Unsupported stock Chromium fingerprint overrides remain unsupported.

## Exit

After Issue #30 merges, return Phase 4 to Closing and rerun the exit review. Phase 5 remains blocked until Phase 4 is Done.