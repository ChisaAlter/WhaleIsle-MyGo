package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func TestNewSessionStoreLockedRespectsTrajectoryTab(t *testing.T) {
	s := &mainState{}
	s.mu.Lock()
	defaultStore := s.newSessionStoreLocked()
	s.chatPaneTab = "chat"
	chatStore := s.newSessionStoreLocked()
	s.chatPaneTab = "trajectory"
	trajStore := s.newSessionStoreLocked()
	s.mu.Unlock()

	if defaultStore.TrajectoryEnabled() {
		t.Fatal("default session store must start with lazy trajectory disabled")
	}
	if chatStore.TrajectoryEnabled() {
		t.Fatal("chat-tab session store must start with lazy trajectory disabled")
	}
	if !trajStore.TrajectoryEnabled() {
		t.Fatal("trajectory-tab session store must start with trajectory enabled")
	}
}

func TestLazyTrajectoryStoreRendersOnFirstTrajectoryFrame(t *testing.T) {
	previousLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(previousLocale) })

	store, _ := trajectoryFrameStore(t, true)
	store.SetTrajectoryEnabled(false)

	// Confirm the hot Published() view omits trajectory slices while disabled.
	hot := store.Published().Value
	if hot.TrajectoryMessages != nil || hot.TrajectoryRequests != nil || hot.TrajectoryRequestHeaders != nil {
		t.Fatalf("lazy trajectory still populated Published() view: msgs=%d reqs=%d hdrs=%d",
			len(hot.TrajectoryMessages), len(hot.TrajectoryRequests), len(hot.TrajectoryRequestHeaders))
	}

	// Switching to the Trajectory tab must populate the ledger immediately on
	// the first frame via on-demand TrajectoryPublished() with zero blank flash.
	layout := frame.DefaultLayout()
	layout.Bottom = 0
	tab := "trajectory"
	list := &ui.ListState{}
	tt := ui.NewTester(func(c *ui.Context) {
		frameSnap := store.BeginFrame()
		defer frameSnap.EndFrame()
		ui.Column(c).Fill().Children(func() {
			frame.View(c, frame.Model{
				Store:       store,
				FrameView:   frameSnap,
				Sizes:       &layout,
				ThemeID:     theme.IDDeepseekLight,
				ChatPaneTab: &tab,
				ChatList:    list,
			})
		})
	}, 1280, 800)
	tt.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})

	ledger, ok := tt.Find("Trajectory ledger")
	if !ok || ledger.W <= 0 || ledger.H <= 0 {
		t.Fatalf("trajectory ledger not rendered on lazy first frame: ledger=%+v found=%v", ledger, ok)
	}
	if tt.HasText(i18n.T(i18n.KeyTrajEmpty)) {
		t.Fatalf("lazy first trajectory frame showed empty state instead of on-demand projection: texts=%q", tt.Texts())
	}
	if !tt.HasText("Owned trajectory history") || !tt.HasText("Owned trajectory reply") {
		t.Fatalf("lazy first trajectory frame missing history turns: texts=%q", tt.Texts())
	}
	// Ensure frame rendering did not mutate the store's cached ViewState.
	if after := store.Published().Value; after.TrajectoryMessages != nil {
		t.Fatal("lazy frame fallback mutated store cached ViewState")
	}
}

func TestFlushPendingConfigWritesDrainsAllAsyncLayers(t *testing.T) {
	dir := t.TempDir()
	cfgStore, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Use very long debounces across every async persistence layer so only an
	// explicit synchronous flushPendingConfigWrites() can persist before reopen.
	const longDebounce = 10 * time.Second
	cfgStore.SetAsyncDebounce(longDebounce)

	vp := newWorkspaceViewPreferences(cfgStore)
	vp.enableAsync(longDebounce, func(err error) {
		t.Errorf("unexpected workspace preference flush error: %v", err)
	})
	prefs, err := vp.WorkspaceNavigationPreferences("host-flush")
	if err != nil {
		t.Fatal(err)
	}
	prefs.GroupBy = "flat"
	prefs.Collapsed = map[string]bool{"ws-flush": true}
	if err := vp.SaveWorkspaceNavigationPreferences("host-flush", prefs); err != nil {
		t.Fatal(err)
	}

	writer := newAsyncConfigWriter(cfgStore, longDebounce, func(err error) {
		t.Errorf("unexpected asyncConfigWriter flush error: %v", err)
	})
	writer.EnqueueMerge(func(c *config.Config) {
		c.SidebarWidth = 344
	})

	if err := cfgStore.UpdateAsync(func(c *config.Config) {
		c.Theme = "deepseek.dark"
	}); err != nil {
		t.Fatal(err)
	}

	sessStore := session.New()
	sessStore.SetWorkspacePreferencesAsync(true)
	sessStore.SetWorkspacePreferencesDebounce(longDebounce)

	state := &mainState{
		cfgStore:        cfgStore,
		configWriter:    writer,
		viewPreferences: vp,
		sessions:        sessStore,
	}

	if err := state.flushPendingConfigWrites(); err != nil {
		t.Fatalf("flushPendingConfigWrites failed: %v", err)
	}

	reopened, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	gotCfg := reopened.Get()
	if gotCfg.Theme != "deepseek.dark" {
		t.Fatalf("config.Store UpdateAsync not flushed: theme=%q", gotCfg.Theme)
	}
	if gotCfg.SidebarWidth != 344 {
		t.Fatalf("asyncConfigWriter not flushed: sidebarWidth=%v", gotCfg.SidebarWidth)
	}
	gotPrefs, err := reopened.WorkspaceNavigationPreferences("host-flush")
	if err != nil {
		t.Fatal(err)
	}
	if gotPrefs.GroupBy != "flat" || !gotPrefs.Collapsed["ws-flush"] {
		t.Fatalf("workspaceViewPreferences not flushed: %+v", gotPrefs)
	}
}
