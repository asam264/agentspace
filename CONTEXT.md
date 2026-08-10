# AgentSpace Workflow

AgentSpace manages isolated code changes that an AI agent performs before a separate agent reviews and merges them.

## Language

**Master**:
The user-facing agent that owns task clarification, delegation, review, and merging.
_Avoid_: main workspace, worker

**Worker**:
An agent assigned to implement one bounded task only in its execution worktree.
_Avoid_: master, reviewer

**Execution Workspace**:
One AgentSpace-managed git worktree and its `agentspace/<name>` branch.
_Avoid_: main worktree, task folder

**Handoff**:
The Worker result identified by one final Git commit plus its summary, changed files, and checks.
_Avoid_: snapshot, informal completion message

**Review**:
The Master's recorded decision on the exact commit named by a Handoff.
_Avoid_: test result, merge

**Main Worktree**:
The original worktree with the target base branch checked out, where approved Handoffs are merged.
_Avoid_: Master, execution workspace

**Task Manifest**:
The durable coordination details of an Execution Workspace: acceptance criteria, File Scope, dependencies, and latest dispatch time.
_Avoid_: agent session, prompt-only task

**File Scope**:
A repository-relative file or directory path declared as owned by an Execution Workspace. It is used for overlap warnings, not as an access-control mechanism.
_Avoid_: glob, changed-file list

**Dependency**:
An Execution Workspace that must be merged before the dependent workspace can merge.
_Avoid_: related task, suggested order

**Workflow Event**:
An append-only record of a meaningful workspace transition, review outcome, or runner outcome.
_Avoid_: current status, log line
