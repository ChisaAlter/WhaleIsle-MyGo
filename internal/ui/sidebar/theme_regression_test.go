package sidebar

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestSidebarBackgroundUsesThemeRole(t *testing.T) {
	for _, id := range []string{"deepseek.light", "deepseek.dark"} {
		t.Run(id, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) { View(c, Model{ThemeID: id}) }, 280, 800)
			colors := theme.Resolve(id).Active(false)
			want := chrome.Color(colors.Sidebar)
			got := tt.Image().RGBAAt(275, 790)
			if got.R != want.R || got.G != want.G || got.B != want.B {
				t.Fatalf("sidebar background=%+v, want theme.Sidebar=%+v", got, want)
			}
		})
	}
}

func TestSelectedSessionPreservesProvidedThemeLabelColors(t *testing.T) {
	fg, muted := ui.RGB(232, 12, 18), ui.RGB(12, 220, 20)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			rowSession(c, session.Item{SessionID: "a", Title: "CUSTOM TITLE", Cwd: "CUSTOM CWD"}, 0, Model{ThemeID: "deepseek.light", Snapshot: session.ViewState{SelectedID: "a"}}, fg, muted, ui.Transparent)
		})
	}, 280, 100)
	for _, tc := range []struct {
		label string
		color ui.Color
	}{{"CUSTOM TITLE", fg}, {"CUSTOM CWD", muted}} {
		r, ok := tt.Find(tc.label)
		if !ok {
			t.Fatalf("missing %q", tc.label)
		}
		found := false
		for y := int(r.Y); y < int(r.Y+r.H); y++ {
			for x := int(r.X); x < int(r.X+r.W); x++ {
				p := tt.Image().RGBAAt(x, y)
				if p.R == tc.color.R && p.G == tc.color.G && p.B == tc.color.B {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("selected %q did not retain provided theme color %+v", tc.label, tc.color)
		}
	}
}
