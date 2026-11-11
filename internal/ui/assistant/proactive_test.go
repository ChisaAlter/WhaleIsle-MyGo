package assistant

import (
	"encoding/json"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/i18n"
)

func TestProactiveInboxDecisionAndRendering(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(previous)
	dir := t.TempDir()
	service, err := proactive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	task, err := service.Create(proactive.CreateInput{Title: "核对鲸屿更新说明", Description: "逐项核对功能与已完成的验证，附上可追溯的结果。", Autonomy: 3, Source: "github"})
	if err != nil {
		t.Fatal(err)
	}
	approval, err := service.RequestApproval(proactive.ApprovalInput{TaskID: task.ID, Action: "将核对报告发送到指定项目群", Target: json.RawMessage(`"项目讨论群"`), Risk: "medium"})
	if err != nil {
		t.Fatal(err)
	}
	m := Model{ThemeID: "deepseek.light"}
	m.Proactive = ProactiveModel{Open: true, State: service.Snapshot(), Selected: task.ID, OnToggle: func() {}}
	m.Proactive.OnDecide = func(id, decision string) {
		if id != approval.ID {
			t.Fatal("wrong approval target")
		}
		if _, err := service.Decide(id, decision, ""); err != nil {
			t.Fatal(err)
		}
		m.Proactive.State = service.Snapshot()
	}
	m.Proactive.OnSelect = func(id string) { m.Proactive.Selected = id }
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 760)
	if !tt.HasText("需要你决定") || !tt.HasText("将核对报告发送到指定项目群") {
		t.Fatal(tt.Texts())
	}
	saveWhaleTasksCapture(t, tt, "proactive-inbox-light.png")
	tt.SetSize(420, 760)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-inbox-narrow.png")
	m.ThemeID, m.SystemDark = "deepseek.dark", true
	tt.SetDark(true)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-inbox-dark.png")
	if err := tt.Click("批准此操作"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	approvalButtonRemains := false
	for _, text := range tt.Texts() {
		approvalButtonRemains = approvalButtonRemains || text == "批准此操作"
	}
	if approvalButtonRemains || !tt.HasText("已批准，待继续") {
		t.Fatal("decision did not reach rendered task", tt.Texts())
	}
	reopened, err := proactive.Open(dir)
	if err != nil || reopened.Snapshot().Outbox[0].Status != "approved" {
		t.Fatal("click not persisted", err)
	}
}
