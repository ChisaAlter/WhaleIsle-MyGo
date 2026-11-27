package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	assistantctl "whaleisle/internal/app/assistant"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/errcopy"
	"whaleisle/internal/harness"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	assistantui "whaleisle/internal/ui/assistant"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/coss"
	"whaleisle/internal/ui/design"
	"whaleisle/internal/ui/frame"
	"whaleisle/internal/ui/i18n"
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

func (s *mainState) openWhaleFromSidebar(win *mygo.Window) {
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	ep, bootState := s.bootStatus.Endpoint, s.bootStatus.State
	sessions := s.sessions
	cfgStore := s.cfgStore
	if win == nil {
		win = s.win
	}
	s.mu.Unlock()
	if cfgStore == nil || !s.effectiveConfigFor(cfgStore).WhaleAssistantEnabled {
		s.assistantService().OpenWhaleFromSidebar()
		return
	}
	if bootState != harness.Ready || client == nil || sessions == nil {
		s.assistantService().OpenWhaleFromSidebar()
		return
	}

	// If the live Host already lists a healthy Whale session, switch directly
	// through the canonical service so the user gets immediate navigation.
	snap := sessions.Published().Value
	liveWhaleID := ""
	for _, item := range snap.Items {
		if item.Presentation != nil && item.Presentation.Owner == "dsh-whale:assistant" && strings.TrimSpace(item.SessionID) != "" {
			liveWhaleID = strings.TrimSpace(item.SessionID)
			break
		}
	}
	if liveWhaleID != "" && !(snap.SelectedID == liveWhaleID && snap.HistoryError != nil) {
		s.mu.Lock()
		s.whaleOpening = false
		s.whaleOpenErr = ""
		s.whaleOpenGen++
		s.mu.Unlock()
		s.assistantService().OpenWhaleFromSidebar()
		return
	}

	// Avoid selecting an unverified disk session ID before Host confirms or
	// creates it (which otherwise causes session/not-found and a blank canvas).
	s.revealMainConversation()
	s.mu.Lock()
	s.whaleOpening = true
	s.whaleOpenErr = ""
	s.whaleOpenGen++
	gen := s.whaleOpenGen
	s.mu.Unlock()
	if win != nil {
		win.Invalidate()
	}

	rpc := s.pluginRPCClient(ep)
	svc := s.assistantService()
	asst := svc.AssistantStore()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		result, err := rpc.WhaleEnsureTyped(ctx)
		sid := strings.TrimSpace(result.SessionID)
		if err == nil && sid == "" {
			err = errors.New(i18n.T(i18n.KeyCmdCanonicalSessionMissing))
		}
		apply := func() {
			s.dshMu.Lock()
			stillCurrent := s.dshClient == client
			s.dshMu.Unlock()
			s.mu.Lock()
			if !stillCurrent || s.whaleOpenGen != gen {
				s.mu.Unlock()
				return
			}
			s.whaleOpening = false
			if err != nil {
				s.whaleOpenErr = errcopy.Under(i18n.Pick("打开鲸鱼娘对话", "Opening Whale assistant"), err)
				s.mu.Unlock()
				if win != nil {
					win.Invalidate()
				}
				return
			}
			s.whaleOpenErr = ""
			s.resumePendingSession = ""
			s.mu.Unlock()
			s.revokePhoneGit()
			_, whaleName, whaleHome := svc.WhalePersistedInfo()
			sessions.EnsureSessionPresentation(sid, "dsh-whale:assistant", whaleName, whaleHome)
			// If this exact ID previously failed with session/not-found, clear
			// selection first so Select(sid) restarts history/follow cleanly.
			if cur := sessions.Published().Value; cur.SelectedID == sid && cur.HistoryError != nil {
				sessions.Select("")
			}
			sessions.Select(sid)
			if asst != nil {
				asst.SetSessionID(sid)
			}
			if !sessions.SelectionSummary().ListLoading {
				sessions.RefreshList(context.Background())
			}
			if win != nil {
				win.Invalidate()
			}
		}
		if win != nil && !win.IsDestroyed() {
			win.Update(apply)
		} else {
			apply()
		}
	}()
}

func (s *mainState) renderWhaleConversationStatus(c *ui.Context, win *mygo.Window) {
	s.mu.Lock()
	opening := s.whaleOpening
	openErr := s.whaleOpenErr
	sessions := s.sessions
	s.mu.Unlock()

	whaleHistoryBroken := false
	if sessions != nil {
		snap := sessions.Published().Value
		if snap.SelectedID != "" {
			isWhale := strings.HasPrefix(snap.SelectedID, "session-whale-")
			if !isWhale {
				for _, item := range snap.Items {
					if item.SessionID == snap.SelectedID && item.Presentation != nil && item.Presentation.Owner == "dsh-whale:assistant" {
						isWhale = true
						break
					}
				}
			}
			if isWhale && snap.HistoryError != nil && len(snap.Messages) == 0 {
				whaleHistoryBroken = true
			} else if !isWhale && openErr != "" && !opening {
				s.mu.Lock()
				s.whaleOpenErr = ""
				s.mu.Unlock()
				openErr = ""
			}
		}
	}
	if !opening && openErr == "" && !whaleHistoryBroken {
		return
	}
	t := design.For(c)
	ui.Column(c.Key("whale-conversation-status")).FillWidth().Padding(design.Sp(3), design.Sp(6)).Gap(design.Sp(2)).Children(func() {
		if opening {
			coss.Alert(c, "", i18n.Pick("正在初始化鲸鱼娘对话…", "Initializing Whale assistant conversation…"), coss.AlertOptions{
				Variant: coss.AlertInfo,
				Key:     "whale-opening-alert",
			})
			return
		}
		msg := openErr
		if msg == "" {
			msg = i18n.Pick("鲸鱼娘会话尚未就绪或已被重置，点击下方按钮重新初始化。", "The Whale assistant session is unavailable or was reset. Click below to reinitialize it.")
		}
		coss.Alert(c, "", msg, coss.AlertOptions{
			Variant: coss.AlertWarning,
			Key:     "whale-recovery-alert",
		})
		ui.Row(c).Gap(design.Sp(2)).AlignItems(ui.Center).Children(func() {
			if coss.Button(c, i18n.Pick("重新打开鲸鱼娘对话", "Retry Whale assistant"), coss.ButtonOptions{
				Variant: coss.ButtonOutline,
				Size:    coss.ButtonSM,
			}).Clicked() {
				s.openWhaleFromSidebar(win)
			}
			ui.Text(c, i18n.Pick("将自动同步或新建常驻对话。", "Syncs or creates the persistent assistant session automatically.")).FontSize(design.TextXS.Size).TextColor(t.MutedForeground)
		})
	})
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
			d.s.submitComposer(w, sessions, draft, sid, cwd, false, hostGen)
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
