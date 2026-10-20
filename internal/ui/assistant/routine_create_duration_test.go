package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestBotRoutineDurationUIReviewOnly(t *testing.T) {
	for _, frequency := range []string{"once", "interval"} {
		t.Run(frequency, func(t *testing.T) {
			var creator dshbotstore.RoutineCreator
			key := dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}
			creator.Sync(key, 1, dshbotstore.RoutineSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "client", BotID: "b"}, Loaded: true, Revision: 4, SessionID: "s"})
			creator.Start(key, "Asia/Shanghai")
			creator.Change(key, "prompt", "Fixture prompt")
			submitted := 0
			var schedule string
			tt := ui.NewTester(func(c *ui.Context) {
				s := creator.Snapshot(key)
				m := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true, RoutineCreate: RoutineCreateModel{State: s, HostLabel: "https://fixture.invalid", BotName: "Bot B", OnStart: func() {}, OnChange: func(field, value string) { creator.Change(key, field, value) }, OnReview: func() { creator.Prepare(key) }, OnDismiss: func() { creator.Dismiss(key) }, OnConfirm: func() {
					_, cmd, ok := creator.Begin(key, s.Review)
					if ok {
						submitted++
						schedule = cmd.Edit.Schedule
					}
				}}}
				routineCreateBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
			}, 1100, 2800)
			if err := tt.Click(routineCreateFrequencyLabel(frequency)); err != nil {
				t.Fatal(err)
			}
			if submitted != 0 || creator.Snapshot(key).Draft.Frequency != frequency {
				t.Fatal("selector submitted")
			}
			want := "30m"
			if frequency == "interval" {
				if err := tt.Click(i18n.T(i18n.KeyMiscAssistantMinutes)); err != nil {
					t.Fatal(err)
				}
				want = "every 1m"
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantNewRoutinePaused)); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantReviewCreation)); err != nil {
				t.Fatal(err)
			}
			if submitted != 0 || !tt.HasText(routineCreateImpact(true)) {
				t.Fatal("review did not disclose enabled effect", tt.Texts())
			}
			if creator.Snapshot(key).Review.Command.Edit.Schedule != want {
				t.Fatal("review schedule incorrect")
			}
			// A reviewed form locks frequency and duration; only Back enables changes.
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantDaily)); err == nil && creator.Snapshot(key).Draft.Frequency != frequency {
				t.Fatal("reviewed frequency changed")
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantBackToDraft2)); err != nil {
				t.Fatal(err)
			}
			if submitted != 0 || creator.Snapshot(key).Review != nil {
				t.Fatal("back submitted")
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantReviewCreation)); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmNewRoutine)); err != nil {
				t.Fatal(err)
			}
			if submitted != 1 || schedule != want {
				t.Fatal("explicit confirmation failed", submitted, schedule)
			}
		})
	}
}
