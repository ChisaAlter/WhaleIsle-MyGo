package sidebar

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestCollapsedRailMatchesNativeGeometry pins the collapsed rail to the native
// 56-DIP column measured in round261 (UIA, 150%): every control is a 36x36 seat
// at x=10, the expand toggle sits at y=18, 新建会话 at y=66, and the 全局面板
// group runs 插件 y=114, 自动化任务 y=162, 鲸鱼娘 y=210 — a 48-DIP pitch.
//
// Baseline truth is the reference stylesheet, not the tester: SidebarRoot.module.css
// `.root.collapsed { padding: 18px 10px 6px }` ("18px from the rail top to the
// whale's box"), `.collapsed .newSession { margin: 0 0 12px }` and
// `.collapsed .panelList { gap: 12px }`. With that top padding the DIP stack lands
// on all six native seats with zero error, so ±0.1 is a real pin.
//
// The live window measured the whole column +0.67 low in round261. That offset is
// not rail padding: it is one 1/1.5-DIP step at the window origin, the same
// quantization native's own rects show (native 账号菜单 x=9.33, w=36.67;
// 全局面板 Group w=35.33). Rounding it into the padding (17.33, round262) is only
// right at 150% and sits the rail 1 DIP high at 100%, so the rail keeps the
// reference constant and the window-origin delta stays a separate item.
//
// Native's caption-band 168.67-wide 新建会话 is an inert duplicate in both rail
// and expanded states (round259/261), so it is deliberately not reproduced here.
func TestCollapsedRailMatchesNativeGeometry(t *testing.T) {
	m := Model{Collapsed: true, OnToggle: func() {}, OnCreate: func() {}, OnOpenAssistant: func() {},
		Snapshot: session.ViewState{WorkspaceReady: true}}
	tt := ui.NewTester(func(c *ui.Context) { View(c, m) }, 72, 900)
	row := func(name string) ui.Rect {
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("rail control %q missing; texts=%v", name, tt.Texts())
		}
		return r
	}
	near := func(name string, got, want float32) {
		// 0.1, not 1: ±1 accepted both 18 and 17.33, so it could not tell the
		// native rail from a uniformly shifted one.
		if math.Abs(float64(got-want)) > 0.1 {
			t.Fatalf("%s = %.2f, want %.2f ±0.1", name, got, want)
		}
	}
	seat := func(t *testing.T, key string, wantY float32) {
		t.Helper()
		r := row(i18n.T(key))
		near(key+" x", r.X, 10)
		near(key+" y", r.Y, wantY)
		near(key+" size", r.W, 36)
		near(key+" pitch-safe height", r.H, 36)
	}

	seat(t, i18n.KeySidebarExpand, 18)
	seat(t, i18n.KeySidebarNewSessionFallback, 66)

	created := 0
	m.OnCreate = func() { created++ }
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeySidebarNewSessionFallback)); err != nil {
		t.Fatal(err)
	}
	if created != 1 {
		t.Fatalf("rail 新建会话 created=%d sessions, want 1", created)
	}

	seat(t, i18n.KeySidebarMarket, 114)
	seat(t, i18n.KeySidebarSchedule, 162)
	seat(t, i18n.KeySidebarAssistant, 210)
}
