package workspacepicker

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestWorkspaceLocaleCatalogZHAndEN(t *testing.T) {
	prev := i18n.Locale()
	t.Cleanup(func() {
		i18n.SetLocale(prev)
	})

	i18n.SetLocale(i18n.LocaleZH)
	if got := WorkspaceLocaleText("draft.initializationFailed"); got != "暂时无法填入草稿，请稍后重试" {
		t.Fatalf("draft.initializationFailed zh = %q", got)
	}
	if got := FormatWorkspaceDeleteDescription("whaleisle"); got != "将把“whaleisle”从工作区列表中移除。文件夹与会话记录会保留，但其会话将不再显示；重新添加该工作目录后会话会回来。" {
		t.Fatalf("FormatWorkspaceDeleteDescription zh = %q", got)
	}
	if got := FormatWorkspaceCreateFailed("offline"); got != "新建会话失败：offline" {
		t.Fatalf("FormatWorkspaceCreateFailed zh = %q", got)
	}
	if got := FormatWorkspaceSubagentsRunning(2); got != "2 个子智能体运行中" {
		t.Fatalf("FormatWorkspaceSubagentsRunning zh = %q", got)
	}
	if got := FormatWorkspaceDateYMD(2026, 11, 7); got != "2026年11月7日" {
		t.Fatalf("FormatWorkspaceDateYMD zh = %q", got)
	}
	if got := FormatWorkspaceRelativeAge(5); got != "5天前" {
		t.Fatalf("FormatWorkspaceRelativeAge(5) zh = %q", got)
	}
	if got := FormatWorkspaceRelativeAge(400); got != "1年前" {
		t.Fatalf("FormatWorkspaceRelativeAge(400) zh = %q", got)
	}

	i18n.SetLocale(i18n.LocaleEN)
	if got := WorkspaceLocaleText("shortcut.noPicker"); got != "Directory picker unavailable" {
		t.Fatalf("shortcut.noPicker en = %q", got)
	}
	if got := FormatWorkspaceSubagentsRunning(1); got != "1 subagent running" {
		t.Fatalf("FormatWorkspaceSubagentsRunning(1) en = %q", got)
	}
	if got := FormatWorkspaceSubagentsRunning(3); got != "3 subagents running" {
		t.Fatalf("FormatWorkspaceSubagentsRunning(3) en = %q", got)
	}
	if got := FormatWorkspaceDateYMD(2026, 11, 7); got != "2026-11-7" {
		t.Fatalf("FormatWorkspaceDateYMD en = %q", got)
	}
	if got := FormatWorkspaceRelativeAge(5); got != "5d ago" {
		t.Fatalf("FormatWorkspaceRelativeAge(5) en = %q", got)
	}

	for key, pair := range workspaceLocaleCatalog {
		if pair.ZH == "" || pair.EN == "" {
			t.Fatalf("workspaceLocaleCatalog[%q] has empty translation: %+v", key, pair)
		}
	}
}
