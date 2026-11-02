package settings

import "github.com/egoist/mygo/ui"

// UsagePreferences paints the two Host-owned rows the installed 用量与费用 page
// keeps under its 设置 button. They moved there from Interface in 0.3.13; the
// rows read and write the same ui-conversation fields as before.
func UsagePreferences(c *ui.Context, host *BodySettingsAccess, fg, muted ui.Color) {
	bodyHostPreference(c, host, "ui-conversation", "sessionCost", providerCopy("会话累计费用", "Session cost"), providerCopy("在输入框下方显示峰谷状态条与当前会话累计费用。", "Show the peak/valley strip and this session's accumulated cost below the composer."), fg, muted)
	bodyHostPreference(c, host, "ui-conversation", "officialPeakValley", providerCopy("官方峰谷时", "Official peak and valley hours"), providerCopy("会话累计费用开启时，在会话统计下方显示官方峰谷时段状态条。开启后始终显示；关闭时仅在检测到 DeepSeek API 路线时显示。", "With session cost on, show the official peak/valley strip below the statistics. On always shows it; off shows it only on a detected DeepSeek API route."), fg, muted)
}
