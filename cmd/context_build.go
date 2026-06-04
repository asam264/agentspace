package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asam264/agentspace/internal/git"
	"github.com/asam264/agentspace/internal/workspace"
)

func buildContext(p workspace.Paths, ws *workspace.Workspace) string {
	absPath := filepath.Join(p.Root, filepath.FromSlash(ws.Path))

	stat, err := git.Run(absPath, "diff", "--stat", ws.BaseCommit)
	if err != nil || stat == "" {
		stat = "(尚无改动)"
	}

	created := ws.CreatedAt
	if t := parseTime(ws.CreatedAt); t != "" {
		created = t
	}

	var b strings.Builder
	fmt.Fprintln(&b, "=== AgentSpace 工作区上下文 ===")
	fmt.Fprintf(&b, "工作区名称: %s\n", ws.Name)
	fmt.Fprintf(&b, "工作目录: %s\n", absPath)
	fmt.Fprintf(&b, "任务描述: %s\n", orDash(ws.Description))
	fmt.Fprintf(&b, "基于分支: %s (commit: %s)\n", ws.BaseBranch, ws.BaseCommit)
	fmt.Fprintf(&b, "创建时间: %s\n", created)
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
	fmt.Fprintln(&b, "你的工作目录是上方的【工作目录】路径，所有修改请在此目录内进行")
	fmt.Fprintln(&b, "不要修改 .git 目录和 .agentspace 目录")
	fmt.Fprintf(&b, "需要保存进度时，执行：agentspace snapshot %s -m \"描述\"\n", ws.Name)
	fmt.Fprintln(&b, "完成任务后告知用户，由用户执行合并")
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
