package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestBotRoutinePanelRequiresExplicitReview(t *testing.T) {
	confirmed := 0
	reviewed := ""
	state := dshbotstore.RoutineSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}, Loaded: true, Revision: 9, BotName: "Bot B", SessionID: "s", SelectedID: "r", Routines: []pluginrpc.DshbotRoutine{{ID: "r", BotID: "b", Name: "Fixture routine", Prompt: "Fixture objective", Schedule: "every 1h", Timezone: "UTC", Enabled: true, RunHistory: []pluginrpc.DshbotRoutineRun{{RunID: "run1", Status: "failed", Error: "Fixture run failure"}}}}}
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true, Routines: RoutineModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() {}, OnSelect: func(string) {}, OnReview: func(a string) {
			reviewed = a
			state.Review = &dshbotstore.RoutineReview{Command: pluginrpc.DshbotRoutineCommand{Action: a, ID: "r", BotID: "b", Revision: 9, EnabledBefore: true}, Name: "Fixture routine", Prompt: "Fixture objective", SessionID: "s", Schedule: "every 1h", Timezone: "UTC"}
		}, OnConfirm: func() { confirmed++ }, OnDismiss: func() { state.Review = nil }}}
		routineControlBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
	}, 1100, 3000)
	if !tt.HasText(i18n.T(i18n.KeyMiscAssistantPrompt) + "Fixture objective") || !tt.HasText("Fixture run failure") {
		t.Fatal(tt.Texts())
	}
	if confirmed != 0 {
		t.Fatal("render issued write")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantRunNow)); err != nil {
		t.Fatal(err)
	}
	if reviewed != "run" || confirmed != 0 || !tt.HasText(i18n.T(i18n.KeyMiscAssistantConfirmRoutineAction)) {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantBackToRoutine)); err != nil {
		t.Fatal(err)
	}
	if confirmed != 0 || state.Review != nil {
		t.Fatal("cancel issued action")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantDeleteRoutine)); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText(routineImpact(state.Review.Command)) {
		t.Fatal("no deletion consequence")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmRoutineAction)); err != nil {
		t.Fatal(err)
	}
	if confirmed != 1 {
		t.Fatal("confirm not wired")
	}
}
