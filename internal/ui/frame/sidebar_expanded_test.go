package frame

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

// TestSidebarStaysExpandedWithoutAWorkspace is the Pass 2 shell fix: a desktop
// window keeps the 280-DIP navigation column — brand, new session, the three
// pages, the workspace section and the settings/account foot — whether or not a
// workspace is configured. Only the phone drawer band below 768 DIP leaves the
// grid.
func TestSidebarStaysExpandedWithoutAWorkspace(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	sizes := DefaultLayout()
	sizes.Bottom, sizes.Right = 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Store: session.New(), Sizes: &sizes, OnOpenSettings: func() {}})
	}, 960, 613)

	if sizes.Sidebar != SidebarDefault {
		t.Fatalf("sidebar preference = %v DIP, want the %v default at 960 DIP", sizes.Sidebar, SidebarDefault)
	}
	divider, ok := tt.Find("Sidebar divider")
	if !ok || divider.X != SidebarDefault {
		t.Fatalf("sidebar divider = %+v, want a %v-DIP track", divider, SidebarDefault)
	}
	// The brand block: wordmark and subtitle, plus the collapse control at its
	// top right (the rail's expand control must not be built).
	for _, text := range []string{"WHALE ISLE", "BASED ON DEEPSEEK HARNESS"} {
		if !tt.HasText(text) {
			t.Fatalf("expanded sidebar lost the brand %q; texts=%q", text, tt.Texts())
		}
	}
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarCollapse)); !ok {
		t.Fatalf("expanded sidebar has no collapse control; texts=%q", tt.Texts())
	}
	if _, ok := tt.Find(i18n.T(i18n.KeySidebarExpand)); ok {
		t.Fatal("expanded sidebar still shows the rail's expand control")
	}
	// The three page rows. Find reports the text's box, so the row height is the
	// pitch between consecutive rows: the reference's 36 DIP (rows are 36 high
	// and butt together; a 4-DIP gap used to make the pitch 40).
	rows := make([]ui.Rect, 0, 3)
	for _, row := range []string{
		i18n.T(i18n.KeySidebarMarket),
		i18n.T(i18n.KeySidebarSchedule),
		i18n.T(i18n.KeySidebarAssistant),
	} {
		r, ok := tt.Find(row)
		if !ok {
			t.Fatalf("expanded sidebar lost the %q row; texts=%q", row, tt.Texts())
		}
		if r.X < 13.5 || r.X+r.W > SidebarDefault-13.5+0.5 {
			t.Fatalf("%q row %+v is outside the 14-DIP sidebar insets", row, r)
		}
		rows = append(rows, r)
	}
	for i := 1; i < len(rows); i++ {
		if pitch := rows[i].Y - rows[i-1].Y; pitch < 35.5 || pitch > 36.5 {
			t.Fatalf("nav row pitch = %.1f DIP, want the reference's 36 (rows %v)", pitch, rows)
		}
	}
	if !tt.HasText(i18n.T(i18n.KeySidebarWorkspaceSection)) {
		t.Fatalf("expanded sidebar lost the workspace section; texts=%q", tt.Texts())
	}
	// The foot is the single settings/account seat inside the window.
	seat, ok := tt.Find(i18n.T(i18n.KeySettingsAccountTitle))
	if !ok || seat.Y+seat.H > 613 {
		t.Fatalf("settings foot = %+v, want it inside the 613-DIP window", seat)
	}
}

// TestSidebarKeepsTheTrackAtDesktopWidths pins the band boundaries: the column
// stays a track at 768 DIP and above, and only narrower windows hand the
// navigation to the phone drawer.
func TestSidebarKeepsTheTrackAtDesktopWidths(t *testing.T) {
	for _, tc := range []struct {
		width     int
		wantTrack bool
	}{
		{768, true},
		{900, true},
		{960, true},
		{1280, true},
		{767, false},
		{700, false},
	} {
		sizes := DefaultLayout()
		sizes.ResolveColumns(float32(tc.width), false)
		track := !sizes.drawerMode() && sizes.sidebarPreference() > 0
		if track != tc.wantTrack {
			t.Fatalf("width %d: track=%v drawer=%v, want track=%v", tc.width, track, sizes.drawerMode(), tc.wantTrack)
		}
	}
}
