package settings

import (
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

// CloseToTray/AutoStartDesktop are booleans in the config, but the reference
// shows two-option selects. The selected option must follow the stored value.
func TestDesktopBehaviourSelectsFollowTheStoredBooleans(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })

	closeToTray, autoStart := true, false
	tt := ui.NewTester(func(c *ui.Context) {
		d := &DesktopAccess{
			Version:          "fixture",
			Fields:           DesktopFields{CloseToTray: &closeToTray},
			AutoStartDesktop: &autoStart,
			OnSave:           func() {},
		}
		desktopPane(c, Model{Desktop: d}, c.Theme().Text, c.Theme().TextMuted)
	}, 900, 1100)
	if !tt.HasText("Minimize to tray") {
		t.Fatalf("close=true did not select the tray option: %v", tt.Texts())
	}
	if !tt.HasText("Open the launcher") {
		t.Fatalf("autostart=false did not select the launcher option: %v", tt.Texts())
	}

	closeToTray, autoStart = false, true
	tt.Frame()
	if !tt.HasText("Quit") {
		t.Fatalf("close=false did not select the quit option: %v", tt.Texts())
	}
	if !tt.HasText("Open the desktop") {
		t.Fatalf("autostart=true did not select the desktop option: %v", tt.Texts())
	}
}
