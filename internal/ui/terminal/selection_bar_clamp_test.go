package terminal

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/chrome"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestTerminalSelectionActionBarClampedAt280DIP(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	for _, tc := range []struct {
		name   string
		winW   int
		winH   int
		bounds ui.Rect
	}{
		{
			name:   "narrow-280-full",
			winW:   280,
			winH:   200,
			bounds: ui.Rect{X: 0, Y: 0, W: 280, H: 200},
		},
		{
			name:   "narrow-280-split-right",
			winW:   420,
			winH:   240,
			bounds: ui.Rect{X: 140, Y: 20, W: 280, H: 200},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &Panel{}
			selection := "see https://example.com/docs/path for details"
			fragment := FormatSelection(selection)
			m := Model{
				Panel:        p,
				ThemeID:      "deepseek",
				CanAddToChat: true,
				OnAddToChat:  func(string) error { return nil },
				OnOpenTarget: func(string) error { return nil },
			}
			key := captureSelectionKey(m, selection)
			colors := theme.Resolve(m.ThemeID).Active(false)
			border := chrome.Color(colors.Border)

			tt := ui.NewTester(func(c *ui.Context) {
				paintSelectionActionBar(c, m, p, nil, colors, border, tc.bounds, selection, fragment, key)
			}, tc.winW, tc.winH)

			barRect, ok := tt.Find("Terminal selection actions")
			if !ok {
				t.Fatalf("missing selection action bar; texts=%q", tt.Texts())
			}
			maxRight := min(tc.bounds.X+tc.bounds.W, float32(tc.winW))
			if barRect.X < tc.bounds.X {
				t.Fatalf("selection bar overflowed left: bar=%+v bounds=%+v", barRect, tc.bounds)
			}
			if barRect.X+barRect.W > maxRight+0.5 {
				t.Fatalf("selection bar overflowed right at %d DIP: bar=%+v maxRight=%.1f", tc.winW, barRect, maxRight)
			}
			if barRect.Y < 0 || barRect.Y+barRect.H > float32(tc.winH)+0.5 {
				t.Fatalf("selection bar overflowed vertically: bar=%+v winH=%d", barRect, tc.winH)
			}
			for _, label := range []string{"Copy", "Add to chat", "Open link in system browser"} {
				r, found := tt.Find(label)
				if !found {
					t.Fatalf("missing button %q; texts=%q", label, tt.Texts())
				}
				if r.X < tc.bounds.X || r.X+r.W > maxRight+0.5 {
					t.Fatalf("button %q clipped outside bounds: rect=%+v bounds=%+v maxRight=%.1f", label, r, tc.bounds, maxRight)
				}
			}
		})
	}
}
