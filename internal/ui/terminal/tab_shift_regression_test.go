package terminal

import (
	"testing"

	"whaleisle/internal/ui/i18n"
)

func TestTerminalTabHoverZeroLayoutShift(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })

	p := paneStripPanel(2)
	tt := renderPaneStrip(t, p)
	t1Before, ok1 := tt.Find("Terminal 1")
	t2Before, ok2 := tt.Find("Terminal 2")
	if !ok1 || !ok2 {
		t.Fatalf("missing terminal tab labels: %v", tt.Texts())
	}
	tt.Move(t1Before.X+t1Before.W/2, t1Before.Y+t1Before.H/2)
	tt.Frame()
	t1After, _ := tt.Find("Terminal 1")
	t2After, _ := tt.Find("Terminal 2")
	if t1After != t1Before || t2After != t2Before {
		t.Fatalf("hovering inactive terminal tab shifted layout: t1 %+v -> %+v, t2 %+v -> %+v",
			t1Before, t1After, t2Before, t2After)
	}
}
