package sidebar

import "whaleisle/internal/store/session"

func sessionStatus(m Model, it session.Item) string {
	switch status := m.PendingStatuses[it.SessionID]; status {
	case "approval", "plan-review", "question":
		return status
	}
	if it.Running {
		return "running"
	}
	if it.CompletionUnread {
		return "completed"
	}
	return ""
}
func statusPrefix(status string) string {
	switch status {
	case "approval", "plan-review", "question":
		return "! " + statusLabel(status) + " · "
	case "running":
		return "● "
	case "completed":
		return "✓ "
	}
	return ""
}
func statusLabel(status string) string {
	switch status {
	case "approval":
		return workspaceText("等待批准", "Waiting for approval")
	case "plan-review":
		return workspaceText("等待计划审阅", "Waiting for plan review")
	case "question":
		return workspaceText("等待回答", "Waiting for an answer")
	case "running":
		return workspaceText("运行中", "Running")
	case "completed":
		return workspaceText("已完成 · 未读", "Completed · Unread")
	}
	return ""
}
