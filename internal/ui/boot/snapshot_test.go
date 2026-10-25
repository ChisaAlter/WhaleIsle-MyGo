package boot

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/harness"
	"whaleisle/internal/ui/theme"
)

// TestBootShot is a scaffold: BOOT_SHOT_DIR=<dir> exports the page's own pixels
// so the batch's visual claim can be inspected instead of assumed.
func TestBootShot(t *testing.T) {
	dir := os.Getenv("BOOT_SHOT_DIR")
	if dir == "" {
		t.Skip("set BOOT_SHOT_DIR to export boot-page renders")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	prefs := theme.CurrentPreferences()
	theme.SetPreferences(theme.Defaults())
	defer theme.SetPreferences(prefs)
	cases := []struct {
		name string
		w, h int
		dark bool
		st   harness.Status
		logs []string
	}{
		{"starting-light", 1200, 800, false, harness.Status{State: harness.Starting, PluginLoad: harness.PluginProgress{Ready: 2, Total: 5, Detail: "web-app"}}, []string{"host: spawning dsh web", "web: listening on 127.0.0.1:8710"}},
		{"starting-dark", 1200, 800, true, harness.Status{State: harness.Starting}, []string{"host: spawning dsh web"}},
		{"failed-dark", 1200, 800, true, harness.Status{State: harness.Failed, Err: errStatus{}}, []string{"host: exit 1"}},
		{"starting-narrow", 700, 520, false, harness.Status{State: harness.Starting}, nil},
	}
	for _, tc := range cases {
		tt := ui.NewTester(func(c *ui.Context) {
			View(c, Model{Status: tc.st, Logs: tc.logs, ThemeID: "system", SystemDark: tc.dark, OnRetry: func() {}, OnExport: func() {}, OnConfigure: func() {}})
		}, tc.w, tc.h)
		tt.Frame()
		f, err := os.Create(filepath.Join(dir, tc.name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(f, tt.Image()); err != nil {
			f.Close()
			t.Fatal(err)
		}
		if err = f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	// The sheen: the same page two seconds into its 6s cycle, when the highlight
	// crosses the wordmark.
	sheen := ui.NewTester(func(c *ui.Context) {
		View(c, Model{Status: harness.Status{State: harness.Starting}, ThemeID: "system"})
	}, 1200, 800)
	sheen.SetTime(time.Unix(0, 0))
	sheen.SetTime(time.Unix(0, int64(2*time.Second)))
	f, err := os.Create(filepath.Join(dir, "sheen.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(f, sheen.Image()); err != nil {
		f.Close()
		t.Fatal(err)
	}
	if err = f.Close(); err != nil {
		t.Fatal(err)
	}
}

type errStatus struct{}

func (errStatus) Error() string { return "host exited before the port opened" }
