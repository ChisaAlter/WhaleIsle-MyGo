package settings

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

// TestSettingsNativeTabCopy pins the copy the installed plugins/skills/MCP/agent
// preset tabs paint, including the count-first English group line.
func TestSettingsNativeTabCopy(t *testing.T) {
	old := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(old) })

	i18n.SetLocale(i18n.LocaleEN)
	if got := skillsGroupCountLabel(i18n.T(i18n.KeySurfacesSkillsWorkspaceGroup), 0); got != "0 workspace and personal skills" {
		t.Fatalf("skills group count = %q", got)
	}
	for key, want := range map[string]string{
		KeyPass4ABuiltinHint:       "Inspect the plugins this deployment ships.",
		KeyPass4AMCPEmptySecondary: "Add one to connect local tools or a remote service.",
		KeyPass4TaskDefault:        "New task default",
	} {
		if got := i18n.T(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}

	i18n.SetLocale(i18n.LocaleZH)
	if got := skillsGroupCountLabel(i18n.T(i18n.KeySurfacesSkillsWorkspaceGroup), 0); got != "工作区与个人技能 0 项" {
		t.Fatalf("zh skills group count = %q", got)
	}
	for key, want := range map[string]string{
		KeyPass4ABuiltinHint:       "查看内置部署的插件列表",
		KeyPass4AMCPEmptySecondary: "添加一个服务器以连接本地工具或远程服务。",
		KeyPass4TaskDefault:        "新任务默认",
	} {
		if got := i18n.T(key); got != want {
			t.Fatalf("%s = %q, want %q", key, got, want)
		}
	}
}
