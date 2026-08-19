#!/usr/bin/env python3
"""Validate Veilium's single current development source of truth."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
REQUIRED_FILES = (
    "README.md",
    "AGENTS.md",
    "docs/DEVELOPMENT.md",
    ".github/PULL_REQUEST_TEMPLATE.md",
    ".github/ISSUE_TEMPLATE/work_item.md",
)
PRODUCT_CODE_PREFIXES = ("cmd/", "internal/", "frontend/src/")
PRODUCT_CODE_FILES = {
    "desktop_app.go",
    "desktop_main.go",
    "desktop_proxy_diagnostics.go",
    "desktop_app_local_recovery.go",
    "desktop_app_multi_profile.go",
    "desktop_app_portable.go",
    "desktop_app_storage_locations.go",
    "go.mod",
    "go.sum",
    "Makefile",
    "wails.json",
    "frontend/package.json",
    "frontend/package-lock.json",
}


def read_text(path: str, errors: list[str]) -> str:
    target = ROOT / path
    if not target.is_file():
        errors.append(f"required file is missing: {path}")
        return ""
    return target.read_text(encoding="utf-8")


def field(text: str, label: str, source: str, errors: list[str]) -> str:
    match = re.search(rf"^{re.escape(label)}:\s*(.+?)\s*$", text, re.MULTILINE)
    if not match:
        errors.append(f"{source} is missing metadata field: {label}")
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
        errors.append(f"unable to inspect changed files against {base}: {exc}")
        return set()
    return {line.strip() for line in result.stdout.splitlines() if line.strip()}


def is_product_code(path: str) -> bool:
    return path in PRODUCT_CODE_FILES or path.startswith(PRODUCT_CODE_PREFIXES)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", help="Optional base ref used for PR change rules.")
    args = parser.parse_args()
    errors: list[str] = []

    for path in REQUIRED_FILES:
        read_text(path, errors)

    development = read_text("docs/DEVELOPMENT.md", errors)
    current_state = field(development, "Current state", "docs/DEVELOPMENT.md", errors)
    current_task = field(development, "Current task", "docs/DEVELOPMENT.md", errors)

    if not current_state:
        errors.append("Current state must not be empty")
    if not current_task:
        errors.append("Current task must not be empty")

    if args.base:
        changed = changed_files(args.base, errors)
        product_changes = sorted(path for path in changed if is_product_code(path))
        if product_changes and "docs/DEVELOPMENT.md" not in changed:
            errors.append(
                "product-code changes require docs/DEVELOPMENT.md in the same pull request"
            )

    if errors:
        print("Project governance check failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Project governance check passed: {current_state}; task={current_task}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
