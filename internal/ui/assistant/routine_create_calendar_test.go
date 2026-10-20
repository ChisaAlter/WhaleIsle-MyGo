package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestBotRoutineCalendarUIReview(t *testing.T) {
	for _, tc := range []struct{ frequency, want string }{{"hourly", "every 1h"}, {"weekdays", "0 9 * * 1-5"}, {"weekly", "0 9 * * 0"}, {"monthly", "0 9 31 * *"}, {"advanced", "30 2 * * *"}} {
		t.Run(tc.frequency, func(t *testing.T) {
			var creator dshbotstore.RoutineCreator
			key := dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}
			creator.Sync(key, 1, dshbotstore.RoutineSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "client", BotID: "b"}, Loaded: true, Revision: 4, SessionID: "s"})
			creator.Start(key, "America/New_York")
			creator.Change(key, "prompt", "Fixture prompt")
			creator.Change(key, "monthDay", "31")
			creator.Change(key, "cron", "30 2 * * *")
			submitted := 0
			var schedule string
			tt := ui.NewTester(func(c *ui.Context) {
				s := creator.Snapshot(key)
				m := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true, RoutineCreate: RoutineCreateModel{State: s, HostLabel: "https://fixture.invalid", BotName: "Bot B", OnStart: func() {}, OnChange: func(field, value string) { creator.Change(key, field, value) }, OnReview: func() { creator.Prepare(key) }, OnConfirm: func() {
					_, cmd, ok := creator.Begin(key, s.Review)
					if ok {
						submitted++
						schedule = cmd.Edit.Schedule
					}
				}}}
				routineCreateBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
			}, 1100, 3000)
			if err := tt.Click(routineCreateFrequencyLabel(tc.frequency)); err != nil {
				t.Fatal(err)
			}
			if tc.frequency == "weekly" {
				if err := tt.Click(i18n.T(i18n.KeyMiscAssistantSun)); err != nil {
					t.Fatal(err)
				}
			}
			if !tt.HasText(routineCreateTimingHint(tc.frequency)) {
				t.Fatal("Host timing/validation boundary missing", tt.Texts())
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantReviewCreation)); err != nil {
				t.Fatal(err)
			}
			if submitted != 0 || !tt.HasText(routineCreateImpact(false)) || creator.Snapshot(key).Review.Command.Edit.Schedule != tc.want {
				t.Fatal("review created or schedule differs")
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmNewRoutine)); err != nil {
				t.Fatal(err)
			}
			if submitted != 1 || schedule != tc.want {
				t.Fatal("explicit confirmation missing")
			}
		})
	}
}
