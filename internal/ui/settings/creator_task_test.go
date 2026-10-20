package settings

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
)

func TestCreatorTaskEntryUIFixture(t *testing.T) {
	started, extraConfirm, located, opened := 0, 0, 0, 0
	state := session.CreatorTaskState{}
	rows := session.AgentPresetManagerState{Loaded: true, Rows: []session.ManagedAgentPreset{{ID: "cordis"}}}
	tt := ui.NewTester(func(c *ui.Context) {
		creatorTaskPane(c, &CreatorTaskAccess{State: state, OnPrepare: func() {
			started++
			// The adapter admits Prepare+Create from this same user gesture.
			state = session.CreatorTaskState{Ticket: 7, Attempted: true, Busy: true, SessionID: "session-fixture", Host: "https://fixture.invalid", Cwd: "/fixture", Phase: "creating"}
		}, OnCreate: func(uint64) { extraConfirm++ }, OnLocate: func(ticket uint64) {
			if ticket != 7 {
				t.Fatal(ticket)
			}
			located++
		}, OnOpen: func(ticket uint64) {
			if ticket != 7 {
				t.Fatal(ticket)
			}
			opened++
		}}, rows, ui.RGB(255, 255, 255), ui.RGB(150, 150, 150))
	}, 1100, 1200)
	if started != 0 || extraConfirm != 0 {
		t.Fatal("render performed action")
	}
	if err := tt.Click("新建 Creator 任务"); err != nil {
		t.Fatal(err)
	}
	if started != 1 || extraConfirm != 0 {
		t.Fatal("one click did not start the task directly")
	}
	for _, old := range []string{"确认创建 Creator 任务", "取消"} {
		if _, ok := tt.Find(old); ok {
			t.Fatal("extra review step remains", old)
		}
	}
	text := strings.Join(tt.Texts(), "\n")
	for _, want := range []string{"https://fixture.invalid", "session-fixture", "/fixture", "cordis", "不会发送", "默认"} {
		if !strings.Contains(text, want) {
			t.Fatal("missing disclosure", want)
		}
	}
	if err := tt.Click("定位已提交目标"); err != nil {
		t.Fatal(err)
	}
	if located != 0 || started != 1 {
		t.Fatal("busy result allowed another operation")
	}
	state.Busy, state.Error = false, "unknown outcome"
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), "\n"), "session-fixture") {
		t.Fatal("lost recoverable result")
	}
	if err := tt.Click("定位已提交目标"); err != nil {
		t.Fatal(err)
	}
	if located != 1 || started != 1 {
		t.Fatal("locate recreated the task")
	}
	state.Created = true
	tt.Frame()
	if err := tt.Click("打开已有任务"); err != nil {
		t.Fatal(err)
	}
	if opened != 1 || started != 1 || extraConfirm != 0 {
		t.Fatal("retained target recovery changed admission")
	}
}

func TestCreatorTaskEntryUnavailableDoesNotStart(t *testing.T) {
	for _, mode := range []string{"loading", "missing", "broken", "hidden", "busy"} {
		t.Run(mode, func(t *testing.T) {
			rows := session.AgentPresetManagerState{Loaded: true, Rows: []session.ManagedAgentPreset{{ID: "cordis"}}}
			switch mode {
			case "loading":
				rows.Loaded = false
			case "missing":
				rows.Rows = nil
			case "broken":
				rows.Rows[0].Broken = "missing package"
			case "hidden":
				rows.Rows[0].Hidden = true
			case "busy":
				rows.Busy = true
			}
			started := 0
			tt := ui.NewTester(func(c *ui.Context) {
				creatorTaskPane(c, &CreatorTaskAccess{OnPrepare: func() { started++ }}, rows, c.Theme().Text, c.Theme().Text)
			}, 900, 700)
			if err := tt.Click("新建 Creator 任务"); err != nil {
				t.Fatal(err)
			}
			if started != 0 {
				t.Fatal("unavailable Creator started")
			}
		})
	}
}
