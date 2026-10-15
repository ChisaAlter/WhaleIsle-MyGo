package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotTaskPanelDetailsAndExplicitConfirmation(t *testing.T) {
	confirmed := 0
	reviewed := ""
	state := dshbotstore.TaskControlSnapshot{
		Scope: dshbotstore.TaskScope{HostKey: "synthetic", BotID: "b"}, Loaded: true, Revision: 7, BotName: "Bot B", SelectedID: "t",
		Tasks: []pluginrpc.DshbotTask{{ID: "t", FromID: "a", ToID: "b", Task: "Fixture task", Status: "failed", Constraints: []string{"Fixture constraint"}, SuccessCriteria: "Fixture criteria", ResultSummary: "Fixture result", Error: "Fixture error", Events: []pluginrpc.DshbotTaskEvent{{Type: "failed", Detail: "Fixture audit"}}}},
	}
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true, Tasks: TaskModel{State: state, OnRefresh: func() {}, OnSelect: func(string) {}, OnReview: func(a, id string) {
			reviewed = a + ":" + id
			state.Review = &dshbotstore.TaskReview{Command: pluginrpc.DshbotTaskCommand{Action: a, TaskID: id, BotID: "b", Revision: 7, RequestID: "fixture-operation"}}
		}, OnConfirm: func() { confirmed++ }, OnDismiss: func() { state.Review = nil }}}
		taskControlBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
	}, 1100, 1800)
	for _, want := range []string{"任务 / Task: Fixture task", "成功标准 / Success criteria: Fixture criteria", "结果 / Result: Fixture result", "错误 / Error: Fixture error", taskActionLabel("retry")} {
		if !tt.HasText(want) {
			t.Fatalf("missing %q; %q", want, tt.Texts())
		}
	}
	if tt.HasText(taskActionLabel("pause")) || tt.HasText(taskActionLabel("resume")) {
		t.Fatal("invalid actions visible for failed task")
	}
	if confirmed != 0 {
		t.Fatal("render executed a write")
	}
	if err := tt.Click(taskActionLabel("retry")); err != nil {
		t.Fatal(err)
	}
	if reviewed != "retry:t" || confirmed != 0 {
		t.Fatal("review skipped confirmation")
	}
	if !tt.HasText("确认执行 / Confirm action") {
		t.Fatalf("no confirmation: %q", tt.Texts())
	}
	if err := tt.Click("确认执行 / Confirm action"); err != nil {
		t.Fatal(err)
	}
	if confirmed != 1 {
		t.Fatal("confirm not wired")
	}
}
