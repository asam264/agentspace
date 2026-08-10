# AgentSpace Master / Worker 计划

## 目标

让用户用 Master 协调多个任务，同时能直接控制每个 Worker。AgentSpace 保存任务、审核和合并状态；Codex 为独立、用户可见的 Worker 任务创建实际执行 worktree。Worker 交接可审核的最终提交，Master 审核后按用户授权合并到主工作区目标分支。

## 领域用语

- **用户所有者**：所有 Worker 任务的拥有者；直接指令、暂停和取消的优先级高于 Master 与 Worker。
- **Master**：与用户对接、非阻塞地派发任务、审核和合并的协调代理；不在 Worker worktree 中实现代码。
- **Worker**：在一个独立 Worker 任务中，只在分配的执行工作区完成一个边界明确的任务的代理。
- **Worker 任务**：由 Master 创建、在 Codex 侧栏可见且由用户拥有的独立任务对话；不是内嵌子代理，Master 结束后仍可继续。
- **执行工作区**：Worker 实际运行的 Git worktree。Desktop 默认由 Codex 创建并显式 attach；外部 runner 可使用 AgentSpace 创建的 legacy worktree。
- **来源 ref**：创建执行工作区时的 Git ref；可为本地分支、远程跟踪分支、tag 或 commit，不等于合并目标。
- **合并目标分支**：批准 Handoff 后在主工作区接收 squash merge 的现有本地分支。
- **任务关联**：执行工作区与 Worker 任务 ID、标题和创建时间的持久化关联。
- **人工接管**：用户直接给 Worker 下达的更正、暂停或取消；它会使原任务包和未重新提交的结果失效，Master 不得据此自动审核或合并。
- **Handoff**：Worker 最终提交、变更文件、测试记录和摘要组成的审核对象。
- **主工作区**：合并目标分支已检出的原始 worktree；Master 只在用户授权的流程下向其中合并。

## 第一期：受控单 Worker 闭环

### 技能

- 安装 `agentspace-master`：明确范围与验收条件，创建工作区，派发 Worker，检查 Handoff，执行代码审核，批准、合并和汇报。
- 安装 `agentspace-worker`：只在任务指定 worktree 中改动，遵守项目指令，执行测试并使用 `submit` 交接；禁止 merge、push、remove。

### CLI 与元数据

- 在 `Workspace` 保存 `Handoff` 和 `Review`，并增加 `submitted`、`changes_requested`、`accepted`、`cancelled` 状态。
- 新增 `submit`：提交未提交改动，运行声明的检查，要求检查后工作区干净，并记录最终 `HEAD`、文件和检查结果。
- 新增 `handoff`：供 Master 人工或程序化读取交接内容。
- 新增 `approve` 和 `request-changes`：记录 Master 对精确提交的审核决定。
- `merge` 仅接受已批准的 Handoff；校验主工作区干净、当前分支是该工作区的合并目标分支、Worker HEAD 未变化。
- `merge --dry-run` 改在临时 detached worktree 中执行，不切换主工作区分支。

### 成功标准

1. Worker 的未提交改动不能绕过 Handoff 进入合并。
2. Worker 在提交后继续改动时，必须重新提交和审核。
3. Master 在主工作区有未提交改动、分支不匹配或未审核时不能合并。
4. 任务可从 `submitted` 被退回到 `changes_requested` 后再次提交。

## 第二期：恢复、并行与运行器适配（已实现）

- `resume`：Master 重启后扫描未终态工作区并输出下一步；支持 JSON。
- 任务清单：`TaskManifest` 持久化验收条件、仓库相对文件/目录范围、依赖任务和派发时间；不持久化短生命周期的 Codex 子代理 ID。
- 多 Worker：`preflight` 将未 merged 依赖作为阻断条件；对打开工作区的声明范围做重叠告警。`merge` 复用依赖门禁，保证合并顺序。
- 可选运行器适配：`dispatch --runner <command>` 先写入 `AGENT_CONTEXT.md`，再在执行工作区启动显式命令；无 runner 时为 Codex Desktop Master 准备上下文。
- 可观测性：Workspace 追加 `Event` 历史，记录创建、派发、提交、审核、状态转换和 runner 结果；`events --json` 与已有 `list --json` 可导出状态。

## 第三期：用户拥有的多 Worker 任务（控制面已实现）

### 设计决定

- Worker 使用独立、用户可见的 Codex 任务对话，而不是 Master 的内嵌子代理。每个任务只绑定一个执行工作区。
- Master 创建 Worker 后立即返回可继续接收需求，不能等待某个 Worker 完成才接受下一个任务；因此可同时运行多个 Worker。
- 用户可直接打开、停止和纠正任一 Worker。用户的直接指令优先于 Master；Master 只能在人工接管后的新 Handoff 上审核。
- AgentSpace 保存任务关联和流程事件；聊天消息是协作通道，Handoff、Review 和工作区状态才是审核与合并的持久依据。
- Master 运行时可通过 Codex 的任务协调能力查询、等待和向 Worker 发送补充说明。Master 停止后，Worker 任务及其 worktree 继续存在；恢复后的 Master 从 AgentSpace 的任务关联、事件和 Handoff 继续协调。
- 本期不承诺“Master 已停止时自动被唤醒并处理完成通知”。这需要 Codex Desktop 的自动化/唤醒能力，作为下一期独立验证，不能伪装成 CLI 已具备的能力。

