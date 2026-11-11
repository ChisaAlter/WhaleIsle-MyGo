package assistant

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/i18n"
)

func TestProactiveSourcesAndResultFeed(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(previous)
	added, paused, refreshed, retried, read := 0, 0, 0, 0, 0
	n := proactive.Notice{ID: "result:1", TaskID: "task", Kind: "result", Title: "核对运行时升级", Body: "已核对会话恢复与队列排序。安装升级仍待验收，结果没有自动外发。", CreatedAt: "2026-11-11T00:01:00Z", DesktopStatus: "unavailable"}
	m := Model{ThemeID: "deepseek.light"}
	m.Proactive = ProactiveModel{Open: true, SourceRepository: "owner/another", OnToggle: func() {}, State: proactive.Snapshot{
		Tasks:   []proactive.Task{{ID: "task", Title: "核对运行时升级", Description: "核对升级前后行为", State: "verifying", Source: "manual", Autonomy: 1}},
		Notices: []proactive.Notice{n}, Sources: []proactive.Source{{ID: "github:owner/repo", Repository: "owner/repo", Enabled: true, LastSuccess: "2026-11-11T00:00:00Z"}},
	}, OnSourceEdit: func(value string) { m.Proactive.SourceRepository = value }, OnSourceAdd: func() { added++ }, OnSourceToggle: func(repo string, enabled bool) {
		if repo != "owner/repo" || enabled {
			t.Fatal(repo, enabled)
		}
		paused++
	}, OnSourceRefresh: func(id string) {
		if id != "github:owner/repo" {
			t.Fatal(id)
		}
		refreshed++
	}, OnNoticeOpen: func(got proactive.Notice) {
		if got.ID != n.ID {
			t.Fatal(got)
		}
	}, OnNoticeRead: func(id string) {
		if id != n.ID {
			t.Fatal(id)
		}
		read++
	}, OnNoticeRetry: func(id string) {
		if id != n.ID {
			t.Fatal(id)
		}
		retried++
	}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 1000)
	if err := tt.Click("查看动态"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(n.Body) || !tt.HasText("桌面提醒不可用") {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click("重试桌面提醒"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("标为已读"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("GitHub 来源 (1)"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("暂停来源"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("立即检查"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("添加来源"); err != nil {
		t.Fatal(err)
	}
	if added != 1 || paused != 1 || refreshed != 1 || retried != 1 || read != 1 {
		t.Fatal(added, paused, refreshed, retried, read)
	}
	saveWhaleTasksCapture(t, tt, "proactive-sources-light.png")
	tt.SetSize(420, 1000)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-sources-narrow.png")
	m.ThemeID = "deepseek.dark"
	m.SystemDark = true
	tt.SetDark(true)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-sources-dark.png")
}
