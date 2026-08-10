# Use user-owned Worker tasks instead of Master-owned subagents

AgentSpace binds each execution worktree to a visible, user-owned Codex Worker task. This favors direct user steering, many concurrent tasks, and task survival after a Master conversation ends over the simpler embedded-subagent topology. AgentSpace persists the task link, workflow events, Handoffs, and user controls; it does not claim to wake an absent Master automatically.
