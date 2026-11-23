package main

import (
	"context"
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/app/launcher"
	"whaleisle/internal/app/market"
	"whaleisle/internal/app/updater"
	"whaleisle/internal/config"
	desktopupdate "whaleisle/internal/desktop/update"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func (s *mainState) updaterCenter() *updater.Center {
	if s == nil {
		return updater.NewCenter(nil)
	}
	s.initCtrlMu.Lock()
	if s.updaterCtl == nil {
		s.updaterCtl = updater.NewCenter(mainUpdaterBridge{s: s})
	}
	u := s.updaterCtl
	s.initCtrlMu.Unlock()
	return u
}

func (s *mainState) marketCoordinator() *marketctl.Coordinator {
	if s == nil {
		return marketctl.NewCoordinator(nil)
	}
	s.initCtrlMu.Lock()
	if s.marketCtl == nil {
		s.marketCtl = marketctl.NewCoordinator(mainMarketDeps{s: s})
	}
	m := s.marketCtl
	s.initCtrlMu.Unlock()
	return m
}

func (s *mainState) launcherController() *launcherctl.Controller {
	if s == nil {
		return launcherctl.NewController(nil)
	}
	s.initCtrlMu.Lock()
	if s.launcherCtl == nil {
		s.launcherCtl = launcherctl.NewController(mainLauncherDeps{s: s})
	}
	l := s.launcherCtl
	s.initCtrlMu.Unlock()
	return l
}

// --- updater.ShellBridge ---

type mainUpdaterBridge struct{ s *mainState }

type mygoNoticeHandle struct{ n *mygo.Notification }

func (h mygoNoticeHandle) OnClick(fn func()) { h.n.OnClick(fn) }
func (h mygoNoticeHandle) Show() error       { return h.n.Show() }
func (h mygoNoticeHandle) Close()            { h.n.Close() }

func (b mainUpdaterBridge) Environment() updater.ShellEnvironment {
	route := b.s.updateDownloadRoute()
	startBusy := b.s.launcherController().StartBusy()

	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	ask := b.s.cfgStore != nil && b.s.cfgStore.Get().AskOnUpdate
	gen := b.s.bootStatus.Generation
	if b.s.sup != nil {
		gen = b.s.sup.Status().Generation
	}
	return updater.ShellEnvironment{
		BuildVersion:       buildVersion,
		DownloadRoute:      route,
		Quitting:           b.s.quitting,
		WorkbenchCanPrompt: !b.s.settingsOpen && !b.s.maintenanceOpen && b.s.welcomeWin == nil && b.s.confirmAsk == nil,
		LauncherStartBusy:  startBusy,
		LauncherImportBusy: b.s.launcherImportBusy,
		TaskRestartPending: b.s.taskRestartPending,
		AskOnUpdate:        ask,
		HostID:             fmt.Sprintf("%p", b.s.sup),
		HostGeneration:     gen,
	}
}

func (b mainUpdaterBridge) AutoDownloadPref() (bool, bool) {
	b.s.mu.Lock()
	cfg := b.s.cfgStore
	b.s.mu.Unlock()
	if cfg == nil {
		return true, false
	}
	return cfg.Get().AutoDownloadUpdates, cfg.HasAutoDownloadUpdates()
}

func (b mainUpdaterBridge) InitAutoDownloadPref(migratedValue bool) error {
	b.s.mu.Lock()
	cfg := b.s.cfgStore
	b.s.mu.Unlock()
	if cfg == nil || cfg.HasAutoDownloadUpdates() {
		return nil
	}
	return cfg.Update(func(c *config.Config) { c.AutoDownloadUpdates = migratedValue })
}

func (b mainUpdaterBridge) SetAutoDownloadPref(enabled bool) error {
	b.s.mu.Lock()
	cfg := b.s.cfgStore
	b.s.mu.Unlock()
	if cfg == nil {
		return nil
	}
	return cfg.Update(func(c *config.Config) { c.AutoDownloadUpdates = enabled })
}

func (b mainUpdaterBridge) LifecycleContext() context.Context {
	return b.s.lifecycleContext()
}

func (b mainUpdaterBridge) UserDataDir() string {
	return paths.UserDataDir()
}

func (b mainUpdaterBridge) OwnedWindow(win updater.WindowRef) bool {
	w, ok := win.(*mygo.Window)
	if !ok || w == nil {
		return false
	}
	b.s.mu.Lock()
	defer b.s.mu.Unlock()
	return b.s.win == w || (b.s.whaleBridgeWin != nil && b.s.whaleBridgeWin.win == w)
}

func (b mainUpdaterBridge) WindowAlive(win updater.WindowRef) bool {
	w, ok := win.(*mygo.Window)
	return ok && w != nil && !w.IsDestroyed()
}

func (b mainUpdaterBridge) WindowVisibility(win updater.WindowRef) (bool, bool) {
	w, ok := win.(*mygo.Window)
	if !ok || w == nil || w.IsDestroyed() {
		return false, false
	}
	vis := w.IsVisible() && !w.IsMinimized()
	return vis, vis && w.IsFocused()
}

func (b mainUpdaterBridge) WindowHasPage(win updater.WindowRef) bool {
	w, ok := win.(*mygo.Window)
	return ok && w != nil && w.Page() != nil
}

func (b mainUpdaterBridge) WindowKey(win updater.WindowRef) string {
	w, _ := win.(*mygo.Window)
	return fmt.Sprintf("%p", w)
}

func (b mainUpdaterBridge) DialogOwner(preferred updater.WindowRef) updater.WindowRef {
	p, _ := preferred.(*mygo.Window)
	return b.s.lifecycleDialogOwner(p)
}

func (b mainUpdaterBridge) PrimaryWindows() (updater.WindowRef, updater.WindowRef) {
	b.s.mu.Lock()
	m := b.s.win
	b.s.mu.Unlock()
	return updaterWindow(m), nil
}

func (b mainUpdaterBridge) RunOnMain(fn func()) { mygo.RunOnMain(fn) }

func (b mainUpdaterBridge) UpdateWindow(win updater.WindowRef, fn func()) {
	if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
		w.Update(fn)
	}
}

