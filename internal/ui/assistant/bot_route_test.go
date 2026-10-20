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

func TestBotRouteUIFullReviewAndSeparateImpact(t *testing.T) {
	status := pluginrpc.DshbotRouteStatus{DshbotCapabilityStatus: pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "old-session"}, Revision: 8, Profile: map[string]json.RawMessage{"model": json.RawMessage(`{"provider":"old-provider","model":"old-model","reasoningEffort":"old-effort"}`), "workspaceId": json.RawMessage(`"w1"`)}}, Options: pluginrpc.DshbotRouteOptions{WorkspacesKnown: true, SessionKnown: true, ModelsKnown: true, Default: pluginrpc.DshbotRouteModel{Provider: "default-p", Model: "default-m"}, Workspaces: []pluginrpc.DshbotRouteWorkspace{{ID: "w1", Path: "/host/old"}, {ID: "w2", Path: "/host/new"}}}}
	state := dshbotstore.RouteSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture-account", BotID: "b"}, Loaded: true, Writable: true, Dirty: true, Status: status, Draft: pluginrpc.DshbotRouteValues{Model: pluginrpc.DshbotRouteModel{Provider: "new-provider", Model: "new-model", ReasoningEffort: "new-effort"}, WorkspaceID: "w2"}}
	reads, saves := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotRouteModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnReview: func() {
			state.Review = &dshbotstore.RouteConfirmation{Command: pluginrpc.DshbotRouteCommand{Expected: status, Values: state.Draft}, Sequence: 1}
		}, OnAcknowledge: func(ack bool) { state.Review.Acknowledged = ack }, OnConfirm: func() { saves++ }}
		botRouteBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
	}, 1100, 3200)
	if reads != 0 || saves != 0 {
		t.Fatal("render dispatched")
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantReviewModelAndWorkspace)); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"old-provider", "old-model", "old-effort", "new-provider", "new-model", "new-effort", "/host/old", "/host/new", "old-session", "revision 8", "新的 canonical sessionId", "旧会话不迁移、不删除", "blank=true", "requestSync", "重建已暂停", "模型", "工具", "外部消息", "费用", "不自动重试", "不主动 prompt/start/resume/open", "https://fixture.invalid"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing review", term)
		}
	}
	tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmRouteSave))
	if saves != 0 {
		t.Fatal("unacknowledged saved")
	}
	if err := tt.Click(botRouteAcknowledgement()); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyMiscAssistantConfirmRouteSave)); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit confirm missing")
	}
}
