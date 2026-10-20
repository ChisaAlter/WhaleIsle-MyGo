package widgets

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// contrastCase is a pair of colors a primitive paints together, and the
// contrast the web's rules ask of it.
type contrastCase struct {
	what   string
	fg, bg ui.Color
	want   float64
}

func TestPrimitiveColorsMeetContrast(t *testing.T) {
	for _, theme := range []*ui.Theme{ui.LightTheme(), ui.DarkTheme()} {
		name := "light"
		if theme.Dark {
			name = "dark"
		}
		// The hover fill a disclosure row takes, over the page.
		hover := theme.SurfaceHover.Alpha(0.5).Over(theme.Background)
		hoverInk := theme.TextMuted.Mix(theme.Text, 0.35)
		track := theme.Border.Mix(theme.TextMuted, 0.7)
		cases := []contrastCase{
			{"disclosure title", theme.TextMuted, theme.Background, TextContrast},
			{"disclosure title on hover", hoverInk, hover, TextContrast},
			{"disclosure summary", theme.TextMuted, theme.Background, TextContrast},
			{"shimmer highlight", shimmerHighlight(theme, theme.TextMuted), theme.Background, TextContrast},
			{"setting title", theme.Text, theme.Background, TextContrast},
			{"setting description", theme.TextMuted, theme.Background, TextContrast},
			{"dialog title", theme.Text, theme.Background, TextContrast},
			{"dialog message", theme.TextMuted, theme.Background, TextContrast},
			{"dialog confirm on danger", readableOn(theme.Danger), theme.Danger, TextContrast},
			{"dialog confirm on accent", readableOn(theme.Accent), theme.Accent, TextContrast},
			{"dialog cancel", theme.Text, theme.Surface, TextContrast},
			{"switch off track", track, theme.Background, ControlContrast},
			{"switch on track", theme.Accent, theme.Background, ControlContrast},
			{"switch thumb on off track", ui.RGB(255, 255, 255), track, ControlContrast},
			{"switch thumb on on track", ui.RGB(255, 255, 255), theme.Accent, ControlContrast},
		}
		for _, state := range []State{StateIdle, StateRunning, StateDone, StateWarning, StateError} {
			cases = append(cases, contrastCase{
				"state dot " + state.String(),
				StateColor(theme, state).Over(theme.Background),
				theme.Background,
				ControlContrast,
			})
		}
		for _, c := range cases {
			if got := contrastRatio(c.fg, c.bg); got < c.want {
				t.Errorf("%s %s: contrast %.2f, want at least %.1f", name, c.what, got, c.want)
			}
		}
	}
}

func TestReadableOnPicksTheBetterInk(t *testing.T) {
	for _, fill := range []ui.Color{
		ui.RGB(255, 255, 255), ui.RGB(0, 0, 0), ui.Hex("#dc2626"),
		ui.Hex("#3b82f6"), ui.Hex("#facc15"), ui.Hex("#808080"),
	} {
		ink := readableOn(fill)
		if got := contrastRatio(ink, fill); got < TextContrast {
			t.Errorf("ink on %v reaches %.2f, want at least %.1f", fill, got, TextContrast)
		}
	}
}
