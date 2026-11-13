package main

import "strings"

func settingsModelsSessionAction(lastAction, modelsErr string) string {
	trimmed := strings.TrimSpace(lastAction)
	if trimmed == "" || trimmed == strings.TrimSpace(modelsErr) {
		return ""
	}
	switch strings.ToLower(trimmed) {
	case "current session model updated", "当前会话模型已更新":
		return trimmed
	default:
		return ""
	}
}

func settingsSkillsSessionAction(lastAction string) string {
	trimmed := strings.TrimSpace(lastAction)
	if trimmed == "" {
		return ""
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "skill ") || strings.HasPrefix(lower, "skills/") {
		return trimmed
	}
	if strings.Contains(lower, "add skill") || strings.Contains(lower, "conflicts with an existing skill") ||
		strings.Contains(lower, "no workspace or session available for skill") || strings.Contains(lower, "unknown skill root") {
		return trimmed
	}
	return ""
}

func joinNonEmptyLines(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return strings.Join(out, "\n")
}
