package assistant

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotObjectiveUIIndependentActionReview(t *testing.T) {
	for _, action := range []string{"start", "pause", "clear"} {
		t.Run(action, func(t *testing.T) {
			var controls dshbotstore.ObjectiveControls
			k := dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}
			controls.Bind(k)
			load, _ := controls.BeginLoad(k)
			value := pluginrpc.DshbotObjectiveStatus{Bot: pluginrpc.DshbotObjectiveBot{DshbotCatalogItem: pluginrpc.DshbotCatalogItem{ID: "b", SessionID: "s", Name: "Fixture"}, Objective: "Whole fixture objective", ObjectiveMaxRounds: 16}, Revision: 9, GoalKnown: true}
			controls.FinishLoad(load, value, nil)
			reads, actions := 0, 0
			tt := ui.NewTester(func(c *ui.Context) {
				s := controls.Snapshot(k)
				m := BotObjectiveModel{State: s, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnReview: func(a string) { controls.Prepare(k, a) }, OnDismiss: func() { controls.Dismiss(k, s.Review) }, OnConfirm: func() {
					if _, cmd, ok := controls.BeginAction(k, s.Review); ok {
						if cmd.Action != action {
							t.Fatal(cmd.Action)
						}
						actions++
					}
				}}
				botObjectiveBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
			}, 1100, 2600)
			if reads != 0 || actions != 0 {
				t.Fatal("render performed operation")
			}
			if err := tt.Click(botObjectiveActionLabel(action)); err != nil {
				t.Fatal(err)
			}
			if actions != 0 {
				t.Fatal("review executed action")
			}
			text := strings.Join(tt.Texts(), "\n")
			for _, word := range []string{"https://fixture.invalid", "Whole fixture objective", "revision 9", "Session: s", "Bot Web", "不自动重发"} {
				if !strings.Contains(text, word) {
					t.Fatal("missing scope or boundary", word)
				}
			}
			if action == "start" && !strings.Contains(text, "发送外部消息") {
				t.Fatal("start hides execution effects")
			}
			if action == "clear" && !strings.Contains(text, "不是暂停或 Bot 全局停止") {
				t.Fatal("clear misrepresented as stop")
			}
			if action != "clear" && !strings.Contains(text, "没有原子 CAS") {
				t.Fatal("invented start/pause CAS")
			}
			if err := tt.Click("取消目标操作 / Cancel objective action"); err != nil {
				t.Fatal(err)
			}
			if actions != 0 {
				t.Fatal("cancel executed")
			}
			if err := tt.Click(botObjectiveActionLabel(action)); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("确认目标操作 / Confirm objective action"); err != nil {
				t.Fatal(err)
			}
			if actions != 1 {
				t.Fatal("confirmed action missing")
			}
		})
	}
}
