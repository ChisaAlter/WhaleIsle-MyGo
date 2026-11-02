package frame

import (
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/shortcuts"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestSidebarNavigationUsesLiveCommandBinding(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previous) })
	sizes := DefaultLayout()
	created := 0
	m := Model{Store: session.New(), Sizes: &sizes, ThemeID: "deepseek.light", OnCreateSession: func() { created++ }}
	tt := ui.NewTester(func(c *ui.Context) {
		registerPageCommands(c, m)
		ui.Column(c).Width(280).FillHeight().Children(func() { navigationView(c, m, m.Store.Snapshot()) })
	}, 1200, 800)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	label := i18n.T(i18n.KeySidebarNewSessionFallback)
	r, ok := tt.Find(label)
	if !ok {
		t.Fatal("missing new session button")
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	display := func() string {
		value := shortcuts.Resolve(m.PageShortcutBindings)["session.new"].String()
		if runtime.GOOS == "windows" {
			return strings.Replace(value, "Mod", "Ctrl", 1)
		}
		if runtime.GOOS == "darwin" {
			return strings.Replace(value, "Mod", "Cmd", 1)
		}
		return value
	}
	initialHint := display()
	if !tt.HasText(initialHint) {
		t.Fatalf("default hint not bound: %v", tt.Texts())
	}
	m.PageShortcutBindings = map[string]string{"session.new": "Mod+Alt+F8"}
	tt.Frame()
	customHint := display()
	if !tt.HasText(customHint) || tt.HasText(initialHint) {
		t.Fatal("new binding not reflected")
	}
	pressFramePhysicalKey(tt, ui.Cmd, ui.KeyN, "KeyN")
	if created != 0 {
		t.Fatal("old command binding remained live")
	}
	pressFramePhysicalKey(tt, ui.Cmd|ui.Alt, ui.KeyF8, "F8")
	if created != 1 {
		t.Fatalf("new command calls=%d", created)
	}
	m.PageShortcutBindings["session.new"] = ""
	tt.Frame()
	if tt.HasText(customHint) {
		t.Fatal("disabled binding left stale hint")
	}
	pressFramePhysicalKey(tt, ui.Cmd|ui.Alt, ui.KeyF8, "F8")
	if created != 1 {
		t.Fatal("disabled command still dispatched")
	}
	m.PageShortcutBindings["session.new"] = "Mod+B" // Conflicts with the existing sidebar binding.
	tt.Frame()
	if got := display(); got != "" || tt.HasText("Ctrl+B") {
		t.Fatal("conflicting command advertised")
	}
}

func TestSidebarNavigationSelectionComesFromFrameRoute(t *testing.T) {
	sizes := DefaultLayout()
	m := Model{Store: session.New(), Sizes: &sizes, ThemeID: "deepseek.light"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(280).FillHeight().Children(func() { navigationView(c, m, m.Store.Snapshot()) })
	}, 1200, 800)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	label := i18n.T(i18n.KeySidebarSchedule)
	r, ok := tt.Find(label)
	if !ok {
		t.Fatal("schedule navigation missing")
	}
	idle := tt.Image().RGBAAt(258, int(r.Y+11))
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	hover := tt.Image().RGBAAt(258, int(r.Y+11))
	if hover == idle {
		t.Fatal("hover affordance absent")
	}
	tt.Move(1199, 700)
	m.ActivePanel = "schedule"
	tt.Frame()
	if got := tt.Image().RGBAAt(258, int(r.Y+11)); got != hover {
		t.Fatalf("route selected paint=%v hover=%v", got, hover)
	}
	m.ActivePanel = ""
	tt.Frame()
	if got := tt.Image().RGBAAt(258, int(r.Y+11)); got != idle {
		t.Fatal("conversation retained old panel selection")
	}
}
