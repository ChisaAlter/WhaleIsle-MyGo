package main

import (
	"testing"

	"github.com/egoist/mygo"

	"whaleisle/internal/config"
)

// fakeLauncherStage records the window moves applyLauncherStage asks for.
// mainVisible follows showMain so a second sync sees the revealed desktop.
type fakeLauncherStage struct {
	haveLauncher bool
	mainShown    bool
	mainVisible_ bool
	quitting_    bool
	// windowAvailable false is the nil-window fallback (headless, isolated,
	// --background, failed creation): the launcher keeps the in-window panel.
	windowAvailable bool
	created         int
	closed          int
	shows           int
	events          []string
}

func (f *fakeLauncherStage) launcherOpen() bool { return f.haveLauncher }

func (f *fakeLauncherStage) showLauncher() bool {
	if !f.windowAvailable {
		return false
	}
	if !f.haveLauncher {
		f.haveLauncher = true
		f.created++
	}
	f.shows++
	f.events = append(f.events, "show-launcher")
	return true
}

func (f *fakeLauncherStage) closeLauncher() {
	f.haveLauncher = false
	f.closed++
	f.events = append(f.events, "close-launcher")
}

func (f *fakeLauncherStage) showMain() {
	f.mainShown = true
	f.mainVisible_ = true
	f.events = append(f.events, "show-main")
}

func (f *fakeLauncherStage) mainVisible() bool { return f.mainVisible_ }

func (f *fakeLauncherStage) quitting() bool { return f.quitting_ }

