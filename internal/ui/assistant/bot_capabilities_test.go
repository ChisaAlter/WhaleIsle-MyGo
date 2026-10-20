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

func TestBotCapabilitiesUIReviewRequiresImpactAcknowledgement(t *testing.T) {
	policies := pluginrpc.DshbotCapabilityPolicies{"tools": {Mode: "all", Names: []string{}}, "skills": {Mode: "selected", Names: []string{}}, "mcp": {Mode: "selected", Names: []string{}}}
	desired := pluginrpc.CloneDshbotCapabilityPolicies(policies)
	desired["tools"] = pluginrpc.DshbotCapabilityPolicy{Mode: "selected", Names: []string{"read"}}
	state := dshbotstore.CapabilitySnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture-auth", BotID: "b"}, Loaded: true, Writable: true, Dirty: true, Draft: desired, Status: pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s", Name: "Fixture"}, Revision: 8, Policies: policies, Profile: map[string]json.RawMessage{"objective": json.RawMessage(`"Whole existing objective"`), "workspaceId": json.RawMessage(`"/fixture"`), "objectiveMaxRounds": json.RawMessage(`16`)}, Catalog: pluginrpc.DshbotCapabilityCatalog{Active: true, SkillsAvailable: true, Tools: []pluginrpc.DshbotCapabilityEntry{{Name: "read", Description: "Fixture reader"}}}}}
	reads, saves := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotCapabilityModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnReview: func() {
			state.Review = &dshbotstore.CapabilityReview{Command: pluginrpc.DshbotCapabilityCommand{Expected: state.Status, Desired: desired}, Sequence: 1}
		}, OnDismiss: func() { state.Review = nil }, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { saves++ }}
		botCapabilitiesBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
	}, 1000, 3000)
	if reads != 0 || saves != 0 {
		t.Fatal("render performed RPC")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantReviewCapabilityChanges)); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"https://fixture.invalid", "revision 8", "Whole existing objective", "重建已暂停目标", "外部消息", "tools", "read", "all", "selected", "不自动重试"} {
		if !strings.Contains(text, term) {
			t.Fatal("hidden impact/scope", term)
		}
	}
	tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmCapabilitySave))
	if saves != 0 {
		t.Fatal("saved without acknowledgement")
	}
	if err := tt.Click(botCapabilityAcknowledgement()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmCapabilitySave)); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit confirmation missing")
	}
}
