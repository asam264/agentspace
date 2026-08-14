# Handoff 审核工作区与 GoLand 快捷审核计划

## 目标

在 Worker 提交 Handoff 后，让 User Owner 不必寻找 Codex Worker 的实际目录、切换主工作区分支或等待合并，即可从 Master 给出的单条命令打开 GoLand 审核该 Handoff 的固定快照：

```bash
agentspace review <workspace-name> --open
```

该审核只决定 `approve` 或 `request-changes`；不得自动合并、push 或修改 Worker 的执行目录。

## 术语与不变量

- **Review Workspace** 是一个 detached Git worktree，`HEAD` 严格等于某一次 `Handoff.Commit`。
- 它不是 Codex Worker 的 **Execution Workspace**，也不是主工作区；对其进行编辑不得影响 Worker、Handoff 或主分支。
- 一个新的 Handoff 必须产生或选择一个新的、按 commit 区分的 Review Workspace；旧审核快照不能被静默复用为新 Handoff。
- `approve` 与 `request-changes` 仍只接受当前 Handoff 的精确 commit；打开审核目录本身不改变工作流状态。
- 合并预检的短生命周期目录与 Review Workspace 必须分开存放和清理。

## 用户操作

1. Worker 执行 `submit`，写入 Handoff，并通过 Completion Relay 告知 Master“可审核”。
2. Master 检查 Handoff 后，直接向 User Owner 输出完整命令，例如：

   ```bash
   agentspace review api-auth --open
   ```

3. User Owner 复制执行。命令创建或复用该 Handoff 的审核快照，并打开 GoLand。
4. User Owner 在 GoLand 中查看 Git Log、提交详情、来源提交到 `HEAD` 的 Diff 和代码；然后只需向 Master 给出“通过”或具体修改意见。
5. Master 根据用户结论执行 `approve` + 后续 `merge`，或 `request-changes` 并把反馈发送给原 Worker。

## 第一期：审核快照命令

### 范围

- 新增 `agentspace review <name>`，要求工作区已有 Handoff。
- 在 `.agentspace/reviews/<workspace-name>-<handoff-short-commit>/` 创建 detached Review Workspace，来源严格使用 `Handoff.Commit`。
- 同一名称与 commit 已存在且仍是已注册 worktree 时复用；路径存在但 commit、Git common-dir 或 worktree 注册不匹配时拒绝，而不是覆盖。
- 输出审核目录、Handoff commit、Source Ref/Base Commit 和可用于终端复核的比较命令。
- 持久化 Review Workspace 的路径、commit 和创建时间，供状态、清理与诊断使用。
- `agentspace status` 与 `agentspace handoff` 显示当前 Handoff 的审核快照是否已创建。
- `remove` 与已合并工作区的 `clean` 仅清理属于该工作区的已登记审核快照；删除前验证路径位于 `.agentspace/reviews/` 且仍是对应 Git worktree。

### 验收

- `review` 不切换或修改 Main Worktree，也不接触 Worker 的 Execution Workspace。
- 审核目录的 `HEAD` 等于 Handoff commit；`git diff <base>...HEAD` 只展示该 Handoff 的结果。
- Worker 提交新 Handoff 后，旧审核目录保持指向旧 commit，新命令创建新目录。
- 没有 Handoff、非本仓库路径、已损坏 worktree 或不匹配 commit 时命令明确失败。

## 第二期：`--open` 与 Master 交互

### 范围

- 为 `agentspace review <name>` 增加 `--open`：在快照创建/校验成功后调用已配置或可发现的 GoLand launcher 并传入审核目录。
- Windows 优先使用 `AGENTSPACE_REVIEW_EDITOR`，再探测 `goland64.exe`、`goland.bat` 或 `goland`；未找到时只输出目录和一次性配置说明，不创建其他 IDE 配置。
- 新窗口由 GoLand 的 **Open project in = New Window** 用户设置决定；AgentSpace 不使用未文档化的命令行参数承诺强制新窗口。
- Master 技能在 Completion Relay 或 `inbox` 显示 `submitted` 时，输出可直接复制的 `agentspace review <name> --open`，再等待 User Owner 的审核结论。
- Worker 技能保持不变：Worker 只提交 Handoff，不创建审核目录，也不自行 approve/merge。

### 验收

- User Owner 只需复制 Master 输出的一条 `review --open` 命令即可进入 GoLand 审核。
- launcher 不可用时，Handoff 和审核目录仍可用，错误信息给出目录与配置方式。
- `--open` 失败不改变 Handoff、Review、Worker 状态或 merge 权限。

## 第三期：merge dry-run 临时目录卫生

### 问题

当前 `merge --dry-run` 在 `.agentspace/merge-dryrun-<timestamp>` 创建 detached worktree，并在 deferred cleanup 中忽略删除错误。意外中断、文件锁或删除失败会留下目录，用户无法区分它与长期审核目录。

### 范围

- 将 dry-run 目录统一放入 `.agentspace/tmp/merge-dryrun-<timestamp>/`；Review Workspace 只放入 `.agentspace/reviews/`。
- cleanup 失败时保留诊断事件和明确的目录提示，不再静默忽略。
- 新增显式临时目录清理入口：先列出候选项；实际删除必须由用户显式确认，并且只处理名称匹配、位于 `.agentspace/tmp/`、带有本仓库 common-dir 标记、且不在 `git worktree list` 中的孤儿临时条目（目录或残留标记）。
- 清理逻辑不得删除 Review Workspace、Execution Workspace、主工作区或任意工作区外路径。

### 验收

- 正常 `merge --dry-run` 结束后不残留临时目录。
- cleanup 失败可见、可恢复，且不会掩盖 dry-run 的冲突结果。
- 临时清理拒绝任何未验证或仍注册的 Git worktree。

## 验证与发布

- 使用临时 Git 仓库覆盖：正常审核快照、重复调用、Handoff 更新、无 Handoff、错误路径、`--open` launcher 成功与失败、merge dry-run 清理及孤儿清理拒绝条件。
- 运行 `gofmt`、`go test ./...`、`go vet ./...`、`go build ./...` 和 `git diff --check`。
- 更新 README、Master 技能和命令帮助；提交或推送前执行双轴代码审核。

## 实施状态

- 第一期、第二期、第三期均已实现；`submit`、Worker 回报和 Master 审核流程都会给出 `agentspace review <name> --open`。
