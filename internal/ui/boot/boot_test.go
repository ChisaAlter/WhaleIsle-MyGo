package boot

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
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
	if got := StatusLine(harness.Status{State: harness.Starting}); got != i18n.T(i18n.KeyBootStarting) {
		t.Fatalf("starting = %q", got)
	}
	if got := StatusLine(harness.Status{State: harness.Stopped}); got != i18n.T(i18n.KeyBootStopped) {
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
			// The seat is the reference page's own: 44/54 letters over the sky,
			// measured from the type scale, not from a face. Pin the metrics so
			// the numbers describe that composition: a host resolving a brand
			// serif it lacks, or a CJK fallback for 鲸屿, wraps these rows and
			// moves the block by half their height.
			tt.UseFixedTextMetrics()
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

func TestFailureDetailDock(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	boom := errors.New("Error: Cannot find module 'dsh-web'")
	for _, tc := range []struct {
		name string
		dark bool
		want [3]uint32
	}{
		{"night", true, [3]uint32{67, 69, 74}},
		{"day", false, [3]uint32{53, 54, 56}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				View(c, Model{Dark: tc.dark, Status: harness.Status{State: harness.Failed, Err: boom},
					Logs: []string{"正在加载本地组件"}, OnRetry: func() {}, OnExport: func() {}, OnConfigure: func() {}})
			}, 1440, 920)
			tt.SetPreferences(ui.Preferences{ReduceMotion: true})
			actions, ok := tt.Find("重试")
			if !ok {
				t.Fatal("no retry action")
			}
			dock, ok := tt.Find("启动详情")
			if !ok {
				t.Fatal("no failure detail dock")
			}
			// The native page seats .failure's pane in the free band below the
			// actions (feedback.js placeStack), not under the status line.
			if dock.Y != actions.Y+actions.H+8 {
				t.Fatalf("dock %+v is not 8 DIP under the actions %+v", dock, actions)
			}
			if dock.W != 640 || dock.X+dock.W/2 != 720 {
				t.Fatalf("dock not the 640 DIP centred pane: %+v", dock)
			}
			if !tt.HasText("找不到需要的文件或位置。请确认它仍然存在后重试；详情见日志。") {
				t.Fatal("dock does not carry the localized failure copy")
			}
			if tt.HasText(boom.Error()) {
				t.Fatal("raw error reached the page copy")
			}
			// Sample the left padding: the copy wraps, and a centred sample can land
			// on a glyph rather than the surface this checks.
			r, g, b, _ := tt.Image().At(int(dock.X+8), int(dock.Y+dock.H/2)).RGBA()
			if got := [3]uint32{r >> 8, g >> 8, b >> 8}; got != tc.want {
				t.Fatalf("dock surface %v, want %v", got, tc.want)
			}
			// The technical detail stays in the drawer the copy points at.
			if err := tt.Click("全部日志"); err != nil {
				t.Fatal(err)
			}
			if !tt.HasText(boom.Error()) {
				t.Fatal("drawer lost the raw failure message")
			}
		})
	}
}

func TestRuntimeFailureTitle(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	tt := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Status: harness.Status{State: harness.Failed, RuntimeFailure: true,
			Err: errors.New("dsh web exited: exit status 1")}, OnRetry: func() {}})
	}, 1440, 920)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	if !tt.HasText("桌面端意外退出") {
		t.Fatal("a ready exit is not titled 桌面端意外退出")
	}
	if !tt.HasText("立即重启") {
		t.Fatal("a ready exit does not offer 立即重启")
	}
}

// TestFailureDockNamesTheFailure pins what the page prints for the startup
// failures a person actually hits: the exit code and the system kill. The
// sentences live in internal/errcopy, and this is the screen-level check that
// they reach it and the raw text does not.
func TestFailureDockNamesTheFailure(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	for _, tc := range []struct{ raw, want string }{
		{"dsh web exited: exit status 137", "桌面运行时已退出（退出码 137），可重试或到日志里查看原因。"},
		{"dsh web exited: signal: killed", "桌面运行时被系统结束，可重试或到日志里查看原因。"},
		{"dsh web exited", "桌面运行时已退出，可重试或到日志里查看原因。"},
		{"EACCES: permission denied", "没有权限访问所需资源，请检查安全软件拦截或系统权限后重试；详情见日志。"},
		{"spawn dsh ENOENT", "桌面运行时文件缺失，请重新安装桌面端。"},
	} {
		raw := tc.raw
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{Status: harness.Status{State: harness.Failed, Err: errors.New(raw)}, OnRetry: func() {}})
		}, 1440, 920)
		tt.SetPreferences(ui.Preferences{ReduceMotion: true})
		if !tt.HasText(tc.want) {
			t.Fatalf("%q did not reach the dock: %q", raw, tt.Texts())
		}
		if tt.HasText(raw) {
			t.Fatalf("raw %q reached the page copy", raw)
		}
	}
}

func TestWorkbenchFailureLines(t *testing.T) {
	i18n.SetLocale("zh")
	defer i18n.SetLocale(i18n.DefaultLocale)
	raw := errors.New("dsh web exited: exit status 1")
	if got := StatusLine(harness.Status{State: harness.Failed, Err: raw}); got != "启动失败，可重试；原因见日志。" {
		t.Fatalf("start failure line = %q", got)
	}
	if got := StatusLine(harness.Status{State: harness.Failed, Err: raw, RuntimeFailure: true}); got != "桌面端意外退出，可立即重启；原因见日志。" {
		t.Fatalf("runtime failure line = %q", got)
	}
	for _, st := range []harness.Status{
		{State: harness.Failed, Err: raw},
		{State: harness.Failed, Err: raw, RuntimeFailure: true},
	} {
		if strings.Contains(StatusLine(st), raw.Error()) {
			t.Fatalf("raw error reached the workbench line: %q", StatusLine(st))
		}
	}
}
