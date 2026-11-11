package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/ui/i18n"
)

func TestProactiveDeliverySelectionApprovalAndReceipt(t *testing.T) {
	prev := i18n.Locale()
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(prev)
	route := proactive.DeliveryRoute{ID: "source", Title: "项目讨论会话", Channel: "feishu", ChatID: "chat-A", BotID: "bot-A", Actor: "owner-A"}
	task := proactive.Task{ID: "task", Title: "发布核对结果", State: "verifying", Autonomy: 2, Execution: &proactive.Execution{Phase: "completed", Summary: "核心链通过，安装升级仍待验收。"}}
	m := Model{ThemeID: "deepseek.light", Proactive: ProactiveModel{Open: true, CanRun: true, Selected: task.ID, State: proactive.Snapshot{Tasks: []proactive.Task{task}}, DeliveryRoutes: []proactive.DeliveryRoute{route}, OnDeliveryRefresh: func() {}}}
	m.Proactive.OnDelivery = func(id, routeID, text string) {
		if id != task.ID || routeID != route.ID || text != task.Execution.Summary {
			t.Fatal(id, routeID, text)
		}
		m.Proactive.State.Outbox = []proactive.Approval{{ID: "approval", TaskID: id, Status: "pending", Action: "external_message", Risk: "review-required", DataClass: "task-result", Delivery: &proactive.Delivery{Route: route, Text: text, Status: "pending"}}}
	}
	m.Proactive.OnDecide = func(id, decision string) {
		if id != "approval" || decision != "approved" {
			t.Fatal(id, decision)
		}
		m.Proactive.State.Outbox[0].Status = "approved"
		m.Proactive.State.Outbox[0].Delivery.Status = "sent"
		m.Proactive.State.Outbox[0].Delivery.MessageIDs = []string{"message-receipt-A"}
	}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 800, 1000)
	if err := tt.Click("发送结果到飞书"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("项目讨论会话"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if err := tt.Click("提交发送审批"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if !tt.HasText("chat-A") || !tt.HasText("owner-A") || !tt.HasText(task.Execution.Summary) {
		t.Fatal(tt.Texts())
	}
	saveWhaleTasksCapture(t, tt, "proactive-delivery-light.png")
	tt.SetSize(420, 1000)
	tt.Frame()
	saveWhaleTasksCapture(t, tt, "proactive-delivery-narrow.png")
	if err := tt.Click("批准此操作"); err != nil {
		t.Fatal(err)
	}
	m.ThemeID = "deepseek.dark"
	m.SystemDark = true
	tt.SetDark(true)
	tt.Frame()
	if !tt.HasText("message-receipt-A") {
		t.Fatal(tt.Texts())
	}
	saveWhaleTasksCapture(t, tt, "proactive-delivery-receipt-dark.png")
}
