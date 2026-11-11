package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/i18n"
)

func TestProactiveToolApprovalAndEffectRendering(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(previous)
	dir := t.TempDir()
	s, err := proactive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.Create(proactive.CreateInput{Title: "提交核对报告", Autonomy: 2})
	if err != nil {
		t.Fatal(err)
	}
	task, err = s.PrepareExecution(task.ID, proactive.RunSpec{HostHome: "test", Cwd: dir}, false)
	if err != nil {
		t.Fatal(err)
	}
	call := proactive.ToolCall{TaskID: task.ID, SessionID: task.Execution.SessionID, RequestID: task.Execution.RequestID, CallID: "exact-call", Name: "send_report", Source: "desktop:acp", Arguments: json.RawMessage(`{"recipient":"项目讨论群","recipientId":9007199254740993,"text":"会话恢复与队列检查已完成，安装升级仍待验收。"}`)}
	d, err := s.PrepareTool(call)
	if err != nil {
		t.Fatal(err)
	}
	m := Model{ThemeID: "deepseek.light", Proactive: ProactiveModel{Open: true, State: s.Snapshot(), Selected: task.ID, OnToggle: func() {}, OnSelect: func(string) {}}}
	m.Proactive.OnDecide = func(id, decision string) {
		if id != d.ApprovalID {
			t.Fatal(id)
		}
		if _, err := s.Decide(id, decision, ""); err != nil {
			t.Fatal(err)
		}
		m.Proactive.State = s.Snapshot()
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 850)
	if !tt.HasText("9007199254740993") || !tt.HasText("项目讨论群") || !tt.HasText("来源：外部 ACP Agent（desktop:acp）") {
		t.Fatal("exact approval parameters or source missing", tt.Texts())
	}
	saveWhaleTasksCapture(t, tt, "proactive-tool-approval-light.png")
	tt.SetSize(420, 850)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-tool-approval-narrow.png")
	if err := tt.Click("批准此操作"); err != nil {
		t.Fatal(err)
	}
	claim, err := s.ClaimTool(call)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishTool(call, claim.Lease, "unknown", "发送结果未确认，请核对目标后再决定。"); err != nil {
		t.Fatal(err)
	}
	m.Proactive.State = s.Snapshot()
	m.ThemeID = "deepseek.dark"
	m.SystemDark = true
	tt.SetDark(true)
	tt.Frame()
	if !tt.HasText("执行结果未知，不会自动重试") {
		t.Fatal("unknown effect hidden", tt.Texts())
	}
	saveWhaleTasksCapture(t, tt, "proactive-tool-effect-dark.png")
}
