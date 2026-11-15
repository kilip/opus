# AGY Builder & PR Workflow Plugin

Custom Antigravity plugin providing a token-efficient Issue-to-PR pipeline (`vibe` skill) and Telegram Bot notifications (`telegram` skill at `skills/telegram/SKILL.md`), backed by five isolated subagents:

- **`scout`** (`agents/scout.md`): Fast, read-only codebase reconnaissance (`read_file`, `find_by_name`, `grep_search`, `list_dir`) that locates exact `file:line` pointers and reuse targets (`<= 30` lines, zero code blocks).
- **`planner`** (`agents/planner.md`): Read-only architectural planner (`read_file` only) that turns Scout pointers and Acceptance Criteria into a pointer-based implementation plan (`<= 40` lines, zero code blocks).
- **`coder`** (`agents/coder.md`): Focused implementation and unit testing agent following workspace conventions with atomic conventional commits.
- **`verifier`** (`agents/verifier.md`): Strict acceptance-criteria auditor without file-write tools (`write_file` and `edit_file` excluded); verifies diffs, runs lint/tests, and marks issue checkboxes (`[x]`) via `gh issue edit`.
- **`pr-manager`** (`agents/pr-manager.md`): Git/GitHub PR publisher and integrator (`pr-pull-ff` linear fast-forward & `pr-deps-squash` bot squash-merge) operating in an isolated context window.
