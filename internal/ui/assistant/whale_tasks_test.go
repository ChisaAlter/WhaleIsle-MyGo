package assistant

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestWhaleTaskPanelActionsAndRendering(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(previous)
	started, picked, owner, revoke := 0, 0, 0, 0
	var controlled, opened string
	m := Model{ThemeID: "deepseek.light", Mode: assiststore.ModeWhale, HarnessReady: true, WhaleEnabled: true}
	m.WhaleTasks = WhaleTasksModel{State: assiststore.WhaleTaskSnapshot{Host: "host", Loaded: true, Goal: "整理项目交接要点", Files: "C:\\资料\\交接.md", Catalog: pluginrpc.WhaleTasks{Available: true}},
		OnRefresh: func() {}, OnStart: func() { started++ }, OnPickFiles: func() { picked++ }, OnAuthorize: func() { owner++ }, OnRevoke: func() { revoke++ },
		OnEdit: func(field, value string) {
			switch field {
			case "bot":
				m.WhaleTasks.State.BotID = value
			case "actor":
				m.WhaleTasks.State.Actor = value
			}
		},
		OnControl: func(id, action string) { controlled = id + ":" + action }, OnOpenSession: func(id string) { opened = id },
	}
	tt := ui.NewTester(func(c *ui.Context) {
		colors := theme.Resolve(m.ThemeID).Active(m.SystemDark)
		ui.Scroll(c).Fill().Background(chrome.Color(colors.BG)).Children(func() {
			ui.Column(c).FillWidth().Padding(20).Children(func() {
				whaleTasksBlock(c, m, chrome.Color(colors.FG), chrome.Color(colors.Muted), chrome.Color(colors.Border), chrome.Color(colors.Accent))
			})
		})
	}, 800, 900)
	if err := tt.Click("选择资料"); err != nil {
		t.Fatal(err)
	}
	if picked != 1 {
		t.Fatal("file picker not dispatched")
	}
	if err := tt.Click("开始整理"); err != nil {
		t.Fatal(err)
	}
	if started != 1 {
		t.Fatal("start not dispatched")
	}
	m.WhaleTasks.State.CreateUnknown = true
	tt.Frame()
	_ = tt.Click("开始整理")
	if started != 1 {
		t.Fatal("unknown start repeated")
	}
	saveWhaleTasksCapture(t, tt, "whale-task-form.png")
	m.WhaleTasks.State.CreateUnknown = false
	m.WhaleTasks.State.Goal = ""
	m.WhaleTasks.State.Files = ""
	var task pluginrpc.WhaleTask
	task.ID, task.Goal, task.State = "task-1", "整理项目交接要点", "running"
	task.Model.Provider, task.Model.Model = "deepseek", "deepseek-flash"
	task.Inputs = []pluginrpc.WhaleTaskInput{{ID: "input-1", Name: "交接.md"}}
	m.WhaleTasks.State.Catalog.Tasks = []pluginrpc.WhaleTask{task}
	tt.Frame()
	if tt.HasText("整理目标") {
		t.Fatal("accepted task did not reveal results")
	}
	if err := tt.Click("停止任务"); err != nil {
		t.Fatal(err)
	}
	if controlled != "task-1:stop" {
		t.Fatal(controlled)
	}
	m.WhaleTasks.State.Catalog.Tasks[0].State = "interrupted"
	tt.Frame()
	if err := tt.Click("继续任务"); err != nil {
		t.Fatal(err)
	}
	if controlled != "task-1:resume" {
		t.Fatal(controlled)
	}
	task.State = "completed"
	task.Result = &pluginrpc.WhaleTaskResult{Summary: "已核对工作区归属和恢复流程。先完成运行时对齐，再验收手机端与安装升级。", Evidence: []string{"input-1"}, Outcome: "completed"}
	task.Effects = []pluginrpc.WhaleTaskEffect{{ID: "delivery", Target: "session", State: "sent"}}
	task.Execution = &struct {
		SessionID string `json:"sessionId"`
		Requests  int    `json:"requests"`
		ReadBytes int64  `json:"readBytes"`
	}{SessionID: "task-session"}
	m.WhaleTasks.State.Catalog.Tasks[0] = task
	tt.Frame()
	if !tt.HasText("来源：交接.md") || !tt.HasText("会话 · 已投递") {
		t.Fatal("missing report provenance or receipt", tt.Texts())
	}
	if err := tt.Click("查看执行记录"); err != nil {
		t.Fatal(err)
	}
	if opened != "task-session" {
		t.Fatal(opened)
	}
	if err := tt.Click("飞书后台任务授权"); err != nil {
		t.Fatal(err)
	}
	m.WhaleTasks.State.BotID = "feishu-bot"
	m.WhaleTasks.State.Actor = "open-id"
	tt.Frame()
	if err := tt.Click("授权此人"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("撤销此机器人授权"); err != nil {
		t.Fatal(err)
	}
	if owner != 1 || revoke != 1 {
		t.Fatal("owner controls not dispatched")
	}
	_ = tt.Click("飞书后台任务授权")
	tt.SetSize(420, 720)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "whale-task-result-narrow.png")
	m.ThemeID = "deepseek.dark"
	m.SystemDark = true
	tt.SetDark(true)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "whale-task-result-dark.png")
}

func saveWhaleTasksCapture(t *testing.T, tt *ui.Tester, name string) {
	if root := os.Getenv("WHALE_ALIGNMENT_CAPTURE_DIR"); root != "" {
		if err := os.MkdirAll(root, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err = png.Encode(f, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}
