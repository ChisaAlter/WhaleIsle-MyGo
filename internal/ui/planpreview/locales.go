package planpreview

import (
	"strings"

	"whaleisle/internal/ui/i18n"
)

// PlanLocalesZH and PlanLocalesEN synchronize @deepseek-ai/dsh-client-ui-plan locales.
var PlanLocalesZH = map[string]string{
	"chip.active":                "计划模式",
	"chip.exit":                  "退出计划模式",
	"chip.exiting":               "正在退出计划模式…",
	"chip.exitFailed":            "退出计划模式失败",
	"chip.shortcut":              "Shift+Tab 切换计划模式",
	"preview.title":              "计划预览",
	"preview.document":           "计划 · Markdown",
	"preview.open":               "在侧边栏打开计划",
	"preview.full":               "查看全文",
	"preview.openNamed":          "打开计划：{title}",
	"preview.close":              "关闭计划预览",
	"preview.loading":            "正在读取计划…",
	"preview.empty":              "当前没有可预览的计划",
	"preview.failed":             "无法读取计划",
	"preview.invalidAddress":     "计划地址无效",
	"preview.historyUnavailable": "无法读取会话历史",
	"preview.unavailable":        "计划预览不可用",
}

var PlanLocalesEN = map[string]string{
	"chip.active":                "Plan mode",
	"chip.exit":                  "Exit plan mode",
	"chip.exiting":               "Exiting plan mode…",
	"chip.exitFailed":            "Failed to exit plan mode",
	"chip.shortcut":              "Shift+Tab toggles plan mode",
	"preview.title":              "Plan preview",
	"preview.document":           "Plan · Markdown",
	"preview.open":               "Open plan in sidebar",
	"preview.full":               "View full plan",
	"preview.openNamed":          "Open plan: {title}",
	"preview.close":              "Close plan preview",
	"preview.loading":            "Loading plan…",
	"preview.empty":              "No plan selected",
	"preview.failed":             "Could not load plan",
	"preview.invalidAddress":     "Invalid plan address",
	"preview.historyUnavailable": "Session history is unavailable",
	"preview.unavailable":        "Plan preview is unavailable",
}

// LocaleText returns the localized string for a dsh-client-ui-plan key.
func LocaleText(key string) string {
	if i18n.Locale() == i18n.LocaleEN {
		if v, ok := PlanLocalesEN[key]; ok {
			return v
		}
	}
	if v, ok := PlanLocalesZH[key]; ok {
		return v
	}
	return key
}

// FormatOpenNamed formats preview.openNamed with the given plan title.
func FormatOpenNamed(title string) string {
	if strings.TrimSpace(title) == "" {
		return LocaleText("preview.open")
	}
	return strings.ReplaceAll(LocaleText("preview.openNamed"), "{title}", title)
}

// StatusMessage resolves a plan preview status/error code to its localized message.
func StatusMessage(status string) string {
	switch status {
	case "loading":
		return LocaleText("preview.loading")
	case "failed":
		return LocaleText("preview.failed")
	case "invalidAddress", "invalid-address":
		return LocaleText("preview.invalidAddress")
	case "historyUnavailable", "history-unavailable":
		return LocaleText("preview.historyUnavailable")
	case "unavailable":
		return LocaleText("preview.unavailable")
	case "exitFailed", "exit-failed":
		return LocaleText("chip.exitFailed")
	default:
		return LocaleText("preview.empty")
	}
}
