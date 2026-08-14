# AgentSpace Workflow

AgentSpace coordinates isolated code changes across user-owned Codex tasks, durable Git worktrees, and a separate review-and-merge flow.

## Language

**User Owner**:
The human who owns every Worker Task and whose direct instructions override Master and Worker instructions.
_Avoid_: observer, approver

**Master**:
The user-facing coordination agent that clarifies work, creates and follows Worker Tasks, reviews Handoffs, and performs the authorized merge.
_Avoid_: main workspace, worker task

**Worker**:
An agent working in one Worker Task and one Execution Workspace on a bounded implementation task.
_Avoid_: master, reviewer

**Worker Task**:
A user-owned, sidebar-visible Codex task in which one Worker works. It survives the Master conversation that created it.
_Avoid_: embedded subagent, runner process

**Execution Workspace**:
The actual Git worktree bound to one Worker Task. In the Desktop flow it is Codex-managed; an AgentSpace-managed worktree remains a legacy runner option.
_Avoid_: main worktree, task folder

**Source Ref**:
The Git ref used to create an Execution Workspace. It can be a local branch, remote-tracking branch, tag, or commit and does not determine where the result merges.
_Avoid_: merge target, base branch

**Target Branch**:
The existing local branch in the Main Worktree that receives an approved Handoff. It is independent of the Source Ref.
_Avoid_: source ref, remote branch

**Task Link**:
The durable association between an Execution Workspace and its Worker Task identity.
_Avoid_: prompt-only relationship, child-agent ID

**Coordination Store**:
The shared durable AgentSpace state for every worktree of one Git repository.
_Avoid_: a per-worktree scratch directory, chat history

**Master Endpoint**:
The durable identity of the Master task that may receive Worker status messages.
_Avoid_: User Owner, merge authority

**Completion Relay**:
An advisory Worker-to-Master message sent only after a Handoff is recorded. It never approves, merges, pushes, or overrides the User Owner.
_Avoid_: automatic merge, durable Handoff

**Manual Override**:
A material User Owner correction that supersedes the current Worker packet and prevents Master from treating its prior outcome as final. Pause is a separate temporary workflow gate.
_Avoid_: suggestion, comment

**Handoff**:
The Worker result identified by one final Git commit plus its summary, changed files, and checks.
_Avoid_: snapshot, informal completion message

**Review**:
The Master's recorded decision on the exact commit named by a Handoff.
_Avoid_: test result, merge

**Review Workspace**:
A detached worktree pinned to one Handoff commit so the User Owner can inspect it before the Master records a Review. It is neither an Execution Workspace nor a temporary merge check.
_Avoid_: Worker worktree, Main Worktree, merge dry-run directory

**Main Worktree**:
The original worktree with an Execution Workspace's Target Branch checked out, where approved Handoffs are merged.
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
