package settings

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestSettingsSwitchSurvivesDialogExitGhost covers the crash the settings
// close hit: DialogBase keeps a fading copy of its subtree for the exit
// transition (applyMinimalMode installs ui.SetDialogExit at startup), and the
// copy still runs each element's paint callbacks a pass after they were
// built. Querying the element itself inside Draw then hits the expired-slot
// diagnostic — the query must be captured while the element is built.
func TestSettingsSwitchSurvivesDialogExitGhost(t *testing.T) {
	ui.SetDialogExit(ui.DialogExit{Duration: 100 * time.Millisecond})
	defer ui.SetDialogExit(ui.DialogExit{})
	open := true
	value := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.DialogBase(c.Key("settings-modal"), &open, func(backdrop, panel ui.Element) {
			panel.Children(func() {
				settingsPreferenceSwitch(c, &value, settingsSwitchLabelled).Label("Probe switch")
			})
		})
	}, 800, 600)
	if _, ok := tt.Find("Probe switch"); !ok {
		t.Fatalf("switch is not reachable: %v", tt.Texts())
	}
	open = false
	tt.Frame()
}
