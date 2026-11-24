package chrome

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/ui/i18n"
)

func TestPRPanelDoesNotSwallowWindowHeight(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)

	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DrawPRPanel(c, DefaultTitleBarColors(), true, PRPanel{
				CurrentLabel: "#42 Fix layout [OPEN]",
				ViewLabel:    "View PR",
				RefreshLabel: "Refresh",
				CloseLabel:   "Close",
				Recent: []PRListItem{
					{Label: "#41 Earlier [OPEN]", CheckoutLabel: "Checkout", OnCheckout: func() {}},
				},
				OnView:    func() {},
				OnRefresh: func() {},
				OnClose:   func() {},
			})
			ui.Box(c.Key("workspace-body")).FillWidth().Grow(1).MinHeight(0).Children(func() {
				ui.Text(c, "MAIN_WORKSPACE_VISIBLE")
			})
		})
	}, 900, 600)

	bodyRect, ok := tt.Find("MAIN_WORKSPACE_VISIBLE")
	if !ok {
		t.Fatalf("workspace body was pushed out of viewport: %v", tt.Texts())
	}
	if bodyRect.Y > 180 {
		t.Fatalf("PR panel consumed too much height: body starts at y=%.1f (want <= 180)", bodyRect.Y)
	}
	if bodyRect.Y <= 0 {
		t.Fatalf("PR panel collapsed to zero height: body starts at y=%.1f", bodyRect.Y)
	}
}

func TestUpdateProgressDoesNotSwallowWindowHeight(t *testing.T) {
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			DrawUpdateProgress(c, DefaultTitleBarColors(), true, UpdateProgress{
				Title:        "Downloading update 1.2.3",
				Detail:       "12.4 MiB / 40.0 MiB",
				Percent:      31,
				DismissLabel: "Dismiss",
			}, func() {})
			ui.Box(c.Key("workspace-body")).FillWidth().Grow(1).MinHeight(0).Children(func() {
				ui.Text(c, "MAIN_WORKSPACE_VISIBLE")
			})
		})
	}, 900, 600)

	bodyRect, ok := tt.Find("MAIN_WORKSPACE_VISIBLE")
	if !ok {
		t.Fatalf("workspace body was pushed out of viewport: %v", tt.Texts())
	}
	if bodyRect.Y > 140 {
		t.Fatalf("update progress banner consumed too much height: body starts at y=%.1f (want <= 140)", bodyRect.Y)
	}
	if bodyRect.Y <= 0 {
		t.Fatalf("update progress banner collapsed to zero height: body starts at y=%.1f", bodyRect.Y)
	}
}

func TestTopStripsReserveTitleBarSafeArea(t *testing.T) {
	cases := []struct {
		name  string
		width int
		bar   ui.TitleBar
	}{
		{name: "narrow-windows", width: 640, bar: ui.TitleBar{Height: 12, Right: 104}},
		{name: "standard-windows", width: 1024, bar: ui.TitleBar{Height: 12, Right: 104}},
		{name: "win11-wide-caption", width: 1024, bar: ui.TitleBar{Height: 32, Right: 138}},
		{name: "macos-traffic-lights", width: 800, bar: ui.TitleBar{Height: 28, Left: 72}},
		{name: "fullscreen", width: 1280, bar: ui.TitleBar{}},
	}
	for _, tc := range cases {
		t.Run("pr-panel/"+tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					DrawPRPanel(c, DefaultTitleBarColors(), true, PRPanel{
						CurrentLabel: "#99 Very long pull request title that must ellipsize cleanly",
						ViewLabel:    "View",
						CreateLabel:  "Create",
						RefreshLabel: "Refresh",
						CloseLabel:   "Close",
						OnView:       func() {},
						OnCreate:     func() {},
						OnRefresh:    func() {},
						OnClose:      func() {},
					})
				})
			}, tc.width, 200)
			tt.SetTitleBar(tc.bar)
			maxRight := float32(tc.width) - tc.bar.Right
			for _, label := range []string{"View", "Create", "Refresh", "Close"} {
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing %q", label)
				}
				if r.X < tc.bar.Left {
					t.Fatalf("%s overlaps left caption safe area: x=%.1f < %.1f", label, r.X, tc.bar.Left)
				}
				if r.X+r.W > maxRight+0.5 {
					t.Fatalf("%s overlaps right caption safe area: right=%.1f > %.1f", label, r.X+r.W, maxRight)
				}
			}
		})

		t.Run("update-progress/"+tc.name, func(t *testing.T) {
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					DrawUpdateProgress(c, DefaultTitleBarColors(), true, UpdateProgress{
						Title:        "Downloading update",
						Percent:      50,
						DismissLabel: "Dismiss",
					}, func() {})
				})
			}, tc.width, 200)
			tt.SetTitleBar(tc.bar)
			r, ok := tt.Find("Dismiss")
			if !ok {
				t.Fatal("missing Dismiss button")
			}
			if r.X+r.W > float32(tc.width)-tc.bar.Right+0.5 {
				t.Fatalf("Dismiss overlaps right caption safe area: right=%.1f > %.1f", r.X+r.W, float32(tc.width)-tc.bar.Right)
			}
		})

		t.Run("git-commit/"+tc.name, func(t *testing.T) {
			msg := "fix: test"
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					DrawGitCommitStrip(c, DefaultTitleBarColors(), true, GitCommitStrip{
						Message:       &msg,
						Dirty:         true,
						Placeholder:   "Commit message",
						GenerateLabel: "Generate",
						CommitLabel:   "Commit",
						CancelLabel:   "Cancel",
						OnGenerate:    func() {},
						OnCommit:      func() {},
						OnCancel:      func() {},
					})
				})
			}, tc.width, 200)
			tt.SetTitleBar(tc.bar)
			maxRight := float32(tc.width) - tc.bar.Right
			for _, label := range []string{"Generate", "Commit", "Cancel"} {
				r, ok := tt.Find(label)
				if !ok {
					t.Fatalf("missing %q", label)
				}
				if r.X+r.W > maxRight+0.5 {
					t.Fatalf("%s overlaps right caption safe area: right=%.1f > %.1f", label, r.X+r.W, maxRight)
				}
			}
		})
	}
}

func TestAmbientGradientLatestResizeStaysAsync(t *testing.T) {
	cache := gradientFrameCache{}
	c := latestTestConfig()
	now := time.Unix(200, 0)
	first := cache.latest(c, now)
	if first == nil || cache.renders != 1 {
		t.Fatalf("first frame must render synchronously: frame=%v renders=%d", first, cache.renders)
	}
	resized := c
	resized.width = 1200
	resized.height = 720
	if got := cache.latest(resized, now); got != first {
		t.Fatal("dimension-only resize rendered synchronously instead of reusing cached frame")
	}
	waitRenders(t, &cache, 2)
	settled := cache.latest(resized, now)
	if settled == nil || settled == first || settled.config != resized {
		t.Fatalf("resized gradient frame did not settle: %+v", settled)
	}
}
