---

name: builder
description: Autonomous Issue-to-PR workflow orchestrator. Use when asked to resolve, fix, handle, implement, or work on GitHub issues end-to-end from code to PR (including "garap issue", "kerjakan issue", "kerjakan #N", "selesaikan issue").
---

# Builder Orchestrator

Execute GitHub issues (`#ID`) end-to-end as a thin **Orchestrator** delegating heavy work to four isolated subagents: `planner` (`flash`), `coder` (`flash`), `verifier` (`flash`), and `pr-manager` (`flash`). Pass `Model` (`pro`, `flash`, `flash_lite`, `inherit`) to `invoke_subagent` only when the user explicitly requests a tier override.

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

## Phase 2: Reconnaissance & Plan (`planner`)

Invoke `planner` via `invoke_subagent`:

- `TypeName`: `"planner"`
- `Role`: `"Issue #<ID> Planner"`
- `Workspace`: `"inherit"`
- `Prompt`: Pass issue `#<ID>`, title, body, `AC1..ACn`, and Definition of Done.

Wait for the `<= 40` line pointer-based Implementation Plan.

---

## Phase 3: Implementation & Atomic Commit (`coder`)

Invoke `coder` via `invoke_subagent`:

- `TypeName`: `"coder"`
- `Role`: `"Issue #<ID> Coder"`
- `Workspace`: `"inherit"`
- `Prompt`: Pass the Planner's Implementation Plan and `AC1..ACn`.

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
- `Prompt`: Run Mode `builder-publish-and-merge` for Issue `#<ID>` on branch `<branch>`, commit `<hash>`, with title `<type>(<scope>): <summary>` and verified `AC1..ACn`.

- **If `pr-manager` returns `CI_FAIL`**: Resume `coder` via `send_message` with the failed CI log (max 1 CI auto-fix), re-run `verifier`, then resume `pr-manager` via `send_message` to push and merge.
- **If `MERGED`**: Report final PR URL, merged `main` commit SHA, and closed Issue `#<ID>` to the user.