func (b mainUpdaterBridge) InvalidateWindow(win updater.WindowRef) {
	if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
		w.Invalidate()
	}
}

func (b mainUpdaterBridge) RestoreShowFocus(win updater.WindowRef) {
	if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
		w.Restore()
		w.Show()
		w.Focus()
	}
}

func (b mainUpdaterBridge) FlashFrame(win updater.WindowRef, flash bool) {
	if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
		w.FlashFrame(flash)
	}
}

func (b mainUpdaterBridge) SystemNotificationsSupported() bool {
	return systemNotificationsSupported()
}

func (b mainUpdaterBridge) NewNotification(title, body string, silent bool) updater.NoticeHandle {
	n := mygo.NewNotification(mygo.NotificationOptions{Title: title, Body: body, Silent: silent})
	if n == nil {
		return nil
	}
	return mygoNoticeHandle{n: n}
}

func (b mainUpdaterBridge) WatchFocusAndClose(win updater.WindowRef, onFocus, onClose func()) func() {
	w, ok := win.(*mygo.Window)
	if !ok || w == nil {
		return func() {}
	}
	var offs []func()
	if onFocus != nil {
		offs = append(offs, w.OnFocus(onFocus))
	}
	if onClose != nil {
		offs = append(offs, w.OnClosed(onClose))
	}
	return func() {
		for _, off := range offs {
			off()
		}
	}
}

func (b mainUpdaterBridge) WatchModalWindows(parent, owner updater.WindowRef, onDismiss func()) func() {
	p, _ := parent.(*mygo.Window)
	o, _ := owner.(*mygo.Window)
	var offs []func()
	if p != nil {
		offs = append(offs, p.OnClosed(onDismiss), p.OnHide(onDismiss))
	}
	if o != nil {
		offs = append(offs, o.OnClosed(onDismiss))
	}
	return func() {
		for _, off := range offs {
			off()
		}
	}
}

