package assistant

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	assiststore "whaleisle/internal/store/assistant"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotRoutineCreateEmptyListReview(t *testing.T) {
	starts, submitted := 0, 0
	state := dshbotstore.RoutineCreateSnapshot{Scope: dshbotstore.RoutineScope{HostKey: "fixture", BotID: "b"}, Loaded: true}
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Mode: assiststore.ModeDshbot, HarnessReady: true, DshbotEnabled: true, RoutineCreate: RoutineCreateModel{State: state, HostLabel: "https://fixture.invalid", BotName: "Bot B", OnStart: func() {
			starts++
			state.Draft = &dshbotstore.RoutineCreateDraft{Name: "新定时任务", Prompt: "Fixture prompt", Hour: "09", Minute: "00", Timezone: "Asia/Shanghai", MaxRuns: "0", Revision: 4}
		}, OnChange: func(field, value string) {
			if field == "enabled" {
				state.Draft.Enabled = value == "true"
			}
		}, OnReview: func() {
			d := state.Draft
			state.Review = &dshbotstore.RoutineCreateReview{Command: pluginrpc.DshbotRoutineCreateCommand{BotID: "b", SessionID: "s", Revision: 4, RequestID: "local-fixture", Edit: pluginrpc.DshbotRoutineEdit{Name: d.Name, Prompt: d.Prompt, Schedule: "0 9 * * *", Timezone: d.Timezone, Enabled: d.Enabled}}}
		}, OnConfirm: func() { submitted++ }, OnDismiss: func() { state.Review = nil }}}
		routineCreateBlock(c, m, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(20, 20, 20), ui.RGB(80, 80, 80), ui.RGB(80, 130, 220))
	}, 1100, 2500)
	if err := tt.Click("新建例行任务 / New routine"); err != nil {
		t.Fatal(err)
	}
	if starts != 1 || submitted != 0 {
		t.Fatal("new button performed create")
	}
	if err := tt.Click("新任务：暂停 / New routine paused"); err != nil {
		t.Fatal(err)
	}
	if !state.Draft.Enabled || submitted != 0 {
		t.Fatal("local enable performed write")
	}
	if err := tt.Click("审阅新建… / Review creation…"); err != nil {
		t.Fatal(err)
	}
	if submitted != 0 || !tt.HasText(routineCreateImpact(true)) {
		t.Fatal(tt.Texts())
	}
	if err := tt.Click("返回修改 / Back to draft"); err != nil {
		t.Fatal(err)
	}
	if submitted != 0 || state.Review != nil {
		t.Fatal("Back submitted")
	}
	if err := tt.Click("审阅新建… / Review creation…"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认保存新例行任务 / Confirm new routine"); err != nil {
		t.Fatal(err)
	}
	if submitted != 1 {
		t.Fatal("confirmation missing")
	}
}
