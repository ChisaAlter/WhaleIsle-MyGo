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

func TestBotNotifyUIExplicitToggleAndConsumerBoundary(t *testing.T) {
	status := pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "bot", SessionID: "session", Kind: "bot"}, Revision: 8, Profile: map[string]json.RawMessage{"notify": json.RawMessage(`false`), "workspaceId": json.RawMessage(`"keep-workspace"`), "model": json.RawMessage(`{"provider":"keep-provider","model":"keep-model","reasoningEffort":"high"}`), "objective": json.RawMessage(`"Keep objective"`)}}
	state := dshbotstore.NotifySnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture", BotID: "bot"}, Loaded: true, Writable: true, Status: status}
	reads, saves, changes := 0, 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotNotifyModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnChange: func(v bool) { changes++; state.Draft = v; state.Dirty = true }, OnReview: func() {
			state.Review = &dshbotstore.NotifyConfirmation{Command: pluginrpc.DshbotNotifyCommand{Expected: status, Enabled: state.Draft}, Sequence: 1}
		}, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnConfirm: func() { saves++ }}
		botNotifyBlock(c, m, false, ui.RGB(15, 17, 21), ui.RGB(129, 133, 140), ui.RGB(65, 118, 230))
	}, 1100, 2400)
	if reads+saves+changes != 0 {
		t.Fatal("render dispatch")
	}
	click := func(s string) {
		if err := tt.Click(s); err != nil {
			t.Fatal(err)
		}
	}
	click(botNotifyToggleLabel())
	if changes != 1 || !state.Draft || saves != 0 {
		t.Fatal("toggle sent")
	}
	click(i18n.T(i18n.KeyMiscAssistantReviewNotificationSetting))
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"notify=false", "notify=true", "https://fixture.invalid", "Bot ID: bot", "Session ID: session", "revision 8", "原版 Bot Web", "窗口未聚焦", "首次观测", "silent", "completed/failed/interrupted", "IM", "不申请系统通知权限", "不证明通知投递", "requestSync", "重建已暂停", "模型", "工具", "外部消息", "费用", "不自动重试", "keep-workspace", "Keep objective"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing boundary", term)
		}
	}
	click(i18n.T(i18n.KeyMiscAssistantConfirmNotificationSetting2))
	if saves != 0 {
		t.Fatal("unacknowledged")
	}
	click(botNotifyAcknowledgement())
	click(i18n.T(i18n.KeyMiscAssistantConfirmNotificationSetting2))
	if saves != 1 {
		t.Fatal("confirm")
	}
}
