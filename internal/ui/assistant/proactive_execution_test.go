package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/i18n"
)

func TestProactiveExecutionActionsAndScheduleRendering(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(previous)
	started, continued, completed, opened, scheduled := 0, 0, 0, 0, 0
	task := proactive.Task{ID: "task", Title: "核对发行说明", Description: "检查本地变更，给出核对结果与未完成项。", State: "discovered", Autonomy: 1, Source: "manual"}
	m := Model{ThemeID: "deepseek.light"}
	m.Proactive = ProactiveModel{Open: true, CanRun: true, WorkingDirectory: `C:\项目\鲸屿`, Selected: task.ID, State: proactive.Snapshot{Tasks: []proactive.Task{task}}, OnToggle: func() {}, OnSelect: func(string) {}, OnCancel: func(string) {},
		OnRun: func(id string, resume bool) {
			if id != task.ID {
				t.Fatal(id)
			}
			if resume {
				continued++
			} else {
				started++
			}
		},
		OnComplete: func(id string) {
			if id != task.ID {
				t.Fatal(id)
			}
			completed++
		},
		OnOpenSession: func(id string) {
			if id != "execution" {
				t.Fatal(id)
			}
			opened++
		},
		OnSchedule: func(id, at, repeat string) {
			if id != task.ID || at == "" {
				t.Fatal(id, at)
			}
			scheduled++
		},
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 900)
	if err := tt.Click("开始执行"); err != nil {
		t.Fatal(err)
	}
	if started != 1 {
		t.Fatal("start not dispatched")
	}
	if err := tt.Click("安排执行时间"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("保存执行安排"); err != nil {
		t.Fatal(err)
	}
	if scheduled != 1 {
		t.Fatal("schedule not dispatched")
	}
	saveWhaleTasksCapture(t, tt, "proactive-schedule-light.png")
	tt.SetSize(420, 900)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-schedule-narrow.png")
	m.Proactive.State.Tasks[0].State = "verifying"
	m.Proactive.State.Tasks[0].Execution = &proactive.Execution{SessionID: "execution", RequestID: "request", Phase: "completed", Spec: proactive.RunSpec{Cwd: `C:\项目\鲸屿`}, Summary: "已核对运行时、队列与会话恢复。安装升级和真实设备验收仍待完成。"}
	tt.Frame()
	if err := tt.Click("查看执行会话"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("继续任务"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认完成"); err != nil {
		t.Fatal(err)
	}
	if continued != 1 || completed != 1 || opened != 1 {
		t.Fatal(continued, completed, opened)
	}
	m.ThemeID = "deepseek.dark"
	m.SystemDark = true
	tt.SetDark(true)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-result-dark.png")
}

func TestProactiveCreationAutonomyRendering(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(previous)
	m := Model{ThemeID: "deepseek.light"}
	created := 0
	m.Proactive = ProactiveModel{Open: true, Autonomy: 1, Title: "核对安装升级", Description: "检查版本与数据保留，再汇总待处理问题。", OnToggle: func() {}, OnCreate: func() { created++ }}
	m.Proactive.OnAutonomy = func(level int) { m.Proactive.Autonomy = level }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 420, 900)
	if err := tt.Click("L3"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if m.Proactive.Autonomy != 3 {
		t.Fatal("autonomy action not dispatched")
	}
	if err := tt.Click("加入收件箱"); err != nil {
		t.Fatal(err)
	}
	if created != 1 {
		t.Fatal(created)
	}
	saveWhaleTasksCapture(t, tt, "proactive-create-narrow.png")
}
