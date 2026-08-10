# AgentSpace Master / Worker 计划

## 目标

让用户只与 Master 对话。Master 用 AgentSpace 为 Worker 创建隔离 worktree，Worker 交接可审核的最终提交，Master 审核后才合并到主工作区目标分支。

## 领域用语

- **Master**：与用户对接、派发任务、审核和合并的主代理；不在 Worker worktree 中实现代码。
- **Worker**：只在分配的执行工作区完成一个任务的代理。
- **执行工作区**：`agentspace new` 创建的 git worktree 和 `agentspace/<name>` 分支。
- **Handoff**：Worker 最终提交、变更文件、测试记录和摘要组成的审核对象。
- **主工作区**：目标基准分支已检出的原始 worktree；只有 Master 可向其中合并。

## 第一期：受控单 Worker 闭环

### 技能

- 安装 `agentspace-master`：明确范围与验收条件，创建工作区，派发 Worker，检查 Handoff，执行代码审核，批准、合并和汇报。
- 安装 `agentspace-worker`：只在任务指定 worktree 中改动，遵守项目指令，执行测试并使用 `submit` 交接；禁止 merge、push、remove。

### CLI 与元数据

- 在 `Workspace` 保存 `Handoff` 和 `Review`，并增加 `submitted`、`changes_requested`、`accepted`、`cancelled` 状态。
- 新增 `submit`：提交未提交改动，运行声明的检查，要求检查后工作区干净，并记录最终 `HEAD`、文件和检查结果。
- 新增 `handoff`：供 Master 人工或程序化读取交接内容。
- 新增 `approve` 和 `request-changes`：记录 Master 对精确提交的审核决定。
- `merge` 仅接受已批准的 Handoff；校验主工作区干净、当前分支是该工作区的 `BaseBranch`、Worker HEAD 未变化。
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

## 非目标

- AgentSpace 不负责替 Master 做语义代码审核；CLI 只保存审核决定并强制流程。
- 第一期不自动将一个需求拆成多个 Worker，也不自动 push 或创建 PR。
