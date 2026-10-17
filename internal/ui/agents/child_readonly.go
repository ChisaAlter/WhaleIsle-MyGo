package agents

import "whaleisle/internal/store/session"

func childReadOnlyReason(v session.AgentNavigationView) string {
	switch v.Address.Mode {
	case "one-shot":
		return agentText("一次性子代理只供查看历史，不能继续发送消息。", "One-shot child: history is read-only; continuation is unavailable.")
	case "unknown":
		return agentText("此子代理的运行模式未知，暂不支持继续发送消息。", "Child mode is unknown; continuation is unavailable.")
	case "continuable":
		if !v.ParentAvailabilityKnown {
			return agentText("正在等待父会话可用性信息；暂不能发送，运行中的子代理仍可停止。", "Waiting for parent availability; sending is disabled. A running child can still be stopped.")
		}
		if !v.ParentAvailable {
			return agentText("父会话当前不可用，无法继续发送；运行中的子代理仍可停止。", "Parent session is unavailable; sending is disabled. A running child can still be stopped.")
		}
	}
	return ""
}