func TestLauncherFirstWindowGatesColdStart(t *testing.T) {
	for _, tc := range []struct {
		name                                   string
		isolated, hidden, background, settings bool
		want                                   bool
	}{
		{"cold foreground start", false, false, false, false, true},
		{"isolated desktop keeps the panel", true, false, false, false, false},
		{"login start keeps the main window", false, true, false, false, false},
		{"background never creates it", false, false, true, false, false},
		{"settings route needs the main window", false, false, false, true, false},
	} {
		if got := launcherFirstWindow(tc.isolated, tc.hidden, tc.background, tc.settings); got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestLauncherWindowOptionsMatchNative(t *testing.T) {
	opts := launcherWindowOptions(config.Config{})
	if opts.Width != 1060 || opts.Height != 660 {
		t.Fatalf("outer size = %dx%d, want 1060x660", opts.Width, opts.Height)
	}
	if opts.MinWidth != 860 || opts.MinHeight != 560 {
		t.Fatalf("minimum = %dx%d, want 860x560", opts.MinWidth, opts.MinHeight)
	}
	// window.js:910 show:false, then showLauncher restores/shows/focuses.
	if !opts.Hidden {
		t.Fatal("launcher window must be created hidden")
	}
	// markWindowTransparent + shellWindowChrome: transparent silhouette.
	if !opts.Transparent {
		t.Fatal("launcher window must be transparent")
	}
	if opts.TitleBarStyle != mygo.TitleBarHidden {
		t.Fatalf("title bar style = %v, want hidden", opts.TitleBarStyle)
	}
	if opts.TitleBarHeight != 48 {
		t.Fatalf("caption height = %d, want 48 (launcher.css:21)", opts.TitleBarHeight)
	}
	// chrome.js:48 autoHideMenuBar plus hideNativeMenu.
	if !opts.AutoHideMenuBar {
		t.Fatal("launcher window must hide the OS menu bar")
	}
	if opts.StateKey != "launcher" {
		t.Fatalf("state key = %q, want launcher (never the main window's geometry)", opts.StateKey)
	}
	cb := opts.WindowsCaptionButtons
	if cb == nil || cb.Width != 32 || cb.Height != 32 || cb.Top != 12 || cb.Right != 8 || cb.CornerRadius != 8 || cb.GlyphSize != 12 {
		t.Fatalf("caption buttons = %+v, want 32x32 top 12 right 8 radius 8 glyph 12", cb)
	}
	if opts.WindowsFrameCornerRadius != 20 {
		t.Fatalf("frame radius = %d, want 20", opts.WindowsFrameCornerRadius)
	}
}

func TestLauncherStageColdStartKeepsOneLauncher(t *testing.T) {
	stage := &fakeLauncherStage{windowAvailable: true}
	// Cold start: the launcher owns the visible window and the main window stays
	// hidden (window_fit.go/main.go: Hidden when the launcher comes first).
	res := applyLauncherStage(stage, true)
	if !res.LauncherShown || res.MainShown || res.LauncherClosed {
		t.Fatalf("cold start result = %+v", res)
	}
	// Any number of later syncs (tray, deep link, boot repaint) reuse it.
	for i := 0; i < 3; i++ {
		again := applyLauncherStage(stage, true)
		if again.LauncherShown || again.MainShown || again.LauncherClosed {
			t.Fatalf("repeat %d created work: %+v", i, again)
		}
	}
	if stage.created != 1 || stage.closed != 0 || stage.mainShown {
		t.Fatalf("created=%d closed=%d mainShown=%v", stage.created, stage.closed, stage.mainShown)
	}
}

func TestLauncherStageStartMovesToMainOnce(t *testing.T) {
	stage := &fakeLauncherStage{windowAvailable: true}
	applyLauncherStage(stage, true)
	// harness Ready with quitAfterStart: the stage flag drops.
	res := applyLauncherStage(stage, false)
	if !res.LauncherClosed || !res.MainShown {
		t.Fatalf("start hand-off = %+v", res)
	}
	// A late boot refresh after ready must not touch the windows again.
	if again := applyLauncherStage(stage, false); again.LauncherClosed || again.MainShown {
		t.Fatalf("second hand-off = %+v", again)
	}
	if stage.closed != 1 || len(stage.events) != 3 || stage.events[0] != "show-launcher" || stage.events[1] != "close-launcher" || stage.events[2] != "show-main" {
		t.Fatalf("events = %v", stage.events)
	}
}

func TestLauncherStageKeepsLauncherOnCancelAndError(t *testing.T) {
	// Cancel and failure both leave launcherOpen true (refreshBoot keeps the
	// stage; a protected restart that is cancelled keeps the current page).
	stage := &fakeLauncherStage{windowAvailable: true}
	applyLauncherStage(stage, true)
	res := applyLauncherStage(stage, true)
	if res.LauncherClosed || res.MainShown || stage.closed != 0 || stage.mainShown {
		t.Fatalf("cancel/error lost the launcher: %+v", res)
	}
	if !stage.haveLauncher {
		t.Fatal("launcher window must survive a failed or cancelled start")
	}
}

func TestLauncherStageFallsBackToPanelWindow(t *testing.T) {
	// No launcher window in this run: the in-window panel needs the main window
	// even before the stage ends (headless, isolated, --background, creation
	// failure).
	stage := &fakeLauncherStage{windowAvailable: false}
	res := applyLauncherStage(stage, true)
	if !res.MainShown || res.LauncherShown {
		t.Fatalf("fallback = %+v", res)
	}
	if stage.created != 0 || stage.haveLauncher {
		t.Fatal("fallback must not create a launcher window")
	}
	// Leaving the stage repaints nothing new; the panel simply disappears.
	stage.mainVisible_ = true
	if again := applyLauncherStage(stage, false); again.LauncherClosed || again.MainShown {
		t.Fatalf("fallback close = %+v", again)
	}
}

func TestLauncherStageQuittingNeverRevealsMain(t *testing.T) {
	stage := &fakeLauncherStage{windowAvailable: true, quitting_: true}
	if res := applyLauncherStage(stage, true); res.MainShown {
		t.Fatalf("quit showed the main window: %+v", res)
	}
	applyLauncherStage(stage, true)
	if res := applyLauncherStage(stage, false); !res.LauncherClosed || res.MainShown {
		t.Fatalf("quit-stage close = %+v", res)
	}
}
