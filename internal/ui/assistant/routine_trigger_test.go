package assistant

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/desktop/pluginrpc"
	dshbotstore "whaleisle/internal/store/dshbot"
)

func TestBotRoutineTriggerUIIndependentConfirmation(t *testing.T) {
	var cstore dshbotstore.RoutineControls
	k := dshbotstore.RoutineScope{HostKey: "fixture-account", BotID: "b"}
	cstore.Bind(k)
	load, _ := cstore.BeginLoad(k)
	view := &pluginrpc.DshbotRoutineView{NS: "dsh-bot", Revision: 4}
	view.Value.Items = []pluginrpc.DshbotCatalogItem{{ID: "b", SessionID: "s"}}
	view.Value.Routines = []pluginrpc.DshbotRoutine{{ID: "r", BotID: "b", Name: "Fixture routine"}}
	cstore.FinishLoad(load, view, nil, nil, nil)
	cstore.Select(k, "r")
	obtains := 0
	var copies []string
	tt := ui.NewTester(func(c *ui.Context) {
		state := cstore.Snapshot(k)
		seq := state.Trigger.Sequence
		m := RoutineModel{State: state, HostLabel: "https://fixture.invalid", OnPrepareTrigger: func() { cstore.PrepareTrigger(k) }, OnAcknowledgeTrigger: func(ack bool) { cstore.AcknowledgeTrigger(k, seq, ack) }, OnConfirmTrigger: func() {
			if ticket, ok := cstore.BeginTrigger(k, seq); ok {
				obtains++
				cstore.FinishTrigger(ticket, []byte("synthetic-ui-credential"), nil)
			}
		}, OnCloseTrigger: func() { cstore.CloseTrigger(k, seq) }, TriggerMaterial: func() (string, string) {
			address, header, _ := cstore.TriggerMaterial(k, seq, "https://fixture.invalid")
			return address, header
		}, OnCopyTrigger: func(part string) {
			address, header, _ := cstore.TriggerMaterial(k, seq, "https://fixture.invalid")
			if part == "address" {
				copies = append(copies, address)
			} else if part == "header" {
				copies = append(copies, header)
			}
		}}
		routineTriggerBlock(c, m, state.Busy, ui.RGB(255, 255, 255), ui.RGB(160, 160, 160), ui.RGB(80, 130, 220))
	}, 1100, 2800)
	if obtains != 0 || len(copies) != 0 {
		t.Fatal("render acquired/copied credential")
	}
	if err := tt.Click("审阅获取当前 Host 的 Bot 目录触发凭据… / Review Host catalog credential…"); err != nil {
		t.Fatal(err)
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, value := range []string{"其他已知任务 ID", "包括暂停任务", "rotate", "revoke", "不撤销 Host", "已认证 Host 连接和目录账号"} {
		if !strings.Contains(text, value) {
			t.Fatal("missing shared credential consequence", value, text)
		}
	}
	_ = tt.Click("确认获取/生成 Host 目录凭据 / Confirm Host catalog credential")
	if obtains != 0 {
		t.Fatal("missing independent acknowledgement")
	}
	if err := tt.Click(routineTriggerAcknowledgement); err != nil {
		t.Fatal(err)
	}
	if obtains != 0 {
		t.Fatal("checkbox obtained credential")
	}
	if err := tt.Click("确认获取/生成 Host 目录凭据 / Confirm Host catalog credential"); err != nil {
		t.Fatal(err)
	}
	if obtains != 1 || len(copies) != 0 || !tt.HasText("X-Dshbot-Token: synthetic-ui-credential") {
		t.Fatal("explicit obtain missing or auto copied")
	}
	if err := tt.Click("复制触发地址 / Copy trigger address"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("复制敏感鉴权头 / Copy secret header"); err != nil {
		t.Fatal(err)
	}
	if len(copies) != 2 || strings.Contains(copies[0], "credential") || copies[1] != "X-Dshbot-Token: synthetic-ui-credential" {
		t.Fatal("copy scope incorrect")
	}
	if err := tt.Click("关闭并清除本页凭据 / Close and clear credential"); err != nil {
		t.Fatal(err)
	}
	if cstore.Snapshot(k).Trigger.Open || tt.HasText("X-Dshbot-Token: synthetic-ui-credential") {
		t.Fatal("close did not clear display")
	}
	if obtains != 1 || len(copies) != 2 {
		t.Fatal("close retried or unexpectedly changed explicit clipboard copy")
	}
}
