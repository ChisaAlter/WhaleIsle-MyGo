package settings

import (
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// Pass 4 batch B copy for the Agent preset, usage and About pages. It lives in
// this package so the shared catalog literals stay untouched.
const (
	KeyPass4PresetsTitle        = "settings.pass4.presets.title"
	KeyPass4PresetsHint         = "settings.pass4.presets.hint"
	KeyPass4BuiltIn             = "settings.pass4.presets.builtin"
	KeyPass4Custom              = "settings.pass4.presets.custom"
	KeyPass4TaskDefault         = "settings.pass4.presets.task_default"
	KeyPass4BadgeBuiltIn        = "settings.pass4.presets.badge_builtin"
	KeyPass4ModeExplanation     = "settings.pass4.presets.mode_explanation"
	KeyPass4HowToUse            = "settings.pass4.presets.how_to_use"
	KeyPass4CreateWithAgent     = "settings.pass4.presets.create_with_agent"
	KeyPass4UsageTitle          = "settings.pass4.usage.title"
	KeyPass4UsageUpdated        = "settings.pass4.usage.updated"
	KeyPass4UsageExport         = "settings.pass4.usage.export"
	KeyPass4UsageSettings       = "settings.pass4.usage.settings"
	KeyPass4UsageRefresh        = "settings.pass4.usage.refresh"
	KeyPass4UsageEmptyTitle     = "settings.pass4.usage.empty_title"
	KeyPass4UsageEmptyBody      = "settings.pass4.usage.empty_body"
	KeyPass4AboutTitle          = "settings.pass4.about.title"
	KeyPass4AboutHint           = "settings.pass4.about.hint"
	KeyPass4DesktopVersion      = "settings.pass4.about.desktop_version"
	KeyPass4OnlineInstall       = "settings.pass4.about.online_install"
	KeyPass4InstallUnavailable  = "settings.pass4.about.install_unavailable"
	KeyPass4OpenRuntime         = "settings.pass4.about.open_runtime"
	KeyPass4GitHubReleases      = "settings.pass4.about.github_releases"
	KeyPass4OfficialHarness     = "settings.pass4.about.official_harness"
	KeyPass4CredentialStorage   = "settings.pass4.about.credential_storage"
	KeyPass4RuntimeDir          = "settings.pass4.about.runtime_dir"
)

func init() {
	i18n.Register(KeyPass4PresetsTitle, "Agent 预设", "Agent presets")
	i18n.Register(KeyPass4PresetsHint, "选择 Agent 的工具和工作方式。日常任务用「标准模式」，扩展 DSH 的能力用「创造模式」。", "Choose the Agent's tools and working style. Use Standard for everyday work and Cordis to extend DSH.")
	i18n.Register(KeyPass4BuiltIn, "内置", "Built-in")
	i18n.Register(KeyPass4Custom, "自定义", "Custom")
	i18n.Register(KeyPass4TaskDefault, "任务默认", "Task default")
	i18n.Register(KeyPass4BadgeBuiltIn, "内置", "Built-in")
	i18n.Register(KeyPass4ModeExplanation, "模式说明", "Mode notes")
	i18n.Register(KeyPass4HowToUse, "如何使用", "How to use")
	i18n.Register(KeyPass4CreateWithAgent, "+ 让 Agent 帮我创建预设模式", "+ Let the Agent create a preset")
	i18n.Register(KeyPass4UsageTitle, "用量统计", "Usage")
	i18n.Register(KeyPass4UsageUpdated, "数据更新于 %s · UTC", "Updated at %s · UTC")
	i18n.Register(KeyPass4UsageExport, "导出", "Export")
	i18n.Register(KeyPass4UsageSettings, "设置", "Settings")
	i18n.Register(KeyPass4UsageRefresh, "刷新", "Refresh")
	i18n.Register(KeyPass4UsageEmptyTitle, "暂无统计数据", "No data yet")
	i18n.Register(KeyPass4UsageEmptyBody, "开始使用 DeepSeek Harness 后，这里会展示 Token 消耗情况", "Token usage will appear here once you start using DeepSeek Harness.")
	i18n.Register(KeyPass4AboutTitle, "关于", "About")
	i18n.Register(KeyPass4AboutHint, "本桌面应用与官方 Harness。", "This desktop app and the official Harness.")
	i18n.Register(KeyPass4DesktopVersion, "桌面版本 %s", "Desktop version %s")
	i18n.Register(KeyPass4OnlineInstall, "在线安装", "Install online")
	i18n.Register(KeyPass4InstallUnavailable, "本版本未接入在线安装器，只显示状态。", "The online installer is not wired in this build; this is status only.")
	i18n.Register(KeyPass4OpenRuntime, "打开运行目录", "Open runtime folder")
	i18n.Register(KeyPass4GitHubReleases, "GitHub 发行版", "GitHub releases")
	i18n.Register(KeyPass4OfficialHarness, "官方 Harness", "Official Harness")
	i18n.Register(KeyPass4CredentialStorage, "凭据存储：%s", "Credential storage: %s")
	i18n.Register(KeyPass4RuntimeDir, "运行目录 %s", "Runtime folder %s")
}

// pass4InstallBadgeFill is the reference's near-black installer badge
// (#4b5563). It stays dark when disabled so the control still reads as a badge.
var pass4InstallBadgeFill = ui.RGB(75, 85, 99)
