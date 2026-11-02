package frame

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/ui/i18n"
)

// TestCaptionButtonsSitInTheNativePlateRow locks the vertical seat of the
// window controls to the injected caption plate, which is the only native
// statement of the row:
//
//	#dshd-shell-controls { height: 48; padding: 12px 8px 4px; align-items: center }
//	#dshd-shell-controls button { width/height: 32; border-radius: 8 }
//	  reference/electron/src/main/harness-chrome-inject.js:11,88-89,96-104
//
// so each control is 32x32 at y=12..44 and the band keeps its 4-DIP bottom
// inset. Drawing at y=8 (the pre-round133 reading) floated the trio 4 DIP above
// the native row *and* above the Win32 overlay, which main.go already creates
// with Top 12 (cmd/whaleisle/main.go WindowsCaptionButtons).
func TestCaptionButtonsSitInTheNativePlateRow(t *testing.T) {
	i18n.SetLocale(i18n.LocaleZH)
	defer i18n.SetLocale(i18n.DefaultLocale)

	const (
		nativePlateTop    = 12
		nativePlateBottom = 4
		nativeButton      = 32
	)
	if captionButtonTop != nativePlateTop || captionButtonSize != nativeButton {
		t.Fatalf("caption button seat = top %v size %v, want the native top %d size %d", captionButtonTop, captionButtonSize, nativePlateTop, nativeButton)
	}
	if bottom := float32(captionButtonTop + captionButtonSize); bottom != workspaceCaptionHeight-nativePlateBottom {
		t.Fatalf("caption button bottom = %.2f, want the plate's 48-4 = %.2f", bottom, workspaceCaptionHeight-nativePlateBottom)
	}
	// The painted trio and the shared caption's content row are the same 32-DIP
	// row: 48 band, 12 top inset, 4 bottom inset.
	if workspaceCaptionInsetTop != nativePlateTop || workspaceCaptionContent != nativeButton {
		t.Fatalf("shared caption content = inset %.2f height %.2f, want %d/%d", workspaceCaptionInsetTop, workspaceCaptionContent, nativePlateTop, nativeButton)
	}

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			drawCaptionButtons(c, ui.RGB(40, 40, 40), captionWindowActions{
				Maximized: func() bool { return false },
			})
		})
	}, 400, 200)
	tt.Frame()
	for name, right := range map[string]float32{"关闭": 8, "最大化": 40, "最小化": 72} {
		r, ok := tt.Find(name)
		if !ok {
			t.Fatalf("caption button %q is missing; texts = %q", name, tt.Texts())
		}
		if math.Abs(float64(r.Y-nativePlateTop)) > 0.01 || math.Abs(float64(r.H-nativeButton)) > 0.01 {
			t.Fatalf("%s y=%.2f h=%.2f, want the native y=%d h=%d", name, r.Y, r.H, nativePlateTop, nativeButton)
		}
		if math.Abs(float64(400-right-(r.X+r.W))) > 0.01 || math.Abs(float64(r.W-nativeButton)) > 0.01 {
			t.Fatalf("%s x=%.2f w=%.2f, want a %d-wide box ending %v from the right edge", name, r.X, r.W, nativeButton, right)
		}
	}

	// The hit box is the painted box: a click at the native close centre still
	// reaches the control after the 4-DIP move.
	closed := 0
	tt2 := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			drawCaptionButtons(c, ui.RGB(40, 40, 40), captionWindowActions{Close: func() { closed++ }})
		})
	}, 400, 200)
	tt2.Frame()
	hit := image.Rect(400-8-32, nativePlateTop, 400-8, nativePlateTop+32)
	tt2.ClickAt(float32(hit.Min.X+16), float32(hit.Min.Y+16))
	tt2.Frame()
	if closed != 1 {
		t.Fatalf("click at the native close seat (%v) ran the close %d times, want 1", hit, closed)
	}
}
