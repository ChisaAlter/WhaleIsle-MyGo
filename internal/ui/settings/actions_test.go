package settings

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestSettingsActionTextHidesProtocolNames(t *testing.T) {
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	i18n.SetLocale(i18n.LocaleZH)
	zhCases := []struct {
		in   string
		want string
	}{
		{"settings document opened", "设置已打开"},
		{"settings document updated", "设置已更新"},
		{"openSettingsDocument: rpc unavailable", "无法打开设置文档：rpc unavailable"},
		{"settings saved", "设置已保存"},
		{"settings mutated", "设置已更新"},
		{"settings read-only", "当前设置处于只读状态"},
		{"viewTabs synced", ""},
		{"viewTabs sync: offline", "同步视图标签失败：offline"},
		{"Skill saved: demo", "技能已保存：demo"},
		{"Skill invocation: 2 updated, 0 failed", "技能调用设置：已更新 2 项，失败 0 项"},
		{"skill created demo", "技能已创建：demo"},
		{"skill hub search term", ""},
		{"settings saved\nviewTabs synced\nSkill saved: demo", "设置已保存\n技能已保存：demo"},
	}
	for _, c := range zhCases {
		if got := settingsActionText(c.in); got != c.want {
			t.Errorf("zh settingsActionText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	i18n.SetLocale("en")
	enCases := []struct {
		in   string
		want string
	}{
		{"settings document opened", "Settings opened"},
		{"settings saved", "Settings saved"},
		{"settings read-only", "Settings are read-only"},
		{"viewTabs synced", ""},
		{"Skill saved: demo", "Skill saved: demo"},
		{"Skill invocation: 2 updated, 0 failed", "Skill invocation: 2 updated, 0 failed"},
		{"skill created demo", "Skill created: demo"},
	}
	for _, c := range enCases {
		if got := settingsActionText(c.in); got != c.want {
			t.Errorf("en settingsActionText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := settingsActionText("已保存"); got != "已保存" {
		t.Errorf("a user-facing action changed: %q", got)
	}
}
