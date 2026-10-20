package usage

import "whaleisle/internal/ui/i18n"

// Pass 4 batch B copy for the usage page header. The usage package is a
// separate UI surface, so its keys live here instead of the settings catalog.
const (
	settingsPass4UsageTitle               = "settings.pass4.usage.title"
	settingsPass4UsageExport              = "settings.pass4.usage.export"
	settingsPass4UsageExportUnavailable   = "settings.pass4.usage.export_unavailable"
	settingsPass4UsageSettings            = "settings.pass4.usage.settings"
	settingsPass4UsageSettingsUnavailable = "settings.pass4.usage.settings_unavailable"
	settingsPass4UsageRefresh             = "settings.pass4.usage.refresh"
	settingsPass4UsageUpdated             = "settings.pass4.usage.updated"
	settingsPass4UsageEmptyTitle          = "settings.pass4.usage.empty_title"
	settingsPass4UsageEmptyBody           = "settings.pass4.usage.empty_body"
)

func init() {
	i18n.Register(settingsPass4UsageTitle, "用量统计", "Usage")
	i18n.Register(settingsPass4UsageExport, "导出", "Export")
	i18n.Register(settingsPass4UsageExportUnavailable, "本版本还没有统计导出。", "Exporting statistics is not wired in this build.")
	i18n.Register(settingsPass4UsageSettings, "设置", "Settings")
	i18n.Register(settingsPass4UsageSettingsUnavailable, "统计范围设置仍由统计服务决定。", "The statistics service still owns range settings.")
	i18n.Register(settingsPass4UsageRefresh, "刷新", "Refresh")
	i18n.Register(settingsPass4UsageUpdated, "数据更新于 %s · UTC", "Updated at %s · UTC")
	i18n.Register(settingsPass4UsageEmptyTitle, "暂无统计数据", "No data yet")
	i18n.Register(settingsPass4UsageEmptyBody, "开始使用 DeepSeek Harness 后，这里会展示 Token 消耗情况", "Token usage will appear here once you start using DeepSeek Harness.")
}
