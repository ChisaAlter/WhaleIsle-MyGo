package assistant

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotMemoryUIReviewDeleteOnlyOnConfirm(t *testing.T) {
	var controls dshbotstore.MemoryControls
	k := dshbotstore.RoutineScope{HostKey: "fixture-account", BotID: "b"}
	controls.Bind(k)
	load, _ := controls.BeginLoad(k)
	v := pluginrpc.DshbotMemoryResult{BotID: "b", Bot: pluginrpc.DshbotMemoryTrack{Entries: []string{"fixture entry"}, Text: "- fixture entry", Revision: "bot-cas"}, User: pluginrpc.DshbotMemoryTrack{Revision: "user-cas"}, Limits: pluginrpc.DshbotMemoryLimits{Bot: 8000, User: 4000}}
	controls.FinishLoad(load, pluginrpc.DshbotCatalogItem{ID: "b", Name: "Fixture Bot", SessionID: "session"}, v, true, nil)
	saves, reads := 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		s := controls.Snapshot(k)
		m := BotMemoryModel{State: s, HostLabel: "https://fixture.invalid", OnRefresh: func() { reads++ }, OnTrack: func(track string) { controls.SetTrack(k, track) }, OnEdit: func(index int) { controls.StartEdit(k, index) }, OnDelete: func(index int) { controls.PrepareDelete(k, index) }, OnChange: func(text string) { controls.Change(k, text) }, OnReview: func() { controls.Prepare(k) }, OnDismiss: func() { controls.Dismiss(k, s.Review) }, OnDiscard: func() { controls.Discard(k) }, OnConfirm: func() {
			if _, cmd, ok := controls.BeginSave(k, s.Review); ok {
				if cmd.Track != "bot" || cmd.Text != "" {
					t.Fatal(cmd)
				}
				saves++
			}
		}}
		botMemoryBlock(c, m, false, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150), ui.RGB(70, 130, 210))
	}, 1100, 3400)
	if reads != 0 || saves != 0 {
		t.Fatal("render performed RPC")
	}
	if err := tt.Click("删除记忆条目 1… / Delete entry 1…"); err != nil {
		t.Fatal(err)
	}
	if saves != 0 {
		t.Fatal("delete skipped explicit review")
	}
	texts := strings.Join(tt.Texts(), "\n")
	for _, word := range []string{"https://fixture.invalid", "session", "bot-cas", "- fixture entry", "整轨文件", "另一轨", "空内容"} {
		if !strings.Contains(texts, word) {
			t.Fatal("review omitted impact", word)
		}
	}
	if err := tt.Click("返回记忆编辑 / Back to memory edit"); err != nil {
		t.Fatal(err)
	}
	if saves != 0 {
		t.Fatal("cancel wrote")
	}
	controls.Prepare(k)
	tt.Frame()
	if err := tt.Click("确认替换此轨记忆 / Confirm track replacement"); err != nil {
		t.Fatal(err)
	}
	if saves != 1 {
		t.Fatal("explicit replacement missing")
	}
}
