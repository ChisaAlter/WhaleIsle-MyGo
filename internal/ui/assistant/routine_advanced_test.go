package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestBotRoutineAdvancedUIExplicitCommandAcknowledgement(t *testing.T) {
	for _, action := range []string{"save", "toggle"} {
		t.Run(action, func(t *testing.T) {
			var controls dshbotstore.RoutineControls
			k := dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}
			controls.Bind(k)
			ticket, _ := controls.BeginLoad(k)
			view := &pluginrpc.DshbotRoutineView{NS: "dsh-bot", Revision: 9}
			view.Value.Items = []pluginrpc.DshbotCatalogItem{{ID: "b", SessionID: "s", WorkspaceID: "/configured/workspace"}}
			view.Value.Routines = []pluginrpc.DshbotRoutine{{ID: "r", BotID: "b", Name: "Fixture routine", Prompt: "Fixture prompt", Schedule: "every 1h", Timezone: "UTC", Enabled: action == "save", ContextFrom: []string{"self"}, SilentAllowed: true, Watch: json.RawMessage(`{"kind":"command","value":"fixture-only-complete-command --synthetic","timeoutMs":15000}`)}}
			controls.FinishLoad(ticket, view, nil, nil, nil)
			controls.Select(k, "r")
			if action == "save" {
				controls.StartEdit(k)
			}
			submitted := 0
			tt := ui.NewTester(func(c *ui.Context) {
				snapshot := controls.Snapshot(k)
				model := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true, Routines: RoutineModel{State: snapshot, HostLabel: "https://fixture.invalid", OnRefresh: func() {}, OnChange: func(field, value string) { controls.Edit(k, field, value) }, OnReview: func(a string) { controls.Prepare(k, a) }, OnAcknowledgeWatch: func(ack bool) { controls.AcknowledgeWatch(k, snapshot.Review, ack) }, OnConfirm: func() {
					if _, _, ok := controls.BeginAction(k, snapshot.Review); ok {
						submitted++
					}
				}}}
				routineControlBlock(c, model, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
			}, 1100, 4000)
			if err := tt.Click(routineActionLabel(action, action == "save")); err != nil {
				t.Fatal(err)
			}
			texts := strings.Join(tt.Texts(), "\n")
			for _, value := range []string{"fixture-only-complete-command --synthetic", "/configured/workspace", "DSH_HOME", "DSHD_HOME", "实际执行目录此刻无法核实", "检查失败仍可能排队执行 Bot", "本任务上一轮输出"} {
				if !strings.Contains(texts, value) {
					t.Fatal("review hides effective consequence", value, texts)
				}
			}
			_ = tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmRoutineAction))
			if submitted != 0 {
				t.Fatal("missing explicit command acknowledgement")
			}
			if err := tt.Click(routineWatchAcknowledgement()); err != nil {
				t.Fatal(err)
			}
			if submitted != 0 || !controls.Snapshot(k).Review.WatchAcknowledged {
				t.Fatal("acknowledgement sent mutation or missing")
			}
			if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmRoutineAction)); err != nil {
				t.Fatal(err)
			}
			if submitted != 1 {
				t.Fatal("confirmed command config missing")
			}
		})
	}
}
