package main

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/egoist/mygo"
	assistantctl "whaleisle/internal/app/assistant"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/frame"
)

func (s *mainState) assistantService() *assistantctl.Service {
	if s == nil {
		return assistantctl.NewService(assistantctl.HostDeps{})
	}
	s.initCtrlMu.Lock()
	if s.assistantCtl == nil {
		s.assistantCtl = assistantctl.NewService(mainAssistantDeps{s: s}.asstHostDeps())
	}
	svc := s.assistantCtl
	s.initCtrlMu.Unlock()
	return svc
}

func (s *mainState) openProactive(userData string) {
	s.assistantService().OpenProactive(userData)
}

func (s *mainState) dispatchProactive(ctx context.Context, action string, payload json.RawMessage) (any, error) {
	return s.assistantService().DispatchProactive(ctx, action, payload)
}

func (s *mainState) bindProactiveRunner(ctx context.Context, client *dsh.Client) {
	s.assistantService().BindProactiveRunner(ctx, client)
}

func (s *mainState) stopProactiveMonitor() {
	s.assistantService().StopProactiveMonitor()
}

func (s *mainState) refreshWhaleSettingsUI() {
	s.assistantService().RefreshWhaleSettingsUI()
}

func (s *mainState) gateAssistantNavigation(m *frame.Model) {
	if m == nil {
		return
	}
	if !s.assistantService().GateNavigation(m.OnOpenAssistant != nil) {
		m.OnOpenAssistant = nil
	}
}

func (s *mainState) openWhaleFromSidebar(_ *mygo.Window) {
	s.assistantService().OpenWhaleFromSidebar()
}

func (s *mainState) assistantWhaleSettingsModel(ep harness.Endpoint) assistantui.WhaleSettingsModel {
	return s.assistantService().WhaleSettingsModel(ep)
}

func (s *mainState) assistantGroupMembers(sessionID string) (bool, []session.GroupMember) {
	return s.assistantService().GroupMembers(sessionID)
}

func (s *mainState) beginAssistantGroupReply(sessions *session.Store, draft *composer.DraftState, id, threadID string) {
	if draft == nil {
		return
	}
	s.assistantService().BeginGroupReply(sessions, draft, id, threadID)
}

func (s *mainState) clearFocusedBotNotifications() {
	s.assistantService().ClearFocusedBotNotifications()
}

func (s *mainState) observeWhaleNavLabel(ep harness.Endpoint) {
	s.assistantService().ObserveWhaleNavLabel(ep)
}

// --- assistantctl.HostDeps ---

type mainAssistantDeps struct{ s *mainState }

type assistantNoticeHandle struct{ n *mygo.Notification }

func (h assistantNoticeHandle) Show() error {
	if h.n == nil {
		return nil
	}
	return h.n.Show()
}

func (h assistantNoticeHandle) Close() {
	if h.n != nil {
		h.n.Close()
	}
}

func (h assistantNoticeHandle) OnClick(fn func()) {
	if h.n != nil {
		h.n.OnClick(fn)
	}
}

