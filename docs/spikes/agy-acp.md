# Spike: `agy acp` Protocol & Runtime Behavior

## 1. Binary, Version & Linux Session Storage (AC1)

- **Binary**: `agy` (`/home/toni/.local/bin/agy`, version `1.5.10`). `antigravity` is not installed in `$PATH` on Linux; `agy acp` is the ACP stdio JSON-RPC server (`--stdio` is default, NDJSON over stdin/stdout).
- **Important CLI Flags**:
  - `--dangerously-skip-permissions`: auto-approves all tool permission requests (useful for unattended worker runs).
  - `--process-per-session`: isolates each session into a worker process.
  - `--sandbox`: enables terminal sandbox restrictions.
  - `--gemini-dir` / `--app-data-dir`: overrides `.gemini` and `antigravity-cli` directories.
  - `--login`: interactive Google login for headless/SSH setups.
- **Linux Session Storage**: Under `<HOME>/.gemini/antigravity-cli/` (respects `HOME` and `--gemini-dir`):
  - `conversations/<sessionId>.db` (SQLite conversation store)
  - `brain/<sessionId>/` (artifacts and `.system_generated/logs/`)
  - `annotations/<sessionId>.pbtxt`
  - `conversation_summaries.db`
  - `log/acp-server-<timestamp>-<pid>.log`

## 2. Core JSON-RPC Wire Shapes (AC1 & AC2)

Fixtures are recorded in `internal/worker/provider/agy/testdata/`:

1. **`initialize`** (`initialize.json`):
   - Client sends `protocolVersion: 1`, `clientInfo`, `clientCapabilities`.
   - Server responds with `agentInfo: {"name": "Antigravity ACP Server", "version": "1.5.10"}`, `agentCapabilities` (`loadSession: true`, `sessionCapabilities: {resume, close, load, list, additionalDirectories}`, `_meta.antigravity.systemPrompt.append: true`), and `authMethods` (`oauth-personal`, `oauth-business`, `gemini-api-key`, `agent-platform`).
2. **`session/new`** (`session_new.json`):
   - Params: `{cwd, mcpServers, _meta: {antigravity: {systemPrompt: {append}}}}`.
   - Result: `{sessionId, models, modes, configOptions, _meta: {antigravity: {tier}}}`.
   - Modes: `default`, `accept-edits`, `auto`, `plan`.
3. **`session/prompt` & `session/update`** (`session_prompt_updates.jsonl`):
   - Params: `{sessionId, prompt: [{type: "text", text: "..."}]}`.
   - Server streams `session/update` notifications with `params.update.sessionUpdate` discriminator:
     - `available_commands_update`
     - `usage_update` (`used`, `size`)
     - `session_info_update` (`title`)
     - `tool_call` (`toolCallId`, `name`, `args`, `status: "in_progress"`, `kind`, `title`, `rawInput`)
     - `tool_call_update` (`status: "completed"`, `rawOutput`, `content`)
     - `agent_message_chunk` (`content: {type: "text", text: "..."}`)
   - Final response: `{result: {stopReason: "end_turn", usage: {inputTokens, outputTokens, cachedReadTokens, cachedWriteTokens, thoughtTokens, totalTokens}}}`.
4. **`session/cancel`** (`session_cancel_resume.json`):
   - Notification `{method: "session/cancel", params: {sessionId}}` cancels the active turn; the in-flight `session/prompt` resolves with `{result: {stopReason: "cancelled"}}`.
5. **Permissions & Question Forms** (`permission_and_question.json`):
   - Server-to-client request `session/request_permission` carries `toolCall` and `options` (`allow_once`, `allow_always`, `reject_once`). Client responds with `{outcome: {outcome: "selected", optionId}}`.
   - Agent questions arrive via `tool_call` updates for `ask_question` with `args.questions[]` (`question`, `is_multi_select`, `options`). In non-interactive/print modes, `ask_question` is auto-answered with `skipped=true`.

## 3. System Prompt Injection, MCP & Subagents (AC3)

- **`_meta.antigravity.systemPrompt.append`**: Verified live. Passing `_meta.antigravity.systemPrompt.append` in `session/new` injects per-session instructions into the system prompt (e.g., instructing `ALWAYS end your reply with XYZ_APPEND_OK.` produced `hi XYZ_APPEND_OK`).
- **MCP Servers**:
  - Per-session `mcpServers` array in `session/new` passes session-scoped MCP server definitions.
  - Persistent user/plugin MCP servers live in `mcp_config.json` (supports `url` as well as stdio `command`/`args`/`env`). Plugin-bundled MCP servers are namespaced as `<plugin>_<server>`.
- **Subagents (`invoke_subagent`)**:
  - Custom agents are defined in Markdown (`agents/*.md`) and exposed to `invoke_subagent` (e.g. `scout`, `planner`, `coder`, `verifier`, `pr-manager` when configured in the active workspace/plugin).
  - Agents listing `invoke_subagent` in their `tools` frontmatter automatically receive the roster of available subagents in their system prompt.

## 4. Mid-Turn Crash & `session/resume` Behavior (AC4)

- Verified by sending `SIGKILL` to `agy acp` mid-turn after receiving `agent_message_chunk`, then starting a fresh `agy acp` process and calling `session/resume` with `{sessionId, cwd, mcpServers: []}`:
  - `session/resume` succeeds and restores the session (`sessionId`, `models`, `modes`, `configOptions`) without replaying prior turn history over `session/update`.
  - Subsequent `session/prompt` and `session/cancel` on the resumed `sessionId` work normally.
  - Worker takeaway: store `sessionId` immediately after `session/new` so a crashed worker can resume via `session/resume` on restart.

## 5. Unauthenticated & Quota-Exhausted Error Shapes (AC5)

- **Unauthenticated**:
  - Isolated `HOME` + `--gemini-dir` probe: `agy acp` still completes `initialize` and `session/new` (advertising `authMethods`), while unauthenticated model/API calls fail with `Authentication required. Please visit the URL to log in:` (`unauthenticatedAuthResult`).
  - Note: on Linux, if OS keyring / default credential store is reachable, `agy` may still resolve stored credentials unless isolated; workers should treat `agy acp --login` as the interactive auth flow and surface `Authentication required` as `unauthenticated`.
- **Quota Exhausted**:
  - Detected in `agy` via `RESOURCE_EXHAUSTED` / HTTP `429` / `exhausted your capacity` / `rate limit` (`errors.json`).
  - Low-quota warnings include remaining percentage and RFC 3339 reset timestamp (`[quota] model <model> is low on quota (<pct> remaining, resets at <RFC3339>)`).
  - Quota-exhausted RPC errors return JSON-RPC error `-32000` with `RESOURCE_EXHAUSTED` message (reset time is surfaced in the message text / quota summary proto when provided by the backend, not as a structured field on the JSON-RPC error object).
