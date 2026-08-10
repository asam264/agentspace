# Codex 执行工作区绑定与完成通知计划

## 目标

让每个侧栏可见的 Codex Worker 任务在其**实际运行的 Codex worktree**中完成工作，并让 AgentSpace 对该 worktree 的身份、Handoff、审核门禁和完成通知拥有可验证的记录。不能再依靠提示词中的绝对路径约束执行目录。

## 不变量

1. 一个 Worker 任务只绑定一个实际执行工作区；主工作区和其他 Worker worktree 都不能作为它的执行目录。
2. Desktop 默认由 Codex 创建执行 worktree；AgentSpace 只管理协调状态、审核和合并，不再为同一个 Desktop Worker 另建竞争性的 worktree。
3. Worker 未完成绑定验证前不得修改代码、不得 `submit`；`submit`、Handoff、审核和合并均以已绑定的工作区和 commit 为准。
4. Worker 的完成回执只通知 Master“可审核/需处理”；批准、合并、push 始终需要既有门禁和用户授权。
5. 回执失败、Master 被归档或平台不支持投递时，Handoff 与事件仍是权威记录；下一次 Master 可用 `inbox` 恢复，不丢失任务。

## 第一期：实际工作区绑定与共享协调状态

### 范围

- 将 AgentSpace 元数据从主工作区私有目录迁移到同一 Git 仓库所有 worktree 可定位的 **Coordination Store**；保留旧项目的迁移/兼容路径。
- 为 Workspace 增加执行模式、实际工作区路径、Git common-dir 身份、绑定时 HEAD、绑定时间和绑定任务 ID。
- Desktop 的 `new` 创建任务清单而非额外执行 worktree；保留现有 AgentSpace worktree/runner 流程作为显式 legacy 模式。
- 新增 `attach <name>`：只能从 Worker 当前 worktree 执行，验证同仓库、存在于 `git worktree list`、不是主工作区，且当前 HEAD 严格等于记录的来源提交，并写入绑定事件。
- 使 `dispatch`、`submit`、`handoff`、`status`、`inbox`、`preflight`、`merge` 从 Coordination Store 和已绑定工作区读取状态；未绑定 Desktop Worker 不可提交、审核或合并。
- Master 技能改为：创建 Codex 的 worktree Worker → 关联任务 → 等待 Worker 在其当前目录 `attach` 成功；Worker 技能把 `attach` 设为第一步和编码前置条件。

### 验收

- 两个 Codex Worker 的实际目录各不相同，且均不能是主工作区。
- 在任一 Worker worktree 执行 `submit` 后，主工作区中的 Master 可读取同一 Handoff；未 attach 或从错误 worktree 提交会失败。
- `pause`、`override`、依赖和 scope 门禁仍对已绑定 Worker 生效。
- 现有显式 runner 流程继续可用，且不会被误标为 Codex Worker。

## 第二期：完成回执、Master 通信与恢复

### 范围

- 在派发时记录可选的 Master Endpoint（任务 ID、host ID 与投递能力），不得把它当作用户授权。
- Worker 在 `submit` 成功且 Handoff 已写入后，使用 Codex 的任务消息能力发送 Completion Relay：任务名、Handoff commit、检查摘要和状态；投递前后均追加事件。
- Master 收到回执后只更新观察状态、提示用户可审核；不得自动 approve、merge、push 或覆盖用户直接指令。
- 投递失败、Master 已归档或消息工具不可用时记录失败事件，并在 Worker 对话、Desktop Activity/完成通知及 `inbox` 中保留恢复路径。
- Master 技能支持一次观察多个 Worker、处理回执和用户纠正；Worker 技能支持阻塞/完成/需澄清三类回执。

### 验收

- Worker 正常完成后，仍存在的 Master 任务收到一条可追踪回执；Master 当前回合结束不妨碍该回执进入任务。
- Master 不存在或无法接收时，用户仍可从 Worker 任务完成状态和下一次 `inbox` 找到可审核 Handoff。
- 连续十个 Worker 回执不会导致自动合并、串错任务或覆盖用户直接指令。

## 实施前验证

- 先用实际 Codex Desktop 创建一个 worktree Worker，验证任务创建接口返回的任务身份、Worker 内可见的 Git common dir，以及向既有 Master 任务投递消息的行为。
- 在验证通过前，不把“自动唤醒 Master”写入成功标准；它是平台能力，不是 CLI 可以自行承诺的行为。
- 实施时新增 ADR，记录“Codex 管执行工作区、AgentSpace 管协调状态”的长期取舍及 legacy runner 的边界。
