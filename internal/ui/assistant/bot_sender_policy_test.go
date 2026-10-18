package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotSenderPolicyUISeparateRightsAndExplicitExpansion(t *testing.T) {
	status := pluginrpc.DshbotSenderPolicyStatus{DshbotCapabilityStatus: pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Name: "Fixture", Kind: "bot", SessionID: "s"}, Revision: 8, PeerIDs: []string{"peer-a", "peer-b"}, Profile: map[string]json.RawMessage{"allowedSenderIds": json.RawMessage(`[]`), "objective": json.RawMessage(`"Existing objective"`), "model": json.RawMessage(`{"provider":"p","model":"m"}`)}}, Peers: []pluginrpc.DshbotCatalogItem{{ID: "peer-a", Name: "Same label"}, {ID: "peer-b", Name: "Same label", Hidden: true}}, SharedRoomPeerIDs: []string{"peer-a"}}
	state := dshbotstore.SenderPolicySnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}, Loaded: true, Writable: true, Dirty: true, Status: status, Draft: pluginrpc.DshbotSenderPolicyValues{Selected: true, IDs: []string{"peer-b"}}}
	reads, saves := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := BotSenderPolicyModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnReview: func() {
			state.Review = &dshbotstore.SenderPolicyConfirmation{Command: pluginrpc.DshbotSenderPolicyCommand{Expected: state.Status, Values: state.Draft}, Sequence: 1}
		}, OnAcknowledge: func(v bool) { state.Review.Acknowledged = v }, OnExpansionAcknowledge: func(v bool) { state.Review.ExpansionAcknowledged = v }, OnConfirm: func() { saves++ }, OnDismiss: func() { state.Review = nil }}
		botSenderPolicyBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
	}, 1300, 4000)
	if reads != 0 || saves != 0 {
		t.Fatal("render caused action")
	}
	if err := tt.Click("审阅消息与任务来源 / Review source policy"); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(tt.Texts(), "\n")
	for _, term := range []string{"https://fixture.invalid", "revision 8", "普通消息 / Ordinary messages", "直接任务委派 / Direct task delegation", "id=peer-a", "id=peer-b", "hidden", "allowedSenderIds", "未来", "Existing objective", "requestSync", "重建已暂停目标", "不取消", "不自动重试"} {
		if !strings.Contains(all, term) {
			t.Fatal("missing review", term)
		}
	}
	tt.Click("确认保存来源规则 / Confirm source save")
	if saves != 0 {
		t.Fatal("no impact acknowledgement")
	}
	if err := tt.Click(botSenderPolicyAcknowledgement); err != nil {
		t.Fatal(err)
	}
	tt.Click("确认保存来源规则 / Confirm source save")
	if saves != 0 {
		t.Fatal("permission expansion without specific approval")
	}
	if err := tt.Click(botSenderPolicyExpansionAcknowledgement); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认保存来源规则 / Confirm source save"); err != nil || saves != 1 {
		t.Fatal(err, saves)
	}
	if err := tt.Click("取消来源保存 / Cancel source save"); err != nil || state.Review != nil {
		t.Fatal("dismiss failed", err)
	}
}
