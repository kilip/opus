#!/usr/bin/env python3
"""Update GitHub Project #12 (opus-mvp) Status for an Opus issue."""

from __future__ import annotations

import argparse
import subprocess
import sys

PROJECT_NUMBER = "12"
PROJECT_OWNER = "kilip"
ISSUE_BASE_URL = "https://github.com/kilip/opus/issues"

STATUS_MAP = {
    "backlog": "status:backlog",
    "status:backlog": "status:backlog",
    "todo": "status:todo",
    "status:todo": "status:todo",
    "in-progress": "status:in-progress",
    "status:in-progress": "status:in-progress",
    "review": "status:review",
    "status:review": "status:review",
    "done": "Done",
    "status:done": "Done",
}


def run_gh(args: list[str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(["gh", *args], capture_output=True, text=True, check=False)


def main() -> int:
    parser = argparse.ArgumentParser(description="Set opus-mvp project status for an issue.")
    parser.add_argument("issue", help="Issue number (e.g. 108 or #108)")
    parser.add_argument(
        "status",
        help="Target status: backlog, todo, in-progress, review, done (with or without status: prefix)",
    )
    args = parser.parse_args()

    issue_num = args.issue.lstrip("#")
    if not issue_num.isdigit():
        print(f"error: invalid issue number {args.issue!r}", file=sys.stderr)
        return 1

    target = STATUS_MAP.get(args.status.strip().lower())
    if target is None:
        print(
            f"error: unknown status {args.status!r}; expected one of: backlog, todo, in-progress, review, done",
            file=sys.stderr,
        )
        return 1

    issue_url = f"{ISSUE_BASE_URL}/{issue_num}"
    edit_args = [
        "project",
        "item-edit",
        PROJECT_NUMBER,
        "--owner",
        PROJECT_OWNER,
        "--url",
        issue_url,
        "--field",
        "Status",
        "--value",
        target,
    ]

    res = run_gh(edit_args)
    if res.returncode != 0:
        # Auto-add issue to Project #12 if not yet linked, then retry edit once.
        add_res = run_gh(["project", "item-add", PROJECT_NUMBER, "--owner", PROJECT_OWNER, "--url", issue_url])
        if add_res.returncode != 0:
            print(add_res.stderr.strip() or res.stderr.strip(), file=sys.stderr)
            return 1
        res = run_gh(edit_args)
        if res.returncode != 0:
            print(res.stderr.strip(), file=sys.stderr)
            return 1

    print(f"#{issue_num} -> {target}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
