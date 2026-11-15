# AGY Builder & PR Workflow Plugin

Custom Antigravity plugin providing a token-efficient Issue-to-PR pipeline (`builder` skill) backed by four isolated subagents:

- **`planner`** (`agents/planner.md`): Read-only codebase reconnaissance that outputs a pointer-based implementation plan (`<= 40` lines, zero code blocks).
- **`coder`** (`agents/coder.md`): Focused implementation and unit testing agent following `ponytail` and `AGENTS.md` rules with atomic conventional commits.
- **`verifier`** (`agents/verifier.md`): Strict acceptance-criteria auditor without file-write tools (`write_file` and `edit_file` excluded); verifies diffs, runs lint/tests, and marks issue checkboxes (`[x]`) via `gh issue edit`.
- **`pr-manager`** (`agents/pr-manager.md`): Git/GitHub PR publisher and integrator (`pr-pull-ff` linear fast-forward & `pr-deps-squash` bot squash-merge) operating in an isolated context window.
