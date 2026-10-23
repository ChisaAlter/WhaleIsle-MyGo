package chat

import (
	"bytes"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestContextForeignFieldsKeepProducerAndToolVisibility(t *testing.T) {
	contextBodyEnglish(t)
	msg := contextBodyFixture(`{"kind":"session-reference","references":[{"label":"Prior work"}]}`, `[{"type":"future","text":{"opaque":true},"toolName":[1]},{"type":"tool-addition","toolName":"new_tool"}]`)
	if len(visibleChatMessages([]session.Message{msg})) != 1 {
		t.Fatal("foreign block hid a valid tool change")
	}
	tt := ui.NewTester(func(c *ui.Context) { ContextInjectionView(c, msg, "test", theme.Default().Light) }, 600, 400)
	if _, ok := tt.Find("Session recall Prior work"); !ok {
		t.Fatalf("foreign fields erased source attribution: %q", tt.Texts())
	}
	if err := tt.Click("Session recall Prior work"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Unknown content block") {
		t.Fatal("foreign block lost")
	}
}

func TestTurnTriggerContentClockAndGeometry(t *testing.T) {
	contextBodyEnglish(t)
	before := theme.CurrentPreferences()
	defer theme.SetPreferences(before)
	prefs := theme.Defaults()
	prefs.Chat.Size = 18
	theme.SetPreferences(prefs)
	at := time.Date(2026, 10, 23, 14, 0, 0, 0, time.Local)
	msg := contextBodyFixture(`{"kind":"schedule","form":"snapshot","sections":[{"name":"ignored source","text":"ignored data"}]}`, `[{"type":"text","text":"first\nsecond\nthird"},{"type":"future","text":{"opaque":true}}]`)
	original := append([]byte(nil), msg.Raw...)
	msg.ContextWaking = true
	msg.EventTime = float64(at.AddDate(0, 0, -1).UnixMilli())
	palette := theme.Default().Light
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).FillWidth().Padding(12).Children(func() {
			TurnTriggerView(c, msg, "test", palette)
			ui.Button(c, "after")
		})
	}, 600, 700)
	tt.SetTime(at)
	header, found := tt.Find(i18n.T(i18n.KeyChatTriggerSchedule))
	if !found || header.H >= 51 {
		t.Fatalf("trigger title retained fixed 1.5 line height: %+v", header)
	}
	if !tt.HasText("10/22 14:00") {
		t.Fatalf("historical trigger clock absent: %q", tt.Texts())
	}
	if err := tt.Click(i18n.T(i18n.KeyChatTriggerSchedule)); err != nil {
		t.Fatal(err)
	}
	text, ok := tt.Find("first\nsecond\nthird")
	if !ok || text.H != 54 || text.X != 52.5 {
		t.Fatalf("12/18 trigger content inset/height: %+v found=%v", text, ok)
	}
	if tt.HasText("ignored source") || !tt.HasText("Unknown content block") {
		t.Fatal("trigger used producer form or dropped unknown block")
	}
	expanded, _ := tt.Find("after")
	// Body hover applies the root background even outside the header.
	tt.Move(text.X+1, text.Y+1)
	tt.SetTime(at.Add(110 * time.Millisecond))
	pixel := tt.Image().RGBAAt(20, int(text.Y+4))
	bg := chrome.Color(palette.TriggerHover)
	want := bg.Over(ui.RGB(255, 255, 255))
	if pixel.R != want.R || pixel.G != want.G || pixel.B != want.B {
		t.Fatalf("body hover background=%v want=%v", pixel, want)
	}
	if err := tt.Click(i18n.T(i18n.KeyChatTriggerSchedule)); err != nil {
		t.Fatal(err)
	}
	closed, _ := tt.Find("after")
	if closed.Y >= expanded.Y || tt.HasText("first\nsecond\nthird") {
		t.Fatal("trigger body did not unmount immediately")
	}
	if !bytes.Equal(msg.Raw, original) {
		t.Fatal("retained source changed")
	}
}