### 任务 3.1：持久化任务关联与人工接管

- `Workspace` 保存 Worker 任务 ID、标题、创建时间和最近一次已知关联状态；任务 ID 不作为进程 ID 或子代理 ID 使用。
- `link-task <workspace> --task-id <id> --title <title>` 供 Master 在创建 Codex 任务后立即写入关联；`observe-task` 记录已读取的任务状态，不伪造平台 API。
- `pause`、`continue`、`override`、`cancel` 都追加 Workflow Event 并保留原因。`override` 使已有 Handoff/Review 失效，必须重新提交后才可审核。
- 为过渡兼容保留现有 `dispatch` 的 runner 行为；无 runner 的 Desktop 流程改为生成 Worker 任务包和待关联状态，不能宣称已经启动 Worker。
- `new` 分离 `--from`（来源 ref）与 `--into`（合并目标本地分支）；默认均为创建时当前本地分支，`--from origin/main --into main` 支持显式发布分支任务。

**验收标准**：重启后能从 `list --json`、`status` 和 `events` 找回每个工作区对应的 Worker 任务及人工接管原因；暂停或人工接管后的旧 Handoff 不能被批准或合并。

### 任务 3.2：Master / Worker 技能与可见任务启动

- 两个技能作为仓库内 `.agents/skills/` 可分发 Codex 技能维护，避免只在某一台机器上存在。
- Master 技能按顺序执行：明确任务包与 scope → `new` / `dispatch` → 创建用户拥有的 Codex Worker 任务 → `link-task` → 立即回到可接收新需求的状态。
- Worker 任务首行显式使用 `$agentspace-worker`，并带入任务名、验收条件、范围、依赖和反馈协议；实际执行目录由后续 `attach` 验证，不依赖提示词路径。
- Worker 技能规定：收到用户直接更正时先记录人工接管，再以新的范围继续；被暂停时停止修改并汇报当前状态；不得自行合并或 push。

**验收标准**：同一 Master 对话可连续创建多个在侧栏可见的 Worker 任务；每个 Worker 的任务关联、人工接管和 Handoff 均可独立恢复。

### 任务 3.3：状态收件箱与审核门禁

- `agentspace inbox --json` 按“待关联、运行中、已暂停、待审核、需要修改、阻塞、可合并”列出工作区及下一步。
- 将任务关联、人工接管、暂停/继续和 Worker 反馈写入事件；`resume` 的下一步动作以这些状态为准。
- 审核与合并前检查：必须已关联 Worker 任务、未暂停、无未处理人工接管、Handoff 与当前确认范围一致、依赖已合并。
- Master 通过 Codex 任务协调能力读取或等待已关联 Worker 的状态，并以事件/状态同步作为恢复依据；不把短暂的聊天上下文当作唯一状态源。

**验收标准**：十个并行任务中，Master 可一次看出哪些需要用户注意、哪些可审核；用户暂停或纠正其中一个任务后，其余任务不受影响，且该任务无法绕过门禁合并。

### 任务 3.4：端到端验证与文档

- 使用临时 Git 仓库端到端验证多 Worker 创建、任务关联、并行 scope 告警、暂停、人工接管、重新提交、审核、依赖合并和恢复。
- 更新 README，区分“独立用户拥有的 Worker 任务”“内嵌子代理”“外部 runner”三种运行形式与适用边界。
- 为“独立用户拥有的任务而非内嵌子代理”的长期架构取舍新增 ADR，说明用户控制权、并发需求和不能自动唤醒的限制。

**验收标准**：文档能让用户按命令与技能创建、观察、纠正、审核多个 Worker；`go test ./...`、`go vet ./...` 和端到端流程均通过。

## 第四期：Codex 执行工作区绑定与完成回执（已实现）

详细实施与验收见 [Codex 执行工作区绑定与完成通知计划](codex-worktree-integration-plan.md)。

- Desktop `new --execution codex` 只创建任务清单；Codex 创建实际 worktree 后，Worker 必须在当前目录执行 `attach`，验证同仓库、已注册 worktree、非主工作区、来源提交和关联任务 ID。
- attach 后的绝对路径、Git common dir、绑定 HEAD、绑定时间与 Worker 任务 ID 持久化；未 attach 的 Codex Worker 不能 submit、review 或 merge。
- `submit`、review、diff、snapshot、restore 与 merge 都使用已绑定实际 worktree；merge 使用被审核 Handoff commit，不依赖一条竞争性的 AgentSpace 分支。
- 可选 Master Endpoint 允许 Worker 在 submit 后使用 Codex 任务消息能力投递 Completion Relay，并用 `relay` 持久化投递结果。回执只提示待审核；失败时由 Worker 状态、Handoff、events 与 inbox 恢复。

## 非目标

- AgentSpace 不负责替 Master 做语义代码审核；CLI 只保存审核决定并强制流程。
- 第一期不自动将一个需求拆成多个 Worker，也不自动 push 或创建 PR。
- 第三期不把用户拥有的 Worker 任务降级为 Master 的内嵌子代理，也不因 Master 对话结束而取消 Worker。
