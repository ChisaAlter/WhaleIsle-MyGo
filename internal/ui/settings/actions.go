package settings

import (
	"fmt"
	"strings"
)

// settingsActionText turns the settings store's internal action names into
// user-facing status lines. The store records protocol events such as
// "settings document opened" and "settings document updated"; rendering those
// verbatim leaks English diagnostics into the Chinese UI.
func settingsActionText(action string) string {
	if strings.TrimSpace(action) == "" {
		return ""
	}
	lines := strings.Split(action, "\n")
	out := make([]string, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		var mapped string
		switch {
		case line == "settings document opened":
			mapped = providerCopy("设置已打开", "Settings opened")
		case line == "settings document updated":
			mapped = providerCopy("设置已更新", "Settings updated")
		case strings.HasPrefix(line, "openSettingsDocument: "):
			mapped = providerCopy("无法打开设置文档：", "Could not open settings: ") +
				strings.TrimPrefix(line, "openSettingsDocument: ")
		case line == "settings saved":
			mapped = providerCopy("设置已保存", "Settings saved")
		case line == "settings mutated":
			mapped = providerCopy("设置已更新", "Settings updated")
		case line == "settings replaced":
			mapped = providerCopy("设置已替换", "Settings replaced")
		case line == "settings read-only":
			mapped = providerCopy("当前设置处于只读状态", "Settings are read-only")
		case line == "No settings changes":
			mapped = providerCopy("设置无变更", "No settings changes")
		case line == "custom instructions saved":
			mapped = providerCopy("自定义指令已保存", "Custom instructions saved")
		case line == "current session model updated":
			mapped = providerCopy("当前会话模型已更新", "Current session model updated")
		case line == "viewTabs synced":
			mapped = ""
		case strings.HasPrefix(line, "viewTabs sync: "):
			mapped = providerCopy("同步视图标签失败：", "Could not sync view tabs: ") +
				strings.TrimPrefix(line, "viewTabs sync: ")
		case strings.HasPrefix(line, "Skill saved: "):
			mapped = providerCopy("技能已保存：", "Skill saved: ") +
				strings.TrimPrefix(line, "Skill saved: ")
		case strings.HasPrefix(line, "Skill save failed: "):
			mapped = providerCopy("技能保存失败：", "Skill save failed: ") +
				strings.TrimPrefix(line, "Skill save failed: ")
		case strings.HasPrefix(line, "Skill invocation: "):
			rest := strings.TrimPrefix(line, "Skill invocation: ")
			var updated, failed int
			if n, err := fmt.Sscanf(rest, "%d updated, %d failed", &updated, &failed); err == nil && n == 2 {
				mapped = providerCopy(fmt.Sprintf("技能调用设置：已更新 %d 项，失败 %d 项", updated, failed), line)
			} else {
				mapped = providerCopy("技能调用设置：", "Skill invocation: ") + rest
			}
		case line == "Skill directory opened on its Host":
			mapped = providerCopy("已在对应 Host 打开技能目录", "Skill directory opened on its Host")
		case strings.HasPrefix(line, "skill updated: "):
			mapped = providerCopy("技能已更新：", "Skill updated: ") +
				strings.TrimPrefix(line, "skill updated: ")
		case strings.HasPrefix(line, "skill update: "):
			mapped = providerCopy("技能更新失败：", "Skill update failed: ") +
				strings.TrimPrefix(line, "skill update: ")
		case strings.HasPrefix(line, "skill created "):
			mapped = providerCopy("技能已创建：", "Skill created: ") +
				strings.TrimPrefix(line, "skill created ")
		case strings.HasPrefix(line, "skill invocation "):
			mapped = providerCopy("技能调用权限已更新：", "Skill invocation updated: ") +
				strings.TrimPrefix(line, "skill invocation ")
		case strings.HasPrefix(line, "skill hub "):
			mapped = ""
		default:
			mapped = raw
		}
		if mapped != "" {
			out = append(out, mapped)
		}
	}
	return strings.Join(out, "\n")
}
