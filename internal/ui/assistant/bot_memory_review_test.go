package assistant

import (
	"encoding/json"
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotMemoryReviewUIExplicitEffectsAndConfirmation(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		t.Run(map[bool]string{true: "enable", false: "disable"}[enabled], func(t *testing.T) {
			raw, _ := json.Marshal(!enabled)
			state := dshbotstore.MemoryReviewSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}, Loaded: true, Writable: true, Dirty: true, Draft: enabled, Status: pluginrpc.DshbotCapabilityStatus{Bot: pluginrpc.DshbotCatalogItem{ID: "b", Kind: "bot", SessionID: "s"}, Revision: 8, Profile: map[string]json.RawMessage{"memoryReview": raw, "objective": json.RawMessage(`"Whole fixture objective"`), "objectiveMaxRounds": json.RawMessage(`16`), "model": json.RawMessage(`{"provider":"fixture-provider","model":"fixture-model","reasoningEffort":""}`)}}}
			reads, saves := 0, 0
			tt := ui.NewTester(func(c *ui.Context) {
				m := BotMemoryReviewModel{State: state, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnReview: func() {
					state.Review = &dshbotstore.MemoryReviewConfirmation{Command: pluginrpc.DshbotMemoryReviewCommand{Expected: state.Status, Enabled: enabled}, Sequence: 1}
				}, OnAcknowledge: func(ack bool) { state.Review.Acknowledged = ack }, OnConfirm: func() { saves++ }, OnDismiss: func() { state.Review = nil }}
				botMemoryReviewBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
			}, 1100, 2300)
			if reads != 0 || saves != 0 {
				t.Fatal("render performed action")
			}
			if err := tt.Click("审阅自动记忆配置 / Review memory learning"); err != nil {
				t.Fatal(err)
			}
			text := strings.Join(tt.Texts(), "\n")
			for _, term := range []string{"https://fixture.invalid", "revision 8", "Whole fixture objective", "增改删", "不取消在途", "不清除已有记忆", "重建已暂停目标", "不自动重试", "默认 10", "不创建例行任务"} {
				if !strings.Contains(text, term) {
					t.Fatal("missing effect", term)
				}
			}
			tt.Click("确认保存自动记忆配置 / Confirm memory learning")
			if saves != 0 {
				t.Fatal("saved without acknowledgement")
			}
			if err := tt.Click(botMemoryReviewAcknowledgement); err != nil {
				t.Fatal(err)
			}
			if err := tt.Click("确认保存自动记忆配置 / Confirm memory learning"); err != nil {
				t.Fatal(err)
			}
			if saves != 1 {
				t.Fatal("explicit save missing")
			}
		})
	}
}
