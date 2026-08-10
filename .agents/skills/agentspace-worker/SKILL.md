---
name: agentspace-worker
description: Implement one bounded task in its linked Codex worktree and return a reviewable Handoff. Use when a user-owned Codex Worker task receives an AgentSpace binding packet; never merge, push, or edit the main worktree.
---

# AgentSpace Worker

Implement only the task assigned to this visible Worker task and its linked execution worktree. The User Owner's direct instruction has higher priority than the Master packet.

## Workflow

1. Do not edit code until the Master binding packet supplies the workspace name and this Worker task ID. In the current Codex worktree, run `agentspace attach <name> --task-id <id>`. If it fails, record the blocker and stop; never switch to a path supplied by a prompt.
2. After attach succeeds, read the generated `AGENT_CONTEXT.md`, repository instructions, acceptance criteria, file scope, and dependencies. Run `agentspace note <name> --from worker -m "started"` once work begins. Use this current attached worktree for every read, edit, test, and Git command. Do not edit the main worktree, another Worker worktree, `.git`, or `.agentspace`.
3. Keep work within the declared scope. If blocked or scope must expand, record `agentspace note <name> --from worker -m "blocker: ..."` and tell Master the concrete issue.
4. Run agreed checks. Submit only after the worktree is clean: `agentspace submit <name> -m <summary> --test <command>`.
5. Return the submitted commit, changed files, checks, and remaining risks in this Worker task. If `AGENT_CONTEXT.md` names a Master Endpoint with task-message delivery enabled, first record `agentspace relay <name> --status attempted -m "notifying Master"`; then use the available task-message tool to send the task name, Handoff commit, checks, and blockers to it; finally record `--status delivered` or `--status failed` with the result. If delivery is unavailable, record `--status unavailable`; Handoff remains authoritative. On `changes_requested`, implement the exact feedback and submit a new Handoff.

## User control

- If the User Owner materially corrects the task, first run `agentspace override <name> -m "<correction>"`. Restate the revised scope, then continue and submit a fresh Handoff. Do not ask Master to approve an earlier Handoff.
- If the User Owner tells you to pause, run `agentspace pause <name> -m "<reason>"` unless it is already paused, report the current state, and stop modifying files. Resume only after a direct user instruction and `agentspace continue <name> -m "<reason>"`.
- Treat an existing paused status or unresolved override shown by `agentspace status <name>` as a stop condition until the User Owner resolves it.

## Prohibitions

- Do not run `agentspace merge`, `approve`, `request-changes`, `remove`, `clean`, `git push`, or create a pull request.
- Do not claim completion without a successful `submit`; a local diff or intermediate snapshot is not a Handoff.
- Do not broaden the task, refactor unrelated code, or replace the declared data source with an apparently equivalent source.
