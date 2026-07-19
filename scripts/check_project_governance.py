#!/usr/bin/env python3
"""Validate Veilium's project-governance sources of truth.

The check intentionally validates a small number of durable rules instead of
trying to infer product correctness from file names. It is safe to run locally
without arguments and may receive a base commit in pull-request CI.
"""

from __future__ import annotations

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REQUIRED_FILES = (
    "AGENTS.md",
    "docs/PRODUCT.md",
    "docs/ROADMAP.md",
    "docs/STATUS.md",
    "docs/DEVELOPMENT_PROCESS.md",
    "docs/MILESTONE_TEMPLATE.md",
    ".github/PULL_REQUEST_TEMPLATE.md",
    ".github/ISSUE_TEMPLATE/work_item.md",
)
ALLOWED_PHASES = {f"Phase {number}" for number in range(1, 7)}
ALLOWED_PHASE_STATUSES = {"Planning", "Active", "Closing", "Done", "Planned"}
PRODUCT_CODE_PREFIXES = (
    "cmd/",
    "internal/",
    "frontend/src/",
)
PRODUCT_CODE_FILES = {
    "desktop_app.go",
    "desktop_main.go",
    "desktop_proxy_diagnostics.go",
    "go.mod",
    "go.sum",
    "Makefile",
    "wails.json",
    "frontend/package.json",
    "frontend/package-lock.json",
}


def fail(message: str, errors: list[str]) -> None:
    errors.append(message)


def read_text(relative_path: str, errors: list[str]) -> str:
    path = ROOT / relative_path
    if not path.is_file():
        fail(f"required file is missing: {relative_path}", errors)
        return ""
    return path.read_text(encoding="utf-8")


def metadata(text: str, label: str, source: str, errors: list[str]) -> str:
    match = re.search(rf"^{re.escape(label)}:\s*(.+?)\s*$", text, re.MULTILINE)
    if not match:
        fail(f"{source} is missing metadata field: {label}", errors)
        return ""
    return match.group(1).strip().strip("`")


def changed_files(base: str, errors: list[str]) -> set[str]:
    try:
        result = subprocess.run(
            ["git", "diff", "--name-only", f"{base}...HEAD"],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        fail(f"unable to inspect changed files against {base}: {exc}", errors)
        return set()
    return {line.strip() for line in result.stdout.splitlines() if line.strip()}


def is_product_code(path: str) -> bool:
    if path in PRODUCT_CODE_FILES:
        return True
    return path.startswith(PRODUCT_CODE_PREFIXES)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--base",
        help="Optional base commit/ref used to enforce pull-request change rules.",
    )
    args = parser.parse_args()
    errors: list[str] = []

    for relative_path in REQUIRED_FILES:
        read_text(relative_path, errors)

    roadmap = read_text("docs/ROADMAP.md", errors)
    status = read_text("docs/STATUS.md", errors)

    roadmap_phase = metadata(roadmap, "Current phase", "docs/ROADMAP.md", errors)
    roadmap_phase_doc = metadata(
        roadmap, "Current phase document", "docs/ROADMAP.md", errors
    )
    roadmap_phase_status = metadata(
        roadmap, "Current phase status", "docs/ROADMAP.md", errors
    )
    status_phase = metadata(status, "Current phase", "docs/STATUS.md", errors)
    status_phase_doc = metadata(
        status, "Current phase document", "docs/STATUS.md", errors
    )
    current_milestone = metadata(
        status, "Current milestone", "docs/STATUS.md", errors
    )
    current_task = metadata(status, "Current task", "docs/STATUS.md", errors)

    if roadmap_phase and roadmap_phase not in ALLOWED_PHASES:
        fail(f"unsupported current phase in roadmap: {roadmap_phase}", errors)
    if status_phase and status_phase not in ALLOWED_PHASES:
        fail(f"unsupported current phase in status: {status_phase}", errors)
    if roadmap_phase and status_phase and roadmap_phase != status_phase:
        fail(
            f"phase mismatch: roadmap={roadmap_phase!r}, status={status_phase!r}",
            errors,
        )
    if roadmap_phase_doc and status_phase_doc and roadmap_phase_doc != status_phase_doc:
        fail(
            "current phase document mismatch between ROADMAP.md and STATUS.md",
            errors,
        )
    if roadmap_phase_status and roadmap_phase_status not in ALLOWED_PHASE_STATUSES:
        fail(f"unsupported phase status: {roadmap_phase_status}", errors)
    if not current_milestone:
        fail("docs/STATUS.md must define a non-empty current milestone", errors)
    if not current_task:
        fail("docs/STATUS.md must define a non-empty current task", errors)

    phase_doc = ""
    if status_phase_doc:
        phase_doc = read_text(status_phase_doc, errors)
        phase_value = metadata(phase_doc, "Phase", status_phase_doc, errors)
        phase_status = metadata(phase_doc, "Status", status_phase_doc, errors)
        implementation_allowed = metadata(
            phase_doc, "Product implementation allowed", status_phase_doc, errors
        )
        if phase_value and status_phase and phase_value != status_phase:
            fail(
                f"phase document declares {phase_value!r}, expected {status_phase!r}",
                errors,
            )
        if phase_status and roadmap_phase_status and phase_status != roadmap_phase_status:
            fail(
                "phase status mismatch between ROADMAP.md and current phase document",
                errors,
            )
        if implementation_allowed not in {"Yes", "No"}:
            fail(
                f"{status_phase_doc} must declare Product implementation allowed: Yes or No",
                errors,
            )
        if phase_status == "Active" and implementation_allowed != "Yes":
            fail("an Active phase must allow product implementation", errors)
        if phase_status in {"Planning", "Closing", "Done", "Planned"} and implementation_allowed != "No":
            fail(
                f"a {phase_status} phase must block product implementation",
                errors,
            )

    if roadmap_phase and f"| {roadmap_phase} |" not in roadmap:
        fail(f"roadmap table does not contain the current phase: {roadmap_phase}", errors)

    if args.base:
        changed = changed_files(args.base, errors)
        product_changes = sorted(path for path in changed if is_product_code(path))
        if product_changes and "docs/STATUS.md" not in changed:
            fail(
                "product-code changes require docs/STATUS.md in the same pull request",
                errors,
            )

        implementation_allowed = ""
        if phase_doc:
            implementation_allowed = metadata(
                phase_doc,
                "Product implementation allowed",
                status_phase_doc,
                errors,
            )
        head_ref = os.environ.get("GITHUB_HEAD_REF", "")
        emergency_branch = head_ref.startswith(("hotfix/", "security/"))
        if product_changes and implementation_allowed != "Yes" and not emergency_branch:
            sample = ", ".join(product_changes[:5])
            fail(
                "product-code changes are blocked while the current phase does not "
                f"allow implementation; changed files include: {sample}",
                errors,
            )

    if errors:
        print("Project governance check failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(
        f"Project governance check passed: {status_phase}, "
        f"{current_milestone}, task={current_task}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
