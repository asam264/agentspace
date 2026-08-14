# agentspace

`agentspace` 是一个用于管理 AI agent 并行工作区的 Go CLI 工具。它以 **git worktree** 作为底层隔离机制，让多个 agent 同时在同一项目的不同工作区里工作，互不干扰，最后再合并回主分支。

## 特性

- Desktop Worker 使用 Codex 实际创建的独立 git worktree；AgentSpace 记录并验证该绑定后才接受 Handoff
- 显式 runner 仍可使用 AgentSpace 自建 worktree + 分支（`agentspace/<name>`）
- 工作区进度以「快照」（snapshot）形式保存，可随时回滚
- 一键生成 agent 上下文文本，或直接拉起 `claude` / `codex`
- squash 合并回主分支，支持冲突检测、编辑器介入、continue / abort
- 元数据存于 `.agentspace/`，写入加文件锁，防止并发损坏

## 安装

需要 Go 1.25+ 与 git。

```bash
# 编译
make build          # 产物：./agentspace（Windows 为 agentspace.exe）

# 安装到 /usr/local/bin（可用 PREFIX 覆盖）
sudo make install
make install PREFIX=$HOME/.local
```

或直接用 go：

```bash
go build -o agentspace .
```

## 目录结构

`agentspace init` 会在项目根目录创建：

```
your-project/
├── .agentspace/
│   ├── config.json          # 项目配置
│   ├── workspaces.json      # 所有工作区元数据
│   ├── logs/                # 操作日志
│   └── workspaces/          # worktree 实际目录（本地 Git exclude，不改 .gitignore）
│       ├── feat-auth/
│       └── fix-payment/
```

## 命令速查

| 命令 | 说明 |
| --- | --- |
| `init` | 初始化 agentspace |
| `new <name>` | 创建工作区（worktree + 分支） |
| `list` | 列出所有工作区 |
| `status <name>` | 查看工作区详情与改动 |
| `snapshot <name> -m <msg>` | 保存快照 |
| `snapshots <name>` | 列出快照 |
| `restore <name> <snap-id>` | 回滚到指定快照 |
| `diff <name>` | 查看 diff |
| `submit <name> -m <msg>` | Worker 提交最终交接，等待 Master 审核 |
| `handoff <name>` | 查看 Worker 的最终提交、文件和测试记录 |
| `review <name> [--open]` | 创建固定 Handoff 的审核 worktree；可选打开 GoLand |
| `approve <name> -m <msg>` | Master 记录审核通过，允许合并 |
| `request-changes <name> -m <msg>` | Master 退回 Worker 修改 |
| `merge <name>` | 将已审核通过的工作区合并回主分支 |
| `resume` | 列出未完成工作区及 Master 的下一步动作 |
| `preflight <name>` | 检查依赖顺序和文件范围重叠 |
| `dispatch <name>` | 写入 Worker 上下文；可选启动外部 runner |
| `attach <name>` | Worker 从当前 Codex worktree 验证并绑定执行目录 |
| `link-task <name>` | 关联用户拥有的 Codex Worker 任务 |
| `relay <name>` | 记录 Worker 向 Master 投递完成回执的结果 |
| `observe-task <name>` | 记录 Master 观察到的 Worker 任务状态 |
| `pause/continue <name> -m <msg>` | 记录用户暂停或恢复工作区 |
| `override <name> -m <msg>` | 记录用户更正，并使旧 Handoff 失效 |
| `note <name> -m <msg>` | 记录 Worker、Master 或用户的重要反馈 |
| `inbox` | 按下一步动作聚合多个工作区 |
| `events <name>` | 查看工作区的追加式流程事件 |
| `cancel/fail <name> -m <msg>` | 记录取消或失败原因 |
| `remove <name>` | 删除工作区 |
| `clean` | 批量清理已合并工作区 |
| `context <name>` | 输出 agent 上下文文本 |
| `run <name> [agent]` | 在工作区内拉起 agent |

## 使用示例

### 初始化

```bash
cd your-project
agentspace init                 # 记录当前分支；new 仍以创建时当前分支为默认
agentspace init --branch main   # 记录初始分支（兼容已有配置）
```

### 创建并查看工作区

