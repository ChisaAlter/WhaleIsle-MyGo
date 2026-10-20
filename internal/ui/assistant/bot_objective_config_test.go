package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/ui/i18n"
)

func TestBotObjectiveConfigUIWholeReviewAndExecutionAcknowledgement(t *testing.T) {
	state := dshbotstore.ObjectiveConfigSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture-auth", BotID: "b"}, Loaded: true, Writable: true, Dirty: true, Draft: pluginrpc.DshbotObjectiveConfigDraft{Objective: "New complete fixture objective", MaxRounds: "200"}, Status: pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 8, Profile: map[string]json.RawMessage{"objective": json.RawMessage(`"Old complete fixture objective"`), "objectiveMaxRounds": json.RawMessage(`16`), "model": json.RawMessage(`{"provider":"fixture-provider","model":"fixture-model","reasoningEffort":""}`), "workspaceId": json.RawMessage(`"fixture-workspace"`), "capabilities": json.RawMessage(`{"tools":{"mode":"all","names":[]},"skills":{"mode":"selected","names":[]},"mcp":{"mode":"all","names":[]}}`)}}}
	reads, saves := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotObjectiveConfigModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnReview: func() {
			state.Review = &dshbotstore.ObjectiveConfigConfirmation{Command: pluginrpc.DshbotObjectiveConfigCommand{Expected: state.Status, Desired: pluginrpc.DshbotObjectiveConfig{Objective: state.Draft.Objective, MaxRounds: 200}}, Sequence: 1}
		}, OnAcknowledge: func(ack bool) { state.Review.Acknowledged = ack }, OnConfirm: func() { saves++ }, OnDismiss: func() { state.Review = nil }}
		botObjectiveConfigBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
	}, 1100, 2400)
	if reads != 0 || saves != 0 {
		t.Fatal("render performed action")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantReviewObjectiveConfiguration)); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"https://fixture.invalid", "revision 8", "Old complete fixture objective", "New complete fixture objective", "16 → 200", "fixture-workspace", "fixture-provider", "requestSync", "重建已暂停", "模型", "工具", "外部消息", "费用", "不自动重试", "不撤销", "1–200", "2,000"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing review", term)
		}
	}
	tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmObjectiveConfiguration))
	if saves != 0 {
		t.Fatal("unacknowledged save")
	}
	if err := tt.Click(botObjectiveConfigAcknowledgement()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmObjectiveConfiguration)); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit save missing")
	}
}
