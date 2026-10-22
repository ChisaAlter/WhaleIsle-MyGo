package settings

import (
	"strings"
)

// settingsActionText turns the settings store's internal action names into
// user-facing status lines. The store records protocol events such as
// "settings document opened" and "settings document updated"; rendering those
// verbatim leaks English diagnostics into the Chinese UI.
func settingsActionText(action string) string {
	if action == "" {
		return ""
	}
	lines := strings.Split(action, "\n")
	for i, line := range lines {
		switch {
		case line == "settings document opened":
			lines[i] = providerCopy("设置已打开", "Settings opened")
		case line == "settings document updated":
			lines[i] = providerCopy("设置已更新", "Settings updated")
		case strings.HasPrefix(line, "openSettingsDocument: "):
			lines[i] = providerCopy("无法打开设置文档：", "Could not open settings: ") +
				strings.TrimPrefix(line, "openSettingsDocument: ")
		}
	}
	return strings.Join(lines, "\n")
}