```bash
agentspace new feat-auth --desc "实现用户认证模块"
# 默认从当前本地分支创建，并合并回该分支；不会切换主工作区分支
agentspace new fix-payment --from origin/main --into main --desc "修复发布分支 bug"

# 第二期：持久化验收条件、文件范围和依赖。范围必须是仓库相对文件或目录，不支持 glob。
agentspace new api-auth --prompt "实现认证 API" \
  --acceptance "登录接口返回令牌" \
  --scope cmd/ --scope internal/auth/ \
  --depends-on schema-change

agentspace list
agentspace list --json
agentspace status feat-auth
```

### 保存与回滚进度

```bash
# 在工作区内编码后保存快照
agentspace snapshot feat-auth -m "完成登录模块"
agentspace snapshots feat-auth

# 回滚到某个快照（之后的快照会被丢弃）
agentspace restore feat-auth snap-001
```

### 对比改动

```bash
agentspace diff feat-auth                   # 与创建时的来源提交对比（默认）
agentspace diff feat-auth --vs develop      # 与某个分支/commit 对比
agentspace diff feat-auth --vs fix-payment  # 与另一个工作区对比
```

### 合并

```bash
# Worker 完成时，提交最终 commit 与测试记录，交给 Master 审核
agentspace submit feat-auth -m "完成登录模块" --test "go test ./..."
agentspace handoff feat-auth

# User Owner 审核固定 Handoff 快照；不会切换主工作区或修改 Worker
agentspace review feat-auth --open

# Master 完成代码审核后，显式批准该交接（第三期必须已关联 Worker 任务）
agentspace approve feat-auth -m "代码审核及测试均通过"

# 先检测冲突，不会切换或修改主工作区
agentspace merge feat-auth --dry-run

# 实际合并（仅 accepted 状态；squash 为一个提交）
agentspace merge feat-auth

# 若有冲突：解决后继续，或中止
agentspace merge --continue feat-auth
agentspace merge --abort feat-auth
```

创建工作区时，`--from` 是来源 ref，`--into` 是最终合并目标本地分支；两者要么同时省略（均为执行 `agentspace new` 时主工作区的当前本地分支），要么同时显式传入。`--from` 可以是 `origin/main`，但 `--into` 必须是可检出的本地分支。创建 worktree 不会切换主工作区分支。合并前会验证主工作区没有未提交修改、当前分支等于工作区的目标分支，并验证 Worker 的 `HEAD` 仍是已审核的提交。Desktop 工作区还会拒绝未关联 Worker 任务、已暂停或有未处理用户更正的工作区；显式 runner 保持原有工作流。冲突时会提示是否用编辑器打开冲突文件。编辑器取自 `config.json` 的 `editor` 字段；为空时按 `code → goland → idea → vim → vi` 顺序自动探测。

### GoLand 审核 Handoff

Worker 提交 Handoff 后，Master 会直接给出 `agentspace review <任务名> --open`。该命令在 `.agentspace/reviews/` 创建或复用一个 detached worktree，`HEAD` 固定为这次 Handoff 的 commit；它不是 Worker 的实际目录，因此 Worker 后续工作或主分支都不会改变你正在审的内容。新 Handoff 会使用新的 commit 目录。

`--open` 会调用 `AGENTSPACE_REVIEW_EDITOR` 指定的 GoLand launcher，或自动探测 PATH 中的 `goland64.exe`、`goland.bat`、`goland`。若没有 launcher，命令仍会输出审核目录，可手动打开。要让每次审核在新 GoLand 窗口打开，在 GoLand 的 **Settings → Appearance & Behavior → System Settings → Open project in** 选择 **New Window**。

`merge --dry-run` 的临时 worktree 位于 `.agentspace/tmp/`，正常完成后会自动删除。若因文件锁等原因遗留，命令会明确提示路径；使用 `agentspace clean --temp` 只会列出并清理带有本仓库 AgentSpace 标记、且未注册的 `merge-dryrun-*` 孤儿临时条目（目录或残留标记），审核目录不会被该命令触碰。

### 清理

```bash
agentspace remove feat-auth     # 删除单个工作区（worktree + 分支 + 元数据）
agentspace clean                # 批量删除所有 status=merged 的工作区
```

### 给 agent 用

```bash
# 打印上下文文本，复制给 agent
agentspace context feat-auth

# 直接在工作区内拉起 agent，并注入上下文
agentspace run feat-auth claude
agentspace run feat-auth codex
agentspace run feat-auth           # 不指定 agent，仅打印上下文与手动启动提示
```

若对应 agent 不在 PATH，或注入失败，会把上下文写入工作区根目录的 `AGENT_CONTEXT.md`，提示你手动让 agent 阅读。

### Codex Desktop：用户拥有的多 Worker 任务