func (b mainUpdaterBridge) OpenModalWindow(parent updater.WindowRef, title string) updater.WindowRef {
	p, _ := parent.(*mygo.Window)
	var dialog *mygo.Window
	dialog = mygo.NewWindow(mygo.WindowOptions{
		Parent: p, Modal: true, Title: title,
		Width: 648, Height: 720, MinWidth: 300, MinHeight: 260,
		Frameless: true, Transparent: true, AutoHideMenuBar: true, BackgroundColor: "#00000000",
		Content: ui.View(func(c *ui.Context) {
			ui.Box(c).Fill().Background(ui.Transparent)
			b.s.syncNativeUpdateModal(c, dialog)
			b.s.syncConfirmAskFor(c, dialog)
		}),
	})
	return dialog
}

func (b mainUpdaterBridge) DestroyWindow(win updater.WindowRef) {
	if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
		w.Destroy()
	}
}

func (b mainUpdaterBridge) RefreshTray() { b.s.refreshDesktopTray() }

func (b mainUpdaterBridge) AskConfirm(req updater.ConfirmRequest) {
	owner, _ := req.Owner.(*mygo.Window)
	b.s.askConfirm(confirmAskRequest{
		Owner:            owner,
		Width:            float32(req.Width),
		Title:            req.Title,
		Body:             req.Body,
		ConfirmLabel:     req.ConfirmLabel,
		CancelLabel:      req.CancelLabel,
		TechnicalDetails: req.TechnicalDetails,
		Run:              req.Run,
		Cancel:           req.Cancel,
	})
}

func (b mainUpdaterBridge) AskConfirmBlocking(req updater.ConfirmRequest) bool {
	owner, _ := req.Owner.(*mygo.Window)
	return b.s.askConfirmBlocking(confirmAskRequest{
		Owner:            owner,
		Width:            float32(req.Width),
		Title:            req.Title,
		Body:             req.Body,
		ConfirmLabel:     req.ConfirmLabel,
		CancelLabel:      req.CancelLabel,
		TechnicalDetails: req.TechnicalDetails,
		Run:              req.Run,
		Cancel:           req.Cancel,
	})
}

func (b mainUpdaterBridge) InstallStagedWithConsent(win updater.WindowRef, staged desktopupdate.ApplyResult, confirmed bool) bool {
	w, _ := win.(*mygo.Window)
	return b.s.installStagedUpdateWithConsent(w, staged, confirmed)
}

func (b mainUpdaterBridge) SaveFailedToast(action string, err error) string {
	return saveFailedToast(action, err)
}

func (b mainUpdaterBridge) SetLauncherUpdateNotice(text string) {
	b.s.launcherController().SetUpdateMessage(text)
}

// --- marketctl.HostDeps ---

type mainMarketDeps struct{ s *mainState }

func (d mainMarketDeps) Env() marketctl.HostEnvSnapshot {
	startBusy := d.s.launcherController().StartBusy()
	d.s.mu.Lock()
	defer d.s.mu.Unlock()
	gen := d.s.bootStatus.Generation
	if d.s.sup != nil {
		gen = d.s.sup.Status().Generation
	}
	node := ""
	if d.s.cfgStore != nil {
		node = d.s.cfgStore.Get().NodeBin
	}
	return marketctl.HostEnvSnapshot{
		LauncherImportBusy: d.s.launcherImportBusy,
		Quitting:           d.s.quitting,
		TaskRestartPending: d.s.taskRestartPending,
		LauncherStartBusy:  startBusy,
		HostStarting:       d.s.bootStatus.State == harness.Starting,
		HasSupervisor:      d.s.sup != nil,
		BootGeneration:     gen,
		NodeBin:            node,
	}
}

