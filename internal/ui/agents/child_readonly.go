package agents

import "whaleisle/internal/store/session"

func childReadOnlyTitle(v session.AgentNavigationView) string {
	if childReadOnlyReason(v) == "" {
		return ""
	}
	if v.Address.Mode == "one-shot" {
		return SubagentLocale("readonly.oneShot.title")
	}
	return SubagentLocale("readonly.title")
}

func childReadOnlyReason(v session.AgentNavigationView) string {
	switch v.Address.Mode {
	case "one-shot":
		return agentText("一次性任务不支持后续消息，可在这里查看完整执行记录。", "One-shot child: history is read-only; continuation is unavailable.")
	case "unknown":
		return agentText("读取子会话后才能确定是否可继续。", "Child mode is unknown; continuation is unavailable.")
	case "continuable":
		if !v.ParentAvailabilityKnown {
			return agentText("正在等待父会话可用性信息；暂不能发送，运行中的子代理仍可停止。", "Waiting for parent availability; sending is disabled. A running child can still be stopped.")
		}
		if !v.ParentAvailable {
			return agentText("父会话当前不在线，重新打开父会话后即可继续发送消息。", "Parent session is unavailable; sending is disabled. A running child can still be stopped.")
		}
	}
	return ""
}
