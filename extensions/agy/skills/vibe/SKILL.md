---

name: vibe
description: Autonomous Issue-to-PR workflow orchestrator that resolves GitHub issues end-to-end without human intervention. Triggered explicitly with /vibe.
---

# Vibe Orchestrator

Execute GitHub issues (`#ID`) end-to-end as a thin **Orchestrator** delegating heavy work to five isolated subagents: `scout` (`flash`), `planner` (`flash`), `coder` (`flash`), `verifier` (`flash`), and `pr-manager` (`flash`). Pass `Model` (`pro`, `flash`, `flash_lite`, `inherit`) to `invoke_subagent` only when the user explicitly requests a tier override.

---

## Phase 1: Context & Branch Gate (Orchestrator)

1. **Fetch Issue**:

   ```bash
   gh issue view <ID> --json number,title,body,labels,state
   ```

   Stop if `state != "OPEN"`.
2. **Dependency Gate**:
   Parse `Blocked by: #N, #M` (or `None`). For every blocker ID, run `gh issue view <BLOCKER_ID> --json state`. If any blocker is `OPEN`, stop and report to the user.
3. **Claim Issue**:

   ```bash
   gh issue edit <ID> --remove-label "status:todo,status:blocked" --add-label "status:in-progress"
   ```
4. **Branch Gate** (never write code on `main`):
   Map label to prefix (`type:feat`/`type:feature` -> `feat`, `type:bug`/`type:fix` -> `fix`, `type:chore` -> `chore`):

   ```bash
   git checkout main && git pull --ff-only
   git switch -c <prefix>/issue-<ID>-<slug>
   ```

---

## Phase 2: Reconnaissance (`scout`) & Plan (`planner`)

1. **Codebase Recon (`scout`)**:
   Invoke `scout` via `invoke_subagent`:
   - `TypeName`: `"scout"`
   - `Role`: `"Issue #<ID> Scout"`
   - `Workspace`: `"inherit"`
   - `Prompt`: Pass issue `#<ID>`, title, body, and `AC1..ACn` to locate target files, line ranges, and reuse pointers.

2. **Architectural Plan (`planner`)**:
   Once `scout` returns its `<= 30` line pointer report, invoke `planner` via `invoke_subagent`:
   - `TypeName`: `"planner"`
   - `Role`: `"Issue #<ID> Planner"`
   - `Workspace`: `"inherit"`
   - `Prompt`: Pass issue `#<ID>`, title, body, `AC1..ACn`, Definition of Done, and `scout`'s pointer report in-memory.

Wait for the `<= 40` line pointer-based Implementation Plan, then save it as an artifact at `<appDataDir>/brain/<conversation-id>/issue-<ID>-plan.md` using `write_file` with `ArtifactMetadata` (`UserFacing: true`, `RequestFeedback: false`).

---

## Phase 3: Implementation & Atomic Commit (`coder`)

Invoke `coder` via `invoke_subagent`:

- `TypeName`: `"coder"`
- `Role`: `"Issue #<ID> Coder"`
- `Workspace`: `"inherit"`
- `Prompt`: Pass the Planner's Implementation Plan (and artifact path) plus `AC1..ACn`.

Save the returned `conversationId`. If `coder` reports an unresolved blocker after 1 warm retry via `send_message`, revert labels (`--remove-label "status:in-progress" --add-label "status:blocked"`) and stop.

---

## Phase 4: Audit & Issue Checkbox Sync (`verifier`)

Invoke `verifier` via `invoke_subagent`:

- `TypeName`: `"verifier"`
- `Role`: `"Issue #<ID> Verifier"`
- `Workspace`: `"inherit"`
- `Prompt`: Pass issue `#<ID>`, `AC1..ACn`, Definition of Done, and `coder`'s commit hash.

- **If `FAIL`**: Send failure details back to `coder`'s existing `conversationId` via `send_message` (max 2 repair loops) to fix and `git commit --amend --no-edit`, then re-run `verifier`.

---

## Phase 5: Publish, CI Watch & Fast-Forward Merge (`pr-manager`)

Once `verifier` returns `PASS`, invoke `pr-manager` via `invoke_subagent`:

- `TypeName`: `"pr-manager"`
- `Role`: `"Issue #<ID> PR Manager"`
- `Workspace`: `"inherit"`
- `Prompt`: Run Mode `vibe-publish-and-merge` for Issue `#<ID>` on branch `<branch>`, commit `<hash>`, with title `<type>(<scope>): <summary>` and verified `AC1..ACn`.

- **If `pr-manager` returns `CI_FAIL`**: Resume `coder` via `send_message` with the failed CI log (max 1 CI auto-fix), re-run `verifier`, then resume `pr-manager` via `send_message` to push and merge.
- **If `MERGED`**:
  1. **Telegram Summary (Optional)**: Check if the `telegram` skill is configured (`TELEGRAM_BOT_TOKEN` & `TELEGRAM_CHAT_ID` env vars set, or `../telegram/settings.json` exists with non-empty `bot_token` and `chat_id`). If configured, send a concise summary (Issue `#<ID>`, PR URL, merged `main` commit SHA, and verified `AC1..ACn`) via `bash <telegram-skill-dir>/scripts/send.sh` (using `HTML` parse mode, never leaking secrets). If not configured, skip without prompting the user.
  2. Report final PR URL, merged `main` commit SHA, and closed Issue `#<ID>` to the user.
