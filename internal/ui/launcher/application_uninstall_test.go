package launcher

import (
	"errors"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/appuninstall"
	"whaleisle/internal/ui/i18n"
)

// All opens are fake. This fixture never invokes the OS or an uninstaller.
func TestApplicationUninstallEntryFixture(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "", "Windows"} {
		called := false
		err := appuninstall.OpenInstalledApps(goos, func(string) error { called = true; return nil })
		if !errors.Is(err, appuninstall.ErrUnsupported) || called {
			t.Fatalf("%q: unsupported platform dispatched, err=%v", goos, err)
		}
	}
	if err := appuninstall.OpenInstalledApps("windows", nil); !errors.Is(err, appuninstall.ErrUnavailable) {
		t.Fatalf("nil opener: %v", err)
	}
	openErr := errors.New("fake settings open failure")
	if err := appuninstall.OpenInstalledApps("windows", func(target string) error {
		if target != "ms-settings:appsfeatures" {
			t.Fatalf("unexpected target %q", target)
		}
		return openErr
	}); !errors.Is(err, openErr) {
		t.Fatalf("lost open error: %v", err)
	}

	for _, locale := range []string{"zh", "en"} {
		t.Run(locale, func(t *testing.T) {
			i18n.SetLocale(locale)
			defer i18n.SetLocale(i18n.DefaultLocale)
			calls := 0
			model := Model{ActiveTab: TabSettings, VendorReady: true}
			model.ApplicationUninstall = ApplicationUninstallModel{Supported: true, OnOpenSettings: func() {
				if err := appuninstall.OpenInstalledApps("windows", func(target string) error {
					if target != "ms-settings:appsfeatures" {
						t.Fatalf("unexpected target %q", target)
					}
					calls++
					return nil
				}); err != nil {
					t.Fatal(err)
				}
			}}
			tt := ui.NewTester(func(c *ui.Context) { View(c, model) }, 1060, 900)
			label := releaseText("打开“已安装应用”", "Open Installed apps")
			for _, text := range []string{
				releaseText("卸载应用", "Uninstall application"),
				releaseText("打开 Windows 的“已安装应用”。请核对所选应用及安装位置，再由你完成卸载；此入口不会自动卸载、清理数据或退出当前应用。", "Open Windows Installed apps. Check the selected application and install location, then complete uninstall there. This entry does not automatically uninstall, delete data or quit the app."),
				label,
			} {
				if !tt.HasText(text) {
					t.Fatalf("missing %q; texts=%q", text, tt.Texts())
				}
			}
			if calls != 0 {
				t.Fatal("render opened settings")
			}
			before := tt.Texts()
			for expected := 1; expected <= 2; expected++ {
				if err := tt.Click(label); err != nil {
					t.Fatal(err)
				}
				if calls != expected {
					t.Fatalf("one explicit click should open once: %d", calls)
				}
				tt.Frame() // Returning from or cancelling system settings has no app receipt.
				if calls != expected || !model.VendorReady || !reflect.DeepEqual(before, tt.Texts()) {
					t.Fatal("return/cancel changed installation state, UI or retriggered settings")
				}
			}
			model.ActiveTab = TabHome
			tt.Frame()
			model.ActiveTab = TabSettings
			tt.Frame()
			if calls != 2 {
				t.Fatal("navigation reopened settings")
			}
			model.ApplicationUninstall.Supported = false
			tt.Frame()
			if !tt.HasText(releaseText("此入口仅支持 Windows。请使用当前系统的应用管理方式卸载。", "This entry is only available on Windows. Use your operating system's application management to uninstall.")) {
				t.Fatal("unsupported platform has no explanation")
			}
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatal("unsupported button called opener")
			}
			model.ApplicationUninstall.Supported = true
			model.ApplicationUninstall.OnOpenSettings = nil
			tt.Frame()
			if err := tt.Click(label); err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatal("missing callback dispatched")
			}
		})
	}
}
