package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
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
	for _, want := range []string{i18n.T(i18n.KeyMiscAssistantTask) + "Fixture task", i18n.T(i18n.KeyMiscAssistantSuccessCriteria) + "Fixture criteria", i18n.T(i18n.KeyMiscAssistantResult) + "Fixture result", i18n.T(i18n.KeyMiscAssistantError) + "Fixture error", taskActionLabel("retry")} {
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
	if !tt.HasText(i18n.T(i18n.KeyMiscAssistantConfirmAction)) {
		t.Fatalf("no confirmation: %q", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmAction)); err != nil {
		t.Fatal(err)
	}
	if confirmed != 1 {
		t.Fatal("confirm not wired")
	}
}
