# agentspace

`agentspace` 是一个用于管理 AI agent 并行工作区的 Go CLI 工具。它以 **git worktree** 作为底层隔离机制，让多个 agent 同时在同一项目的不同工作区里工作，互不干扰，最后再合并回主分支。

## 特性

- 每个工作区是一个独立的 git worktree + 分支（`agentspace/<name>`），物理隔离
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
| `approve <name> -m <msg>` | Master 记录审核通过，允许合并 |
| `request-changes <name> -m <msg>` | Master 退回 Worker 修改 |
| `merge <name>` | 将已审核通过的工作区合并回主分支 |
| `resume` | 列出未完成工作区及 Master 的下一步动作 |
| `preflight <name>` | 检查依赖顺序和文件范围重叠 |
| `dispatch <name>` | 写入 Worker 上下文；可选启动外部 runner |
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
agentspace init                 # base_branch 默认取当前分支
agentspace init --branch main   # 显式指定基准分支
```

### 创建并查看工作区

```bash
agentspace new feat-auth --desc "实现用户认证模块"
agentspace new fix-payment --from develop --desc "修复支付 bug"

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
agentspace diff feat-auth                   # 与主分支对比（默认）
agentspace diff feat-auth --vs develop      # 与某个分支/commit 对比
agentspace diff feat-auth --vs fix-payment  # 与另一个工作区对比
```

### 合并

```bash
# Worker 完成时，提交最终 commit 与测试记录，交给 Master 审核
agentspace submit feat-auth -m "完成登录模块" --test "go test ./..."
agentspace handoff feat-auth

# Master 完成代码审核后，显式批准该交接
agentspace approve feat-auth -m "代码审核及测试均通过"

# 先检测冲突，不会切换或修改主工作区
agentspace merge feat-auth --dry-run

# 实际合并（仅 accepted 状态；squash 为一个提交）
agentspace merge feat-auth

# 若有冲突：解决后继续，或中止
agentspace merge --continue feat-auth
agentspace merge --abort feat-auth
```

合并前会验证主工作区没有未提交修改、当前分支等于该工作区的基准分支，并验证 Worker 的 `HEAD` 仍是已审核的提交。冲突时会提示是否用编辑器打开冲突文件。编辑器取自 `config.json` 的 `editor` 字段；为空时按 `code → goland → idea → vim → vi` 顺序自动探测。

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

### 恢复、并行与外部 runner

```bash
# Master 重启后恢复队列；--json 适合自动化读取
agentspace resume
agentspace resume --json

# 检查依赖必须先 merged；范围重叠只告警，由 Master 决定如何拆分或排序
agentspace preflight api-auth

# Desktop 模式：仅生成上下文，再由 Master 派 Codex Worker
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

- `base_branch`：新工作区默认基于此分支
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
