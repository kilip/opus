---
name: Agent task
about: A unit of work written to be picked up by an AI coding agent
title: "<area>: <short imperative summary>"
labels: ""
---

## Context
<!-- Why this work exists and the design decisions already made. Keep it short. -->

## Scope
<!-- Files or directories the agent may touch. Anything outside is out of scope. -->
-

## Out of scope
<!-- What the agent must NOT build in this issue. -->
-

## Acceptance criteria
<!-- One verifiable statement per item. Each must be checkable by running a command or request. -->
- [ ] AC1:

## Definition of Done
- [ ] DoD1: Branch is not `main` (`git branch --show-current`)
- [ ] DoD2: `go test -race ./...` passes
- [ ] DoD3: `mise run lint` passes
- [ ] DoD4: Commits pass commitlint and are atomic per scope
- [ ] DoD5: PR description contains `Closes` followed by this issue number

## Dependencies
Blocked by: none
Epic: #

## Branch and commits
Suggested branch: `<type>/<slug>` | commit type and scope: `<type>(<scope>)`

> **Rules for the agent**
> 1. Tick a box `[x]` only after you verified that item (run the command or request). Do not tick from reading code alone.
> 2. If you cannot verify an item, leave it `[ ]` and comment the reason on this issue.
> 3. Do not edit or remove acceptance criteria. If one is ambiguous, ask in a comment instead of guessing.
> 4. Stay inside **Scope**. Follow `AGENTS.md` (branch gate, atomic commits, feature boundaries).
> 5. Put short evidence (command output) in the PR description.
