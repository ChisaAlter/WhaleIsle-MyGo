package launcher

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestRuntimeTransitionBlocksComponentActions(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, running := range []bool{false, true} {
		t.Run(map[bool]string{false: "start", true: "stop"}[running], func(t *testing.T) {
			calls := 0
			label := "Start Host"
			state := harness.Stopped
			if running {
				label = "Stop Host"
				state = harness.Ready
			}
			render := func(busy bool) *ui.Tester {
				return ui.NewTester(func(c *ui.Context) {
					Maintenance(c, MaintenanceComponents, Model{ThemeID: "deepseek", VendorReady: true,
						HostRuntimeState: state, ComponentBusy: busy,
						Components: []components.Row{{ID: components.HostComponentID, InstalledVersion: "0.2.1"}},
						OnStart:    func() { calls++ }, OnStop: func() { calls++ }})
				}, 1060, 660)
			}
			blocked := render(true)
			if !blocked.HasText(label) {
				t.Fatal("pending action lost its visible control")
			}
			_ = blocked.Click(label)
			if calls != 0 {
				t.Fatal("lifecycle transition admitted a conflicting action")
			}
			ready := render(false)
			if err := ready.Click(label); err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("settled lifecycle action calls=%d", calls)
			}
		})
	}
}
