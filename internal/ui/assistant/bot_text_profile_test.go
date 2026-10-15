package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotTextProfileUICompleteTextAndIndependentImpact(t *testing.T) {
	status := pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 8, Profile: map[string]json.RawMessage{"name": json.RawMessage(`"old-address"`), "title": json.RawMessage(`"Old display"`), "description": json.RawMessage(`"Old complete persona\nOld final line"`), "workspaceId": json.RawMessage(`"workspace-keep"`), "model": json.RawMessage(`{"provider":"keep-provider","model":"keep-model","reasoningEffort":"high"}`)}}
	next := pluginrpc.DshbotTextProfileValues{Name: "new-address", Title: "", Description: "New complete persona\nNew final line"}
	state := dshbotstore.TextProfileSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture-account", BotID: "b"}, Loaded: true, Writable: true, Dirty: true, Status: status, Draft: next}
	reads, saves := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotTextProfileModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnReview: func() {
			state.Review = &dshbotstore.TextProfileConfirmation{Command: pluginrpc.DshbotTextProfileCommand{Expected: status, Values: next}, Sequence: 1}
		}, OnAcknowledge: func(ack bool) { state.Review.Acknowledged = ack }, OnConfirm: func() { saves++ }}
		botTextProfileBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
	}, 1100, 3600)
	if reads != 0 || saves != 0 {
		t.Fatal("render action")
	}
	if err := tt.Click("审阅文本资料 / Review text profile"); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"old-address", "new-address", "Old display", "Old final line", "New final line", "(空 / empty)", "revision 8", "presentation", "rename", "requestSync", "重建已暂停", "模型", "工具", "外部消息", "费用", "@name", "系统提示词", "不自动重试", "不主动 prompt/start/resume/open", "12000", "workspace-keep", "keep-provider", "https://fixture.invalid"} {
		if !strings.Contains(text, term) {
			t.Fatal("missing full review", term)
		}
	}
	tt.Click("确认保存文本资料 / Confirm text profile save")
	if saves != 0 {
		t.Fatal("unacknowledged")
	}
	if err := tt.Click(botTextProfileAcknowledgement); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认保存文本资料 / Confirm text profile save"); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit save missing")
	}
}
