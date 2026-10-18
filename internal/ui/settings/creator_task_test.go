package settings

import (
	"github.com/egoist/mygo/ui"
	"strings"
	"testing"
	"whaleisle/internal/store/session"
)

func TestCreatorTaskEntryUIFixture(t *testing.T) {
	prepared, confirmed := 0, 0
	state := session.CreatorTaskState{}
	rows := session.AgentPresetManagerState{Loaded: true, Rows: []session.ManagedAgentPreset{{ID: "cordis"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		creatorTaskPane(c, &CreatorTaskAccess{State: state, OnPrepare: func() {
			prepared++
			state = session.CreatorTaskState{Ticket: 7, Review: true, SessionID: "session-fixture", Host: "https://fixture.invalid", Cwd: "/fixture"}
		}, OnCancel: func(uint64) { state.Review = false }, OnCreate: func(ticket uint64) {
			if ticket != 7 {
				t.Fatal(ticket)
			}
			confirmed++
		}}, rows, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150))
	}, 1100, 1200)
	if prepared != 0 || confirmed != 0 {
		t.Fatal("render performed action")
	}
	if err := tt.Click("新建 Creator 任务… / New Creator task…"); err != nil {
		t.Fatal(err)
	}
	if prepared != 1 || confirmed != 0 {
		t.Fatal("review dispatched mutation")
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{"https://fixture.invalid", "session-fixture", "/fixture", "cordis", "不会发送", "默认"} {
		if !strings.Contains(text, want) {
			t.Fatal("missing disclosure", want)
		}
	}
	if err := tt.Click("取消 / Cancel Creator"); err != nil {
		t.Fatal(err)
	}
	if confirmed != 0 {
		t.Fatal("cancel submitted")
	}
	if err := tt.Click("新建 Creator 任务… / New Creator task…"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("确认创建 Creator 任务 / Create Creator task"); err != nil {
		t.Fatal(err)
	}
	if confirmed != 1 {
		t.Fatal("confirm not wired")
	}
	state = session.CreatorTaskState{Attempted: true, SessionID: "session-retained", Error: "unknown outcome"}
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "session-retained") {
		t.Fatal("lost recoverable result")
	}
}