func (d mainAssistantDeps) asstHostDeps() assistantctl.HostDeps {
	return assistantctl.HostDeps{
		RunOnMain: mygo.RunOnMain,
		Window: func() any {
			d.s.mu.Lock()
			w := d.s.win
			d.s.mu.Unlock()
			if w == nil {
				return nil
			}
			return w
		},
		WindowUpdate: func(win any, fn func()) {
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
		},
		Invalidate: func(win any) {
			if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
				w.Invalidate()
			}
		},
		WindowStatus: func(win any) assistantctl.WindowState {
			w, ok := win.(*mygo.Window)
			if !ok || w == nil {
				return assistantctl.WindowState{}
			}
			if w.IsDestroyed() {
				return assistantctl.WindowState{Exists: true, Destroyed: true}
			}
			return assistantctl.WindowState{
				Exists:    true,
				Visible:   w.IsVisible(),
				Minimized: w.IsMinimized(),
				Focused:   w.IsFocused(),
			}
		},
		ShowWindow: func(win any) {
			if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
				w.Show()
			}
		},
		RestoreWindow: func(win any) {
			if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
				w.Restore()
			}
		},
		FocusWindow: func(win any) {
			if w, ok := win.(*mygo.Window); ok && w != nil && !w.IsDestroyed() {
				w.Focus()
			}
		},
		OpenFileDialog: func(win any, opts assistantctl.FileDialogOptions) ([]string, error) {
			w, _ := win.(*mygo.Window)
			var filters []mygo.FileFilter
			for _, f := range opts.Filters {
				filters = append(filters, mygo.FileFilter{Name: f.Name, Extensions: append([]string(nil), f.Extensions...)})
			}
			return mygo.Dialog.Open(mygo.OpenDialogOptions{
				Parent:   w,
				Title:    opts.Title,
				Multiple: opts.Multiple,
				Filters:  filters,
			})
		},
		WriteClipboard:             mygo.Clipboard.WriteText,
		SystemNotificationsSupport: systemNotificationsSupported,
		NewNotification: func(opts assistantctl.NotificationOptions) assistantctl.NotificationHandle {
			n := mygo.NewNotification(mygo.NotificationOptions{Title: opts.Title, Body: opts.Body, Group: opts.Group})
			if n == nil {
				return nil
			}
			return assistantNoticeHandle{n: n}
		},
		IsNotificationsDenied: func(err error) bool {
			return errors.Is(err, mygo.ErrNotificationsDenied)
		},
		CurrentClient: func() *dsh.Client {
			d.s.dshMu.Lock()
			c := d.s.dshClient
			d.s.dshMu.Unlock()
			return c
		},
		BootEndpoint: func() (harness.Endpoint, harness.State, int) {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.bootStatus.Endpoint, d.s.bootStatus.State, d.s.bootStatus.Generation
		},
		LifecycleContext: d.s.lifecycleContext,
		Sessions: func() *session.Store {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.sessions
		},
		Config: func() *config.Store {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.cfgStore
		},
		PluginRPCClient: func(ep harness.Endpoint) *pluginrpc.Client {
			return d.s.pluginRPCClient(ep)
		},
		DshHome:   paths.DshHome,
		VendorDir: paths.VendorDir,
		Quitting: func() bool {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.quitting
		},
		OtherSurfaceBlocking: func() bool {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.maintenanceOpen || d.s.settingsOpen || d.s.usageOpen || d.s.imOpen || d.s.marketOpen || d.s.scheduleOpen || d.s.remoteOpen || d.s.webviewOpen || d.s.browserPreviewOpen
		},
		PetSettingsActive: func() bool {
			d.s.mu.Lock()
			defer d.s.mu.Unlock()
			return d.s.settingsPetActiveLocked()
		},
		ClearResumePendingSession: func() {
			d.s.mu.Lock()
			d.s.resumePendingSession = ""
			d.s.mu.Unlock()
		},
		HasSessionDraftConflict: func(sid string) bool {
			d.s.mu.Lock()
			draft := d.s.composerDraftLocked(sid)
			conflict := draft.Text != "" || len(draft.Attachments) != 0 || draft.Sending || draft.Preparing > 0 || draft.EditSeq != 0
			d.s.mu.Unlock()
			return conflict
		},
		StageSessionDraft: func(sid, text string) bool {
			d.s.mu.Lock()
			draft := d.s.composerDraftLocked(sid)
			draft.Text = text
			d.s.mu.Unlock()
			draft.Client.Sync()
			return true
		},
		SubmitComposer: func(win any, sessions *session.Store, sid, cwd string, hostGen uint64) {
			w, _ := win.(*mygo.Window)
			d.s.mu.Lock()
			draft := d.s.composerDraftLocked(sid)
			d.s.mu.Unlock()
			d.s.submitComposer(w, sessions, draft, sid, cwd, 0, false, hostGen)
		},
		RevokePhoneGit:        d.s.revokePhoneGit,
		NotifyUser:            func(win any, text string) { w, _ := win.(*mygo.Window); d.s.notifyUser(w, text) },
		OpenWebViewHostPlugin: d.s.openWebViewHostPlugin,
		MarkWebViewOpen: func() {
			d.s.navigateMainRoute(routeWebView)
		},
		RevealMainConversation: d.s.revealMainConversation,
		SelectMainPanel:        d.s.selectMainPanel,
		SetAssistantFlag:       d.s.setAssistantFlag,
	}
}