func (d mainMarketDeps) LifecycleContext() context.Context { return d.s.lifecycleContext() }

func (d mainMarketDeps) GoPluginInstallRegistry() (*config.PluginInstallRegistry, error) {
	d.s.mu.Lock()
	cfg := d.s.cfgStore
	d.s.mu.Unlock()
	if cfg == nil {
		return nil, nil
	}
	return cfg.Get().GoPluginInstallRegistry, nil
}

func (d mainMarketDeps) SaveGoPluginInstallRegistry(choice *config.PluginInstallRegistry) error {
	d.s.mu.Lock()
	cfg := d.s.cfgStore
	d.s.mu.Unlock()
	if cfg == nil {
		return fmt.Errorf("%s", i18n.T(i18n.KeyCmdPluginPreferencesUnavailable))
	}
	return cfg.Update(func(value *config.Config) {
		value.GoPluginInstallRegistry = choice
	})
}

func (d mainMarketDeps) SyncHarnessAndRestart() {
	d.s.mu.Lock()
	sup := d.s.sup
	d.s.mu.Unlock()
	if sup == nil {
		return
	}
	d.s.syncHarnessFromConfig()
	d.s.restartUnlessImport(sup)
}

func (d mainMarketDeps) UpdateWindow(fn func()) {
	d.s.mu.Lock()
	win := d.s.win
	d.s.mu.Unlock()
	if win != nil {
		win.Update(fn)
	}
}

func (d mainMarketDeps) InvalidateWindow() {
	d.s.invalidateMainWindow()
}

func (d mainMarketDeps) CloseWebViewHost() {
	d.s.closeWebViewHost()
}

// --- launcherctl.HostDeps ---

type mainLauncherDeps struct{ s *mainState }

func (d mainLauncherDeps) Env() launcherctl.HostEnv {
	route := d.s.updateDownloadRoute()
	asst := d.s.assistantService()
	asstOpen := asst.Open()
	asstGen := asst.ConversationGeneration()
	d.s.mu.Lock()
	defer d.s.mu.Unlock()
	var startLogs []string
	if d.s.sup != nil {
		startLogs = d.s.sup.CurrentStartLogs()
	}
	exclusive := d.s.marketOpen || d.s.scheduleOpen || asstOpen || d.s.settingsOpen ||
		d.s.maintenanceOpen || d.s.usageOpen || d.s.imOpen || d.s.remoteOpen || d.s.webviewOpen || d.s.browserPreviewOpen
	occupiedDraft := false
	for _, draft := range d.s.composerDrafts {
		if workspaceSessionDraftOccupied(draft) || workspaceSessionDraftRevision(draft) != 0 {
			occupiedDraft = true
			break
		}
	}
	var mainWin launcherctl.WindowRef
	if d.s.win != nil {
		mainWin = d.s.win
	}
	return launcherctl.HostEnv{
		BuildVersion:       buildVersion,
		DownloadRoute:      route,
		Quitting:           d.s.quitting,
		LauncherImportBusy: d.s.launcherImportBusy,
		TaskRestartPending: d.s.taskRestartPending,
		HostStarting:       d.s.bootStatus.State == harness.Starting,
		HasSupervisor:      d.s.sup != nil,
		HasSessions:        d.s.sessions != nil,
		BootStatus:         d.s.bootStatus,
		BootLogs:           d.s.bootLogSnapshot(),
		StartLogs:          startLogs,
		ConfigStore:        d.s.cfgStore,
		MainWindow:         mainWin,
		ExclusivePanelOpen: exclusive,
		HasOccupiedDrafts:  occupiedDraft,
		AssistantPanelGen:  asstGen,
		WorkspaceOpenGen:   d.s.workspaceSessionOpen.generation,
	}
}

