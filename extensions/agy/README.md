# AGY Builder & PR Workflow Plugin

Custom Antigravity plugin providing Issue-to-PR workflows (`express` skill and autonomous `vibe` skill via Opus MCP), Telegram Bot notifications (`telegram` skill), Streamable HTTP MCP configuration (`mcp_config.json`), and five isolated subagents:

- **`scout`** (`agents/scout.md`): Fast, read-only codebase reconnaissance (`read_file`, `run_command` for `rg`/`find`/`ls`) that locates exact `file:line` pointers and reuse targets (`<= 30` lines, zero code blocks).
- **`planner`** (`agents/planner.md`): Read-only architectural planner (`read_file` only) that turns Scout pointers and Acceptance Criteria into a pointer-based implementation plan (`<= 40` lines, zero code blocks).
- **`coder`** (`agents/coder.md`): Focused implementation and unit testing agent following workspace conventions with atomic conventional commits.
- **`verifier`** (`agents/verifier.md`): Strict acceptance-criteria auditor without file-write tools (`write_file` and `edit_file` excluded); verifies diffs, runs lint/tests, and marks issue checkboxes (`[x]`) via `gh issue edit`.
- **`pr-manager`** (`agents/pr-manager.md`): Git/GitHub PR publisher and integrator (`vibe-publish-and-merge` via Opus MCP, `express-publish-and-merge`, `pr-pull-ff`, and `pr-deps-squash`) operating in an isolated context window.

## Opus MCP Configuration (`mcp_config.json`)

`extensions/agy/mcp_config.json` configures the default `opus` Streamable HTTP MCP endpoint at `http://localhost:7880/mcp` for local interactive sessions:

- **Tools exposed**: `get_task`, `list_tasks`, `set_status`, `add_activity`, `submit_report`.
- **Dynamic Worker Injection**: When Opus dispatches an automated `/vibe` run to a worker node, the worker dynamically injects the `opus` HTTP `mcpServers` entry into `agy acp` `session/new` and `session/resume` with per-run authentication and scope headers:
  - `Authorization: Bearer <node-credential>`
  - `X-Opus-Run-ID`
  - `X-Opus-Session-ID`
  - `X-Opus-Project-ID`
