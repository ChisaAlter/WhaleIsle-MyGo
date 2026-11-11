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
		return SubagentLocale("readonly.oneShot.body")
	case "unknown":
		return SubagentLocale("readonly.unknown.body")
	case "continuable":
		if !v.ParentAvailabilityKnown {
			return agentText("正在等待父会话可用性信息；暂不能发送，运行中的子代理仍可停止。", "Waiting for parent availability; sending is disabled. A running child can still be stopped.")
		}
		if !v.ParentAvailable {
			return SubagentLocale("readonly.body")
		}
	}
	return ""
}
