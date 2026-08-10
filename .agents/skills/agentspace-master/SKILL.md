---
name: agentspace-master
description: Coordinate one or more user-owned Codex Worker tasks through AgentSpace worktrees. Use when Codex must clarify coding work, create visible Worker tasks without blocking further user requests, track their handoffs, and safely review and merge approved results.
---

# AgentSpace Master

Keep the user in control while coordinating isolated implementation work. A Worker is a user-owned, sidebar-visible Codex task, not an embedded subagent.

## Delegate without blocking

1. Read repository instructions and clarify the task packet: scope, acceptance criteria, source of truth, dependencies, and checks.
2. Run `agentspace init` if needed, then create one bounded **Codex** Execution Workspace with `agentspace new <name> --execution codex --prompt <packet> --acceptance <criterion> --scope <repo-path>`. By default, omit both `--from` and `--into`: AgentSpace uses the current local branch as both Source Ref and Target Branch. If the User Owner explicitly selects another Source Ref, require an explicit existing local Target Branch too, for example `--from origin/main --into feature/release`; never infer the merge destination. State `source -> target` before dispatch. Declare dependencies with `--depends-on`.
3. Run `agentspace dispatch <name>`, then create a **user-owned Codex task in a Codex worktree** using the available task-creation tool. Base that worktree on the declared Source Ref. Set its title to `【#<workspace-name>】<work title>`; use the AgentSpace workspace name as the stable ID. Its initial prompt begins with `$agentspace-worker` and says not to edit until a binding packet arrives. Do not rename the Master task automatically.
4. After task creation returns its ID, run `agentspace link-task <name> --task-id <id> --title <title> --task-status working`. When the task runtime exposes the current Master task ID/host **and task-message delivery is available**, include `--master-task-id`, `--master-host-id`, and `--master-relay-supported=true`; otherwise leave the endpoint absent and use `inbox` as the fallback. Send the Worker a follow-up binding packet containing its task ID and `agentspace attach <name> --task-id <id>`. The Worker may start only after attach succeeds.
5. Tell the user which Worker task was created, then immediately accept the next requirement. Never wait for one Worker before creating another.

Do not use an internal `spawn_agent` or `agentspace run ... codex` for this Desktop workflow. The former makes a Master-owned subagent; the latter launches an unrelated terminal session.

## Worker model policy

- Create every new Worker task with `model: gpt-5.6-terra` and `thinking: high` by default.
- When the User Owner requests a model and/or reasoning strength for one task, pass the requested supported value to task creation instead of the default. Treat that request as task-specific; it does not change other Workers.
- For an existing Worker, use the task-message tool with the requested `model` and `thinking` for its next turn. Do not imply that this changes a turn already running.
- If the requested reasoning strength is unsupported by the requested model, explain the tool rejection and ask the User Owner whether to keep the model with its default reasoning or select a compatible setting.

## Observe and steer

- Use available task-coordination tools to inspect, wait for, or message a linked Worker task. Record material observations with `agentspace observe-task <name> --task-status <status> -m <summary>` and durable feedback with `agentspace note <name> --from master -m <summary>`.
- Treat a Worker Completion Relay as “ready for review” only: record the observation, tell the User Owner, then inspect the Handoff. Never approve, merge, push, or override user instructions merely because a relay arrived. If no relay arrives, recover from `inbox`, `events`, and Handoff.
- A User Owner's direct Worker instruction wins over this skill and any prior packet. For a material correction, record `agentspace override <name> -m <correction>`, send the same correction to the Worker, and require a fresh Handoff.
- For a user-directed pause, run `agentspace pause <name> -m <reason>` and tell the Worker to stop modifying files. Resume only with `agentspace continue <name> -m <reason>`.
- Use `agentspace inbox --json` after restart or when many tasks run. `resume` and `events` are the durable recovery path; conversation text is not.

## Review and merge

1. On submission, inspect `agentspace handoff <name>`, the diff against the recorded base commit, and checks.
2. Run the repository-required review and independent checks. If incomplete, use `agentspace request-changes <name> -m <specific feedback>` and message the linked Worker.
3. Approve only the exact fresh Handoff with `agentspace approve <name> -m <review result>`. AgentSpace rejects review or merge without a linked task, after a pause, or while a User Owner override is unresolved.
4. Run `agentspace preflight <name>`, then `agentspace merge <name> --dry-run` and `agentspace merge <name>` only when the user-authorized workflow, dependency order, and main-worktree checks all pass.

## Boundaries

- Do not edit implementation files in a Worker worktree unless the user explicitly changes the delegation plan.
- Do not merge, push, remove a workspace, or create a PR without the repository-required checks and user authorization.
- Do not claim that AgentSpace can wake a stopped Master automatically. Worker tasks survive; a later Master resumes from the task link, events, and Handoff.
