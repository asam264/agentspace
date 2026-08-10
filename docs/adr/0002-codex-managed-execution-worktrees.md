# Codex manages Desktop execution worktrees; AgentSpace manages coordination

Desktop Worker tasks execute in the Codex-managed worktree associated with their visible task. AgentSpace creates the task manifest, validates an explicit `attach`, stores Handoffs and workflow controls, and merges only an approved Handoff commit. It does not create a second execution worktree for the same Desktop Worker.

This avoids a prompt-only directory convention and preserves the User Owner's ability to open, stop, and steer each Worker. The tradeoff is that a Worker must complete an attach handshake before coding, and AgentSpace cannot remove Codex-managed worktrees. Explicit terminal runners retain the legacy AgentSpace-managed worktree mode.
