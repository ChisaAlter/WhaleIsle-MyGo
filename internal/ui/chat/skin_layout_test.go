package chat

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/theme"
)

func TestSkinUserBubbleUsesInstalledFourCornerRadiiAndSurface(t *testing.T) {
	old := theme.CurrentPreferences()
	t.Cleanup(func() { theme.SetPreferences(old) })
	p := theme.Defaults()
	p.Light, p.Dark = "codex", "claude"
	theme.SetPreferences(p)
	for _, dark := range []bool{false, true} {
		colors := theme.ResolveWithSystem(theme.IDSystem, dark).Active(dark)
		body := "An installed skin bubble"
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).FillWidth().Padding(4).Background(ui.RGB(0, 0, 0)).Children(func() {
				userMessageView(c, Model{ThemeID: theme.IDSystem, SystemDark: dark, ContentWidth: 720}, presentationMessage(body), colors, nil, body, nil)
			})
		}, 800, 100)
		r, ok := tt.Find("User message bubble")
		if !ok {
			t.Fatal("bubble missing")
		}
		im := tt.Image()
		fill := chrome.Color(colors.Bubble)
		center := im.RGBAAt(int(r.X+r.W/2), int(r.Y+3))
		if center.R != fill.R || center.G != fill.G || center.B != fill.B {
			t.Fatalf("surface=%v want%v", center, fill)
		}
		bottomRight := im.RGBAAt(int(r.X+r.W-2), int(r.Y+r.H-2))
		bottomLeft := im.RGBAAt(int(r.X+1), int(r.Y+r.H-2))
		if !dark && (bottomRight == bottomLeft || bottomRight == im.RGBAAt(int(r.X), int(r.Y))) {
			t.Fatalf("codex4bottomright vs18bottomleft not distinct r=%+v right%v left%v", r, bottomRight, bottomLeft)
		}
		if dark && (bottomRight.R != 0 || bottomLeft.R != 0) {
			t.Fatalf("claude12corners are not clipped right%v left%v", bottomRight, bottomLeft)
		}
	}
}
