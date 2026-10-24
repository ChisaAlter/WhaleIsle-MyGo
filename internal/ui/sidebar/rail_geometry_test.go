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
// Native's caption-band 168.67-wide 新建会话 is an inert duplicate in both rail
// and expanded states (round259/261), so it is deliberately not reproduced here.
func TestCollapsedRailMatchesNativeGeometry(t *testing.T) {
	m := Model{Collapsed: true, OnToggle: func() {}, OnCreate: func() {},
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
		if math.Abs(float64(got-want)) > 1 {
			t.Fatalf("%s = %.2f, want %.2f ±1", name, got, want)
		}
	}
	expand := row(i18n.T(i18n.KeySidebarExpand))
	near("expand x", expand.X, 10)
	near("expand y", expand.Y, 18)
	near("expand size", expand.W, 36)
	near("expand height", expand.H, 36)

	created := 0
	m.OnCreate = func() { created++ }
	tt.Frame()
	if err := tt.Click(i18n.T(i18n.KeySidebarNewSessionFallback)); err != nil {
		t.Fatal(err)
	}
	if created != 1 {
		t.Fatalf("rail 新建会话 created=%d sessions, want 1", created)
	}

	for _, tc := range []struct {
		key  string
		want float32
	}{
		{i18n.KeySidebarMarket, 114},
		{i18n.KeySidebarSchedule, 162},
		{i18n.KeySidebarAssistant, 210},
	} {
		r := row(i18n.T(tc.key))
		near(tc.key+" x", r.X, 10)
		near(tc.key+" y", r.Y, tc.want)
		near(tc.key+" size", r.W, 36)
		near(tc.key+" pitch-safe height", r.H, 36)
	}
}
