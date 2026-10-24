package welcome

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// TestWelcomeLayoutMetrics asserts the reference grid's metrics on a rendered
// 600x700 welcome window (welcome-layout.css: #root gap 8, .titlebar 36,
// .welcome 40/1fr/104 with 24 48 76 padding, .actions 240 wide with 16 gaps of
// 44px buttons, .actions .back 28 tall 16 below them).
func TestWelcomeLayoutMetrics(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{ThemeID: "deepseek", Page: PageSignIn})
	}, 600, 700)
	tagline, ok := tt.Find(i18n.T(i18n.KeyWelcomeTagline))
	if !ok {
		t.Fatalf("tagline missing; texts=%q", tt.Texts())
	}
	sub, ok := tt.Find(i18n.T(i18n.KeyWelcomeDescription))
	if !ok {
		t.Fatalf("description missing; texts=%q", tt.Texts())
	}
	// .tagline (and its p): line-height 40.
	if tagline.H != 40 || sub.H != 40 {
		t.Fatalf("tagline line boxes = %.1f/%.1f, want 40/40", tagline.H, sub.H)
	}
	signIn, ok := tt.Find(i18n.T(i18n.KeyWelcomeSignIn))
	if !ok {
		t.Fatalf("sign-in button missing; texts=%q", tt.Texts())
	}
	keyBtn, ok := tt.Find(i18n.T(i18n.KeyWelcomeAPIKey))
	if !ok {
		t.Fatalf("api key button missing; texts=%q", tt.Texts())
	}
	// .actions gap 16 between 44px buttons: label centres sit 60 apart.
	if pitch := keyBtn.Y - signIn.Y; pitch < 59 || pitch > 61 {
		t.Fatalf("button pitch = %.1f, want 44 + 16", pitch)
	}
	skip, ok := tt.Find(i18n.T(i18n.KeyWelcomeSkip))
	if !ok {
		t.Fatalf("skip action missing; texts=%q", tt.Texts())
	}
	// .actions .back top calc(100% + 16px), height 28: centre 22 + 16 + 14 below
	// the last button's centre. The rendered label box adds ~3 DIP of slack, so
	// the range allows the intrinsic text height rather than pretending to exact.
	if gap := skip.Y - keyBtn.Y; gap < 50 || gap > 57 {
		t.Fatalf("skip gap = %.1f, want 44/2 + 16 + 28/2", gap)
	}
}

// TestWelcomeKeyFormMetrics asserts the key page's type and form metrics:
// .key-form 440 wide with 40 gaps, .key-heading h1 20/32, p 16/22.
func TestWelcomeKeyFormMetrics(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	key := ""
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{ThemeID: "deepseek", Page: PageKey, APIKey: &key})
	}, 600, 700)
	title, ok := tt.Find(i18n.T(i18n.KeyWelcomeKeyTitle))
	if !ok {
		t.Fatalf("key heading missing; texts=%q", tt.Texts())
	}
	if title.H != 32 {
		t.Fatalf("key heading line box = %.1f, want 32", title.H)
	}
	desc, ok := tt.Find(i18n.T(i18n.KeyWelcomeKeyDescription))
	if !ok {
		t.Fatalf("key description missing; texts=%q", tt.Texts())
	}
	if desc.H != 22 {
		t.Fatalf("key description line box = %.1f, want 22", desc.H)
	}
	if _, ok := tt.Find(i18n.T(i18n.KeyWelcomeKeySave)); !ok {
		t.Fatalf("save action missing; texts=%q", tt.Texts())
	}
}

// TestWelcomeMetricConstants pins the ported CSS numbers themselves, so a later
// edit cannot drift the layout away from welcome-layout.css silently.
func TestWelcomeMetricConstants(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  int
		want int
	}{
		{"titlebar", titleBarHeight, 36},
		{"brand", brandHeight, 40},
		{"grid top padding", gridPadTop, 24},
		{"grid side padding", gridPadSide, 48},
		{"grid bottom padding", gridPadBottom, 76},
		{"actions height", actionsHeight, 104},
		{"actions width", actionsWidth, 240},
		{"actions gap", actionsGap, 16},
		{"button height", buttonHeight, 44},
		{"button radius", buttonRadius, 10},
		{"back height", backHeight, 28},
		{"back min width", backMinWidth, 58},
		{"back offset", backTop, 16},
		{"key form width", keyFormWidth, 440},
		{"key form gap", keyFormGap, 40},
		{"key heading gap", keyHeadGap, 8},
		{"input height", inputHeight, 48},
		{"input radius", inputRadius, 12},
	} {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d (welcome-layout.css)", tc.name, tc.got, tc.want)
		}
	}
}
