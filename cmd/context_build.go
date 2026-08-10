package cmd

import (
	"fmt"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/workspace"
)

func buildContext(p workspace.Paths, ws *workspace.Workspace) string {
	absPath, pathErr := executionPath(p, ws)
	stat := "(等待 Worker attach)"
	if pathErr == nil {
		if output, err := git.Run(absPath, "diff", "--stat", ws.BaseCommit); err == nil && output != "" {
			stat = output
		}
	}

	created := ws.CreatedAt
	if t := parseTime(ws.CreatedAt); t != "" {
		created = t
	}

	var b strings.Builder
	fmt.Fprintln(&b, "=== AgentSpace 工作区上下文 ===")
	fmt.Fprintf(&b, "工作区名称: %s\n", ws.Name)
	if pathErr == nil {
		fmt.Fprintf(&b, "工作目录: %s\n", absPath)
	} else {
		fmt.Fprintln(&b, "工作目录: 由 Codex Worker 创建后，在其当前目录 attach")
	}
	fmt.Fprintf(&b, "任务描述: %s\n", orDash(ws.Description))
	fmt.Fprintf(&b, "来源 ref: %s (commit: %s)\n", ws.BaseBranch, ws.BaseCommit)
	fmt.Fprintf(&b, "合并目标分支: %s\n", targetBranch(ws))
	fmt.Fprintf(&b, "创建时间: %s\n", created)
	if len(ws.Task.AcceptanceCriteria) > 0 {
		fmt.Fprintln(&b, "验收条件:")
		for _, criterion := range ws.Task.AcceptanceCriteria {
			fmt.Fprintf(&b, "- %s\n", criterion)
		}
	}
	if len(ws.Task.FileScope) > 0 {
		fmt.Fprintf(&b, "文件范围: %s\n", strings.Join(ws.Task.FileScope, ", "))
	}
	if len(ws.Task.DependsOn) > 0 {
		fmt.Fprintf(&b, "依赖工作区: %s\n", strings.Join(ws.Task.DependsOn, ", "))
	}
	if ws.WorkerTask != nil {
		fmt.Fprintf(&b, "Worker 任务: %s (%s)\n", ws.WorkerTask.Title, ws.WorkerTask.ID)
	}
	if ws.Master != nil && ws.Master.TaskID != "" {
		fmt.Fprintf(&b, "Master 回执目标: %s（任务消息投递: %t）\n", ws.Master.TaskID, ws.Master.RelaySupported)
	}
	if ws.Status == workspace.StatusPaused {
		fmt.Fprintf(&b, "工作流已暂停: %s\n", ws.Control.PauseReason)
	}
	if hasPendingOverride(ws) {
		fmt.Fprintf(&b, "用户人工接管: %s\n", ws.Control.Override.Reason)
	}
	fmt.Fprintln(&b, "")

	// 任务说明区块（在注意事项之前）
	fmt.Fprintln(&b, "== 任务说明 ==")
	fmt.Fprintln(&b, "")
	switch {
	case ws.Prompt != "":
		fmt.Fprintln(&b, ws.Prompt)
	case ws.Description != "":
		fmt.Fprintln(&b, ws.Description)
	default:
		fmt.Fprintln(&b, "（未设置任务说明）")
	}
	fmt.Fprintln(&b, "")

	fmt.Fprintln(&b, "== 注意事项 ==")
	fmt.Fprintln(&b, "")
	if executionKind(ws) == workspace.ExecutionCodex && !hasAttachedExecution(ws) {
		fmt.Fprintf(&b, "在当前 Codex Worker worktree 先执行：agentspace attach %s --task-id <你的任务ID>。attach 成功前不得修改代码。\n", ws.Name)
	} else {
		fmt.Fprintln(&b, "你的工作目录是上方的【工作目录】路径，所有修改请在此目录内进行")
	}
	fmt.Fprintln(&b, "不要修改 .git 目录和 .agentspace 目录")
	fmt.Fprintf(&b, "用户直接更正任务时，先执行：agentspace override %s -m \"更正说明\"，再按新范围工作\n", ws.Name)
	fmt.Fprintf(&b, "需要向 Master 记录阻塞或进度时，执行：agentspace note %s --from worker -m \"说明\"\n", ws.Name)
	fmt.Fprintf(&b, "需要保存中间进度时，执行：agentspace snapshot %s -m \"描述\"\n", ws.Name)
	fmt.Fprintf(&b, "完成并通过测试后，执行：agentspace submit %s -m \"完成说明\"\n", ws.Name)
	if ws.Master != nil && ws.Master.TaskID != "" && ws.Master.RelaySupported {
		fmt.Fprintf(&b, "submit 成功后，先执行 agentspace relay %s --status attempted -m \"准备通知 Master %s\"；再用 Codex 任务消息工具通知 Master（任务名、Handoff commit、检查摘要）；成功后记录 --status delivered，失败后记录 --status failed。\n", ws.Name, ws.Master.TaskID)
	} else {
		fmt.Fprintln(&b, "submit 成功后，当前没有可用的 Master 任务消息投递端点；执行 agentspace relay <name> --status unavailable -m \"原因\"，并以 Handoff、事件和 inbox 作为恢复路径。")
	}
	fmt.Fprintln(&b, "提交后把交接摘要交给 Master；不要自行 approve、merge、push、remove 工作区或修改 .agentspace")
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "== 当前状态 ==")
	fmt.Fprintln(&b, stat)
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "== 开始工作 ==")
	fmt.Fprintln(&b, "")
	fmt.Fprintln(&b, "请根据以上【任务说明】开始实现，无需等待进一步指令。")
	return b.String()
}

func parseTime(s string) string {
	if len(s) >= 19 && s[10] == 'T' {
		return s[:10] + " " + s[11:19]
	}
	return ""
}
