---
name: vibe
description: Autonomous Issue-to-PR workflow orchestrator that resolves GitHub issues end-to-end without human intervention. Triggered explicitly with /vibe.
---

# Vibe Orchestrator

Resolve GitHub issue `#<ID>` end-to-end as a thin **Orchestrator** delegating to five subagents via `invoke_subagent` (default `Workspace: "inherit"`; pass `Model` only on explicit user override):

- `scout` (`flash`), `planner` (`flash`), `coder` (`flash`), `verifier` (`flash_lite`), `pr-manager` (`flash_lite`).

## Phase 1: Context & Branch Gate (Orchestrator)

```bash
gh issue view <ID> --json number,title,body,labels,state # require state == "OPEN"
# If body has "Blocked by: #N", verify each blocker state != "OPEN" before proceeding.
gh issue edit <ID> --remove-label "status:todo,status:blocked" --add-label "status:in-progress"
# Map label: type:feat|feature -> feat, type:bug|fix -> fix, type:chore -> chore
git checkout main && git pull --ff-only && git switch -c <prefix>/issue-<ID>-<slug>
```

## Phase 2: Recon (`scout`) & Plan (`planner`)

1. **`scout`** (`Role: "Issue #<ID> Scout"`): Pass `#<ID>`, title, body, and `AC1..ACn` to get `<= 30` lines of `path:start-end` pointers.
2. **`planner`** (`Role: "Issue #<ID> Planner"`): Pass `#<ID>`, title, `AC1..ACn`, Definition of Done, and `scout` pointers (omit raw issue body if redundant).
3. Save the `<= 40` line plan to `<appDataDir>/brain/<conversation-id>/issue-<ID>-plan.md` (`UserFacing: true`, `RequestFeedback: false`).

## Phase 3: Implementation (`coder`)

Invoke `coder` (`Role: "Issue #<ID> Coder"`): Pass the Planner's plan inline + `AC1..ACn` (no need to re-read the plan artifact). Save `conversationId`.

- If blocked after 1 warm retry via `send_message`, set label `status:blocked` (remove `status:in-progress`) and stop.

## Phase 4: Audit (`verifier`)

Invoke `verifier` (`Role: "Issue #<ID> Verifier"`): Pass `#<ID>`, `AC1..ACn`, Definition of Done, and `coder`'s commit hash + verification commands.

- **If `FAIL`**: Send failures to `coder`'s `conversationId` via `send_message` (max 2 repair loops) to fix and `git commit --amend --no-edit`, then re-run `verifier`.

## Phase 5: Publish, CI & Merge (`pr-manager`)

Invoke `pr-manager` (`Role: "Issue #<ID> PR Manager"`): Run `vibe-publish-and-merge` for `#<ID>`, branch `<branch>`, commit `<hash>`, title `<type>(<scope>): <summary>`, and verified `AC1..ACn`.

- **If `CI_FAIL`**: Resume `coder` via `send_message` with failed CI logs (max 1 fix), re-run `verifier`, then resume `pr-manager` via `send_message`.
- **If `MERGED`**:
  1. **Telegram (Optional)**: If `TELEGRAM_BOT_TOKEN` & `TELEGRAM_CHAT_ID` or `../telegram/settings.json` are configured, run `bash <telegram-skill-dir>/scripts/send.sh -p HTML -m "<summary>"` (Issue `#<ID>`, PR URL, `main` SHA, verified `AC1..ACn`). Skip silently if unconfigured.
  2. Report PR URL, `main` commit SHA, and closed `#<ID>`.
