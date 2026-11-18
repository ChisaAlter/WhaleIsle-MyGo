package boot

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/i18n"
)

func TestPluginProgressLine(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	got := pluginProgressLine(harness.Status{
		State:      harness.Starting,
		PluginLoad: harness.PluginProgress{Phase: "runtime", Ready: 2, Total: 5, Detail: "web-app"},
	})
	want := fmt.Sprintf(i18n.T(i18n.KeyBootPlugins), 2, 5) + " — " + fmt.Sprintf(i18n.T(i18n.KeyBootPluginsDetail), "web-app")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStatusLine(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	if got := statusLine(harness.Status{State: harness.Starting}); got != i18n.T(i18n.KeyBootStarting) {
		t.Fatalf("starting = %q", got)
	}
	if got := statusLine(harness.Status{State: harness.Stopped}); got != i18n.T(i18n.KeyBootStopped) {
		t.Fatalf("stopped = %q", got)
	}
}

func TestLoadingPageComposition(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, tc := range []struct {
		name string
		dark bool
	}{{"day", false}, {"night", true}} {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Dark: tc.dark, Status: harness.Status{State: harness.Starting}, Logs: []string{"正在加载本地组件"}})
			}, 1440, 920)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			mark, ok := tt.Find("Whale Isle")
			if !ok || mark.Y < 240 || mark.Y > 300 || mark.X+mark.W/2 < 718 || mark.X+mark.W/2 > 722 {
				t.Fatalf("wordmark outside reference seat: %+v", mark)
			}
			if _, ok := tt.Find("启动中"); !ok {
				t.Fatal("no startup status")
			}
			if dir := os.Getenv("WHALEISLE_BOOT_EVIDENCE"); dir != "" {
				f, err := os.Create(filepath.Join(dir, "loading-"+tc.name+".png"))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if err := png.Encode(f, tt.Image()); err != nil {
					t.Fatal(err)
				}
			}
			if err := tt.Click("全部日志"); err != nil {
				t.Fatal(err)
			}
			if _, ok := tt.Find("关闭日志"); !ok {
				t.Fatal("logs did not open")
			}
			tt.Key(0, ui.KeyEscape)
			if _, ok := tt.Find("关闭日志"); ok {
				t.Fatal("logs still open after Escape")
			}
		})
	}
}

func TestWorkbenchCancelRecoveryThenRetry(t *testing.T) {
	i18n.SetLocale("en")
	defer i18n.SetLocale(i18n.DefaultLocale)
	st := harness.Status{State: harness.Starting, RecoveryActive: true, RecoveryScheduled: true}
	cancelled, retried, exported := 0, 0, 0
	tt := ui.NewTester(func(c *ui.Context) {
		m := Model{Status: st, OnCancelRecovery: func() {
			cancelled++
			st.RecoveryCancelled, st.RecoveryScheduled = true, false
		}}
		if st.RecoveryCancelled {
			m.OnRetry = func() { retried++ }
			m.OnExport = func() { exported++ }
		}
		WorkbenchStatus(c, m, !st.RecoveryCancelled, "", "")
	}, 800, 180)
	if tt.HasText(i18n.T(i18n.KeyBootRetry)) {
		t.Fatal("scheduled recovery must not offer a competing manual retry")
	}
	if err := tt.Click("Cancel automatic recovery"); err != nil {
		t.Fatal(err)
	}
	if cancelled != 1 || tt.HasText("Cancel automatic recovery") || !tt.HasText("Automatic recovery cancelled. You can retry manually.") {
		t.Fatal("cancelled recovery did not update inline actions")
	}
	if err := tt.Click(i18n.T(i18n.KeyBootRetry)); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click(i18n.T(i18n.KeyBootExport)); err != nil {
		t.Fatal(err)
	}
	if retried != 1 || exported != 1 {
		t.Fatalf("recovery actions: retry=%d export=%d", retried, exported)
	}
}