func (d mainLauncherDeps) LifecycleContext() context.Context { return d.s.lifecycleContext() }
func (d mainLauncherDeps) Updater() *updater.Center          { return d.s.updaterCenter() }
func (d mainLauncherDeps) MarketBusy() bool                  { return d.s.marketCoordinator().IsBusy() }
func (d mainLauncherDeps) TryAcquireRepairSlot() (func(), bool) {
	return d.s.marketCoordinator().TryAcquireRepairSlot()
}

func (d mainLauncherDeps) TrySetTaskRestartPending(pending bool) bool {
	d.s.mu.Lock()
	defer d.s.mu.Unlock()
	if pending && d.s.taskRestartPending {
		return false
	}
	d.s.taskRestartPending = pending
	return true
}

func (d mainLauncherDeps) ConfigureGenericComponents() { d.s.configureGenericComponents() }
func (d mainLauncherDeps) SyncHarnessFromConfig()      { d.s.syncHarnessFromConfig() }
func (d mainLauncherDeps) StartHarnessWithConfig(cfg config.Config) {
	startHarness(cfg)
}

func (d mainLauncherDeps) RestartSupervisorIfPresent() {
	d.s.mu.Lock()
	sup := d.s.sup
	d.s.mu.Unlock()
	if sup != nil {
		d.s.restartUnlessImport(sup)
	}
}

func (d mainLauncherDeps) StopSupervisorIfPresent() {
	d.s.mu.Lock()
	sup := d.s.sup
	d.s.mu.Unlock()
	if sup != nil {
		sup.Stop()
	}
}

func (d mainLauncherDeps) HasCurrentSupervisor() bool {
	d.s.mu.Lock()
	defer d.s.mu.Unlock()
	return d.s.sup != nil
}

func (d mainLauncherDeps) WithTaskProtection(win launcherctl.WindowRef, reason string, fn func() error) bool {
	w, _ := win.(*mygo.Window)
	d.s.mu.Lock()
	sup := d.s.sup
	d.s.mu.Unlock()
	return d.s.withTaskProtection(w, sup, reason, fn)
}

func (d mainLauncherDeps) CloseTaskProtectedResources() error {
	return d.s.closeTaskProtectedResources()
}

func (d mainLauncherDeps) OpenStartupBlankSession(win launcherctl.WindowRef, sessions *session.Store, hostGen uint64, workspaceID string) {
	w, _ := win.(*mygo.Window)
	if w == nil || sessions == nil {
		return
	}
	op, err := d.s.captureWorkspaceSessionOpen(w, sessions, hostGen, workspaceID)
	if err != nil || op == nil || !op.current() {
		return
	}
	go func() {
		id, err := op.target.Create(context.Background())
		if !w.IsDestroyed() {
			w.Update(func() { op.finish(id, err) })
		}
	}()
}

func (d mainLauncherDeps) ObserveBridgeAlerts() { d.s.observeWhaleBridgeAlerts() }
func (d mainLauncherDeps) RefuseHostAction()    { d.s.refuseHostAction() }

func (d mainLauncherDeps) UpdateWindow(win launcherctl.WindowRef, fn func()) {
	if fn == nil {
		return
	}
	if w, ok := win.(*mygo.Window); ok {
		if w == nil {
			fn()
			return
		}
		if !w.IsDestroyed() {
			w.Update(fn)
		}
		return
	}
	if win == nil {
		fn()
	}
}

func (d mainLauncherDeps) InvalidateSurface(win launcherctl.WindowRef) {
	w, _ := win.(*mygo.Window)
	d.s.launcherSurfaceInvalidator(w)()
}

type mygoNativePathSetter struct{}

func (mygoNativePathSetter) SetName(n string)         { mygo.App.SetName(n) }
func (mygoNativePathSetter) SetUserDataPath(p string) { mygo.App.SetPath(mygo.PathUserData, p) }
func (mygoNativePathSetter) SetCachePath(p string)    { mygo.App.SetPath(mygo.PathCache, p) }
func (mygoNativePathSetter) SetLogsPath(p string)     { mygo.App.SetPath(mygo.PathLogs, p) }

var _ = theme.Resolve
