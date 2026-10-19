package launcher

import (
	"github.com/egoist/mygo/ui"
	"testing"
	"whaleisle/internal/ui/i18n"
)

func TestRuntimeTransitionBlocksHomeActions(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, running := range []bool{false, true} {
		t.Run(map[bool]string{false: "start", true: "stop"}[running], func(t *testing.T) {
			calls := 0
			label := i18n.T(i18n.KeyLauncherStartDesktop)
			if running {
				label = i18n.T(i18n.KeyLauncherCloseDesktop)
			}
			render := func(busy bool) *ui.Tester {
				return ui.NewTester(func(c *ui.Context) {
					View(c, Model{ThemeID: "deepseek", VendorReady: true, ActiveTab: TabHome,
						DesktopRunning: running, RuntimeBusy: busy,
						OnStart: func() { calls++ }, OnStop: func() { calls++ }})
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
