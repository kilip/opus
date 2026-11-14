---
name: builder
description: Use when executing an approved implementation plan via parallel subagents with a specific model (defaulting to 9router/ag/gemini-3.8-flash-low)
tags: [implementation, subagents, plan, execution]
---

# Builder — Plan Execution via Model-Pinned Subagents

Execute an approved implementation plan by partitioning work into disjoint file-ownership slices and spawning parallel `task` subagents pinned to a specific model (`9router/ag/gemini-3.8-flash-low` by default).

## Preconditions & Gate

1. **Wait for Explicit Approval ("gas")**:
   - NEVER start implementation or spawn coding subagents during planning/interviewing until the user explicitly gives the go-ahead (`"gas"`).
2. **Git Branch Gate**:
   - Check `git branch --show-current`.
   - If on `main`, create or switch to a feature branch (`git switch -c <type>/<slug>`) before spawning subagents.
3. **Shared Prerequisites First**:
   - Run any shared `go.mod` / `package.json` dependency installation once in the parent session before spawning parallel subagents so workers never race on lockfiles.
4. **Disjoint File Ownership**:
   - Partition the plan into self-contained slices where each subagent owns an explicit, non-overlapping list of files (`# Target`).
   - Instruct every subagent to skip `mise run api`, `go test`, `mise run lint`, and formatters mid-flight; run verification once in the parent session after all subagents finish.

## Procedure

### 1. Spawn Subagents via `eval` (`tool.task`)

The direct `task` tool JSON schema wraps `tasks[].model` as an `ARRAY`, which the runtime rejects when given an array and falls back to the parent session model when omitted.

Always spawn model-pinned subagents via `eval` (`language: "js"`) using `await tool.task(...)` so `model` is passed as an exact string:

- **Default model**: `"9router/ag/gemini-3.8-flash-low"` (unless the user explicitly requests another approved `provider/model-id` string).

```js
await tool.task({
  i: "Spawn parallel builder subagents",
  context: [
    "# Goal",
    "Implement the approved plan in local://plan.md.",
    "Skip running build, codegen, lint, or test commands mid-flight; parent runs verification once at the end.",
    "",
    "# Shared Contracts",
    "<exact shared types, signatures, and routes>"
  ].join("\n"),
  tasks: [
    {
      name: "SliceOne",
      model: "9router/ag/gemini-3.8-flash-low",
      solutionSpace: "exact file specs in local://plan.md",
      task: [
        "# Target",
        "Own ONLY: <explicit file list>",
        "",
        "# Change",
        "<step-by-step instructions>",
        "",
        "# Acceptance",
        "All target files created/updated cleanly without running build/test commands."
      ].join("\n")
    }
  ]
});
```

### 2. Wait for Completion

- Immediately leave `eval` and call the `wait` tool until all spawned subagents complete.
- Never poll with `sleep` or `ps`.

### 3. Verify Model & Integrate

1. If model verification is needed, inspect line 3 (`"type":"model_change"`) of the subagent transcript (`history://<Name>` or the session `.jsonl`) to confirm `"model":"9router/ag/gemini-3.8-flash-low"`.
2. Run code generation and verification once in the parent session:
   - `mise run api` (when backend handlers/DTOs changed)
   - `go test ./...`
   - `mise run lint`
   - `bunx biome check .`
3. Fix any integration mismatches and report results concisely.

## Notes / Failure Modes

- **`Task N has an invalid model value [...]`**: Caused by calling the direct `task` tool with `model: ["..."]`. Use `eval` (`js`) `await tool.task(...)` with a string `model: "9router/ag/gemini-3.8-flash-low"`.
- **Subagent used parent model (`gemini-4-argon-medium`)**: Caused by omitting `model` in `tasks[]`. Always set `model: "9router/ag/gemini-3.8-flash-low"` explicitly on every `tasks[]` item inside `eval`.
- ** unavailable model**: If the requested model fails or is unavailable, stop immediately and report to the user; NEVER substitute another model silently.