在 Codex 中对 Master 使用 `$agentspace-master`。Master 为每个任务创建独立、侧栏可见的 Worker 任务；它不是内嵌子代理，因此你可以直接打开、暂停或纠正任意 Worker，同时继续向 Master 派发新需求。默认 Worker 使用 `gpt-5.6-terra` 与 `high` 推理强度；对复杂任务可直接告诉 Master“这个 Worker 使用 `<模型>`、`<推理强度>`”，该覆盖只作用于该 Worker。

```bash
# Master 创建任务清单与任务包；实际 worktree 由 Codex Worker 创建
agentspace new api-auth --prompt "实现认证 API" \
  --execution codex \
  --from origin/main --into main \
  --acceptance "登录接口返回令牌" \
  --scope cmd/ --scope internal/auth/
agentspace dispatch api-auth

# Master 创建 Codex worktree Worker 后关联它；标题始终为【#工作区名】工作标题
agentspace link-task api-auth --task-id "task_abc123" --title "【#api-auth】API authentication" --task-status working --master-task-id "master_456" --master-relay-supported=true

# Worker 仅在自己的 Codex worktree 当前目录执行；attach 成功前不得编码
agentspace attach api-auth --task-id "task_abc123"

# Master 重启或同时管理许多 Worker 时查看收件箱
agentspace inbox
agentspace inbox --json

# 用户直接给 Worker 更正、暂停或恢复时，持久化该控制动作
agentspace override api-auth -m "改用现有 token 服务，不新增认证存储"
agentspace pause api-auth -m "先等待接口口径确认"
agentspace continue api-auth -m "口径已确认，按用户最新指令继续"
```

`attach` 会验证 Worker 当前目录是同一 Git 仓库的非主 worktree，且其 HEAD 严格等于声明的来源提交，再把真实路径写入协调状态；未 attach 的 Worker 不能 submit、审核或合并。`override` 会清除旧 Handoff 和 Review，只有新 Handoff 才能批准或合并。`pause` 是 AgentSpace 的审核门禁；如需立即停止模型执行，请同时直接在对应 Worker 任务中停止或发送暂停指令。Worker 完成时以 `submit` 和 Handoff 交接；若记录了支持任务消息的 Master Endpoint，Worker 会先记录投递尝试、发送 Completion Relay，再记录成功或失败结果。回执只提示“可审核”，不会自动合并。Master 对话结束不会结束 Worker；无法投递回执时，后续 Master 通过 `inbox`、`events`、任务关联和 Handoff 恢复协调。

仓库内的 `$agentspace-master` 与 `$agentspace-worker` 位于 `.agents/skills/`。Codex 会扫描当前仓库中的该目录；若技能列表没有即时刷新，重启 Codex。

### 恢复、并行与外部 runner

```bash
# Master 重启后恢复队列；--json 适合自动化读取
agentspace resume
agentspace resume --json

# 检查依赖必须先 merged；范围重叠只告警，由 Master 决定如何拆分或排序
agentspace preflight api-auth

# Desktop 模式：生成 Worker 任务包；Master 创建独立用户拥有的 Worker 任务后必须 link-task
agentspace dispatch api-auth

# CLI 模式：写入 AGENT_CONTEXT.md 后，在工作区运行显式指定的命令
agentspace dispatch api-auth --runner "codex"

# 失败、取消及审核历史均保存在本地状态，可导出为 JSON
agentspace events api-auth --json
agentspace list --json
agentspace fail api-auth -m "依赖服务不可用"
```

## 配置 config.json

```json
{
  "version": "1",
  "base_branch": "main",
  "created_at": "2024-01-15T10:00:00Z",
  "merge_strategy": "squash",
  "editor": ""
}
```

- `base_branch`：为兼容已有配置保留的初始化分支；新工作区默认使用创建时当前本地分支
- `editor`：合并冲突时使用的编辑器命令，留空则自动探测

## 开发

```bash
make build   # 编译
make test    # 测试
make vet     # go vet
make fmt     # 格式化
make tidy    # go mod tidy
make clean   # 清理产物
```

## 实现说明

- CLI 框架：[cobra](https://github.com/spf13/cobra)
- 彩色输出：[fatih/color](https://github.com/fatih/color)
- git 操作直接调用系统 `git`（`exec.Command`），失败时透传原始 git 错误
- 即使从工作区目录内调用，也能通过 `--git-common-dir` 正确定位主仓库的 `.agentspace`
