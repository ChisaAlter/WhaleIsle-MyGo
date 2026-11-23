package assistantctl

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"whaleisle/internal/desktop/botnotify"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	"whaleisle/internal/state"
	assiststore "whaleisle/internal/store/assistant"
	"whaleisle/internal/ui/i18n"
)

type botNotificationRun struct {
	window   any
	client   *dsh.Client
	endpoint harness.Endpoint
	key      string
	cancel   context.CancelFunc

	mu                sync.Mutex
	tracker           botnotify.Tracker
	contactActivity   []botnotify.ContactActivity
	notices           map[botnotify.Ticket]NotificationHandle
	pending           []botnotify.Notice
	deliveryBlocked   bool
	navigationMessage string
}

func (r *botNotificationRun) snapshotContactActivity() []botnotify.ContactActivity {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]botnotify.ContactActivity(nil), r.contactActivity...)
}

func (r *botNotificationRun) closeAllNotices() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.tracker.Clear()
	r.pending = nil
	handles := make([]NotificationHandle, 0, len(r.notices))
	for _, h := range r.notices {
		if h != nil {
			handles = append(handles, h)
		}
	}
	r.notices = map[botnotify.Ticket]NotificationHandle{}
	r.mu.Unlock()
	for _, h := range handles {
		h.Close()
	}
}

func (s *Service) botNotificationPageActive() bool {
	host := s.safeHost()
	if host.OtherSurfaceBlocking() {
		return false
	}
	s.mu.Lock()
	open := s.open
	s.mu.Unlock()
	return open
}

func (s *Service) botNotificationCurrent(run *botNotificationRun) bool {
	if s == nil || run == nil {
		return false
	}
	host := s.safeHost()
	active := s.botNotificationPageActive()
	win := host.Window()
	s.mu.Lock()
	same := s.botNotifications == run
	asst := s.assistant
	s.mu.Unlock()
	if !same || !active || win != run.window || run.window == nil || asst == nil {
		return false
	}
	if host.WindowStatus(run.window).Destroyed || asst.Snapshot().Value.Mode != assiststore.ModeDshbot {
		return false
	}
	return s.DshbotCurrent(run.endpoint, run.client)
}

// ObserveBotNotifications manages the active Bot page's read-only activity poller.
// OS hide/minimize preserves the observer; page/account/Host departure and OnClosed retire it.
func (s *Service) ObserveBotNotifications(rendered bool) {
	if s == nil {
		return
	}
	host := s.safeHost()
	active := s.botNotificationPageActive()
	win := host.Window()
	ep, _, _ := host.BootEndpoint()
	asst := s.AssistantStore()
	winSt := host.WindowStatus(win)
	if !rendered || !active || win == nil || winSt.Destroyed || asst == nil || asst.Snapshot().Value.Mode != assiststore.ModeDshbot {
		s.StopBotNotifications()
		return
	}
	client := host.CurrentClient()
	if !s.DshbotCurrent(ep, client) {
		s.StopBotNotifications()
		return
	}
	owner := taskScope(ep, client, "notification-catalog")
	key := fmt.Sprintf("%p\x00%s", win, owner.HostKey)
	s.mu.Lock()
	old := s.botNotifications
	s.mu.Unlock()
	if old != nil && old.key == key && s.botNotificationCurrent(old) {
		return
	}
	s.StopBotNotifications()
	ctx, cancel := context.WithCancel(context.Background())
	run := &botNotificationRun{
		window:   win,
		client:   client,
		endpoint: ep,
		key:      key,
		cancel:   cancel,
		notices:  map[botnotify.Ticket]NotificationHandle{},
	}
	if !s.botNotificationPageActive() || host.Window() != win {
		cancel()
		return
	}
	s.mu.Lock()
	if !s.open {
		s.mu.Unlock()
		cancel()
		return
	}
	s.botNotifications = run
	if s.botNotificationReadSlot == nil {
		s.botNotificationReadSlot = make(chan struct{}, 1)
	}
	readSlot := s.botNotificationReadSlot
	s.botNotificationMessage = i18n.T(i18n.KeyCmdWatchingThisBotPageSCompleteCatalo)
	s.mu.Unlock()

	rpc := host.PluginRPCClient(ep)
	go botnotify.Poll(ctx, botnotify.Interval, botnotify.ReadTimeout, func(ctx context.Context) (botnotify.Snapshot, error) {
		select {
		case readSlot <- struct{}{}:
		case <-ctx.Done():
			return botnotify.Snapshot{}, ctx.Err()
		}
		defer func() { <-readSlot }()
		if !s.botNotificationCurrent(run) {
			cancel()
			return botnotify.Snapshot{}, context.Canceled
		}
		snapshot, err := rpc.DshbotNotificationSnapshot(ctx)
		if err == nil {
			err = s.initializeBotReadState(ctx, run, &snapshot)
		}
		return snapshot, err
	}, func(snapshot botnotify.Snapshot, err error) {
		applied := make(chan struct{})
		host.WindowUpdate(win, func() {
			defer close(applied)
			if !s.botNotificationCurrent(run) || host.WindowStatus(win).Destroyed {
				return
			}
			if err != nil {
				run.mu.Lock()
				run.contactActivity = nil
				run.mu.Unlock()
				run.closeAllNotices()
				s.botNotificationStatus(err.Error())
				return
			}
			supportedBase := host.SystemNotificationsSupported()
			focused := host.WindowStatus(win).Focused
			s.mu.Lock()
			denied := s.botNotificationDenied
			s.mu.Unlock()

			run.mu.Lock()
			run.contactActivity = append([]botnotify.ContactActivity(nil), snapshot.Contacts...)
			supported := supportedBase && !denied && !run.deliveryBlocked
			notices := run.tracker.Observe(snapshot, focused, supported)
			var stale []NotificationHandle
			for ticket, n := range run.notices {
				if !run.tracker.Current(ticket) {
					if n != nil {
						stale = append(stale, n)
					}
					delete(run.notices, ticket)
				}
			}
			var pending []botnotify.Notice
			for _, n := range run.pending {
				if run.tracker.Current(n.Ticket) {
					pending = append(pending, n)
				}
			}
			run.pending = append(pending, notices...)
			// At most one queued current notice per Bot, including simultaneous events.
			if len(run.pending) > botnotify.MaxBots {
				run.pending = run.pending[len(run.pending)-botnotify.MaxBots:]
			}
			navMsg := run.navigationMessage
			run.mu.Unlock()
			for _, n := range stale {
				n.Close()
			}
			if !supported {
				s.botNotificationStatus(i18n.T(i18n.KeyCmdSystemNotificationsUnavailableOrNot))
			} else if snapshot.Unavailable > 0 {
				s.botNotificationStatus(fmt.Sprintf(i18n.T(i18n.KeyCmdBotStatusUnavailable), snapshot.Unavailable) + navMsg)
			} else {
				s.botNotificationStatus(i18n.T(i18n.KeyCmdWatchingThisBotPageSCompleteCatalo) + navMsg)
			}
			s.deliverBotNotification(run)
		})
		select {
		case <-ctx.Done():
		case <-applied:
		}
	})
}

func (s *Service) botNotificationStatus(message string) {
	s.mu.Lock()
	changed := s.botNotificationMessage != message
	s.botNotificationMessage = message
	s.mu.Unlock()
	if changed {
		host := s.safeHost()
		host.Invalidate(host.Window())
	}
}

// BotNotificationStatusText returns the current Bot notification status banner text.
func (s *Service) BotNotificationStatusText() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.botNotificationMessage
}

// StopBotNotifications cancels the active Bot notification observer and closes
// any displayed OS notifications.
func (s *Service) StopBotNotifications() {
	if s == nil {
		return
	}
	s.mu.Lock()
	run := s.botNotifications
	s.botNotifications = nil
	s.mu.Unlock()
	if run == nil {
		return
	}
	run.cancel()
	run.closeAllNotices()
}

// ClearFocusedBotNotifications discards unsubmitted reminder notices when the window gains focus.
func (s *Service) ClearFocusedBotNotifications() {
	if s == nil {
		return
	}
	s.mu.Lock()
	run := s.botNotifications
	s.mu.Unlock()
	if run == nil {
		return
	}
	// Some OS backends focus the window before delivering notification click.
	// Retain shown notification identity so that click can still select its
	// exact target; discard only unsubmitted reminders while foregrounded.
	run.mu.Lock()
	run.pending = nil
	run.mu.Unlock()
}

func (s *Service) deliverBotNotification(run *botNotificationRun) {
	host := s.safeHost()
	if !s.botNotificationCurrent(run) {
		run.mu.Lock()
		run.pending = nil
		run.mu.Unlock()
		return
	}
	winSt := host.WindowStatus(run.window)
	if winSt.Destroyed || winSt.Focused {
		run.mu.Lock()
		run.pending = nil
		run.mu.Unlock()
		return
	}
	s.mu.Lock()
	busy, denied := s.botNotificationShowing, s.botNotificationDenied
	s.mu.Unlock()
	if busy || denied {
		return
	}
	for {
		run.mu.Lock()
		if run.deliveryBlocked || len(run.pending) == 0 {
			run.mu.Unlock()
			return
		}
		notice := run.pending[0]
		run.pending = run.pending[1:]
		if !run.tracker.Current(notice.Ticket) {
			run.mu.Unlock()
			continue
		}
		run.mu.Unlock()

		n := host.NewNotification(NotificationOptions{
			Title: notice.Name,
			Body:  strings.Join(notice.Events, "\n"),
			Group: "whaleisle-bot",
		})
		if n == nil {
			run.mu.Lock()
			run.deliveryBlocked = true
			run.pending = nil
			run.mu.Unlock()
			return
		}
		run.mu.Lock()
		run.notices[notice.Ticket] = n
		run.mu.Unlock()

		n.OnClick(func() {
			current := func() bool {
				if !s.botNotificationCurrent(run) {
					return false
				}
				run.mu.Lock()
				ok := run.notices[notice.Ticket] == n && run.tracker.Current(notice.Ticket)
				run.mu.Unlock()
				return ok
			}
			if !current() {
				return
			}
			// Hidden/minimized windows retain this read-only observer. Restore
			// its exact window before selecting or revealing management UI.
			// Native callbacks may change page/account ownership at each step.
			host.ShowWindow(run.window)
			if !current() {
				return
			}
			host.RestoreWindow(run.window)
			if !current() {
				return
			}
			host.FocusWindow(run.window)
			if !current() {
				return
			}
			st := host.WindowStatus(run.window)
			if !st.Visible || st.Minimized {
				s.botNotificationStatus(i18n.T(i18n.KeyCmdCouldNotRestoreTheNotificationTarge))
				return
			}
			run.mu.Lock()
			target, valid := run.tracker.Target(notice.Ticket)
			run.mu.Unlock()
			asst := s.AssistantStore()
			if !valid || asst == nil {
				return
			}
			roster := s.DshbotRoster()
			represented := false
			if roster != nil {
				snap := roster.Snapshot()
				if snap.Origin == run.endpoint.Origin {
					for _, row := range snap.Rows {
						if row.ID == target.ID && row.SessionID == target.SessionID && !row.Hidden {
							represented = true
							break
						}
					}
				}
			}
			if target.Hidden || !represented {
				if !s.RevealBotVisibility(run.endpoint, run.client, target.ID, target.SessionID) {
					return
				}
				navMsg := fmt.Sprintf(i18n.T(i18n.KeyCmdNotificationTarget), target.Name) + i18n.T(i18n.KeyCmdNotificationTargetVerifyInVisibility)
				run.mu.Lock()
				run.navigationMessage = navMsg
				run.mu.Unlock()
				s.botNotificationStatus(navMsg)
			} else {
				asst.SelectDshbot(notice.Ticket.BotID, run.endpoint.Origin)
			}

			n.Close()
			run.mu.Lock()
			delete(run.notices, notice.Ticket)
			run.mu.Unlock()
			host.Invalidate(run.window)
		})

		s.mu.Lock()
		s.botNotificationShowing = true
		s.mu.Unlock()
		go func() {
			var err error
			if !s.botNotificationCurrent(run) {
				err = context.Canceled
			} else {
				st := host.WindowStatus(run.window)
				if st.Destroyed || st.Focused {
					err = context.Canceled
				} else {
					err = n.Show()
				}
			}
			s.mu.Lock()
			s.botNotificationShowing = false
			if host.IsNotificationsDenied(err) {
				s.botNotificationDenied = true
			}
			currentRun := s.botNotifications
			s.mu.Unlock()
			if !s.botNotificationCurrent(run) {
				n.Close()
			}
			if currentRun == nil {
				return
			}
			host.WindowUpdate(currentRun.window, func() {
				run.mu.Lock()
				tracked := run.tracker.Current(notice.Ticket)
				sameHandle := run.notices[notice.Ticket] == n
				run.mu.Unlock()
				focused := host.WindowStatus(run.window).Focused
				if !s.botNotificationCurrent(run) || !tracked || !sameHandle || focused {
					n.Close()
					run.mu.Lock()
					delete(run.notices, notice.Ticket)
					run.mu.Unlock()
				} else if err != nil {
					n.Close()
					run.mu.Lock()
					delete(run.notices, notice.Ticket)
					if err != context.Canceled {
						run.deliveryBlocked = true
						run.pending = nil
					}
					run.mu.Unlock()
					if err != context.Canceled {
						if host.IsNotificationsDenied(err) {
							s.mu.Lock()
							s.botNotificationDenied = true
							s.mu.Unlock()
						}
						s.botNotificationStatus(i18n.T(i18n.KeyCmdSystemNotificationDeliveryFailedChec))
					}
				}
				if s.botNotificationCurrent(currentRun) {
					s.deliverBotNotification(currentRun)
				}
			})
		}()
		return
	}
}

func applyBotReadView(contacts []botnotify.ContactActivity, view pluginrpc.DshbotSettingsView) {
	for i := range contacts {
		for _, p := range view.Value.Items {
			if p.ID == contacts[i].ID && p.SessionID == contacts[i].SessionID {
				contacts[i].ReadInitialized = p.ReadInitialized
				if p.LastSeenSeq > contacts[i].LastSeenSeq {
					contacts[i].LastSeenSeq = p.LastSeenSeq
				}
				break
			}
		}
	}
}

func (s *Service) initializeBotReadState(ctx context.Context, run *botNotificationRun, snapshot *botnotify.Snapshot) error {
	host := s.safeHost()
	sessions := host.Sessions()
	sid := ""
	if sessions != nil {
		sid = sessions.SelectedSessionID()
	}
	candidates := botnotify.ReadCandidates(snapshot.Contacts, sid)
	if len(candidates) == 0 {
		return nil
	}
	var authority struct {
		Writable bool `json:"writable"`
	}
	if err := run.client.Call(ctx, "settings/describe", nil, &authority); err != nil {
		return err
	}
	if !authority.Writable || !s.botNotificationCurrent(run) {
		return nil
	}
	marks := make([]pluginrpc.DshbotReadMark, 0, len(candidates))
	for _, a := range candidates {
		marks = append(marks, pluginrpc.DshbotReadMark{ID: a.ID, ThroughSeq: a.ActivitySeq})
	}
	view, err := host.PluginRPCClient(run.endpoint).DshbotMarkRead(ctx, marks)
	if err == nil {
		applyBotReadView(snapshot.Contacts, view)
		snapshot.Revision = view.Revision
		host.WindowUpdate(run.window, func() {
			if !s.botNotificationCurrent(run) {
				return
			}
			if roster := s.DshbotRoster(); roster != nil {
				roster.SetFromDescribeForHost(pluginrpc.DshbotDescribeResult{View: view}, false, run.endpoint.Origin)
			}
		})
	}
	return err
}

// MarkOpenedBotRead advances the Host's read cursor after canonical Bot navigation.
func (s *Service) MarkOpenedBotRead(ep harness.Endpoint, client *dsh.Client, id, sid string) {
	if s == nil || client == nil {
		return
	}
	host := s.safeHost()
	rpc := host.PluginRPCClient(ep)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		current := func() bool {
			same := host.CurrentClient() == client
			bootEp, _, _ := host.BootEndpoint()
			sessions := host.Sessions()
			return same && bootEp == ep && sessions != nil && sessions.SelectedSessionID() == sid
		}
		if !current() {
			return
		}
		activity, err := rpc.DshbotActivityTyped(ctx, []string{id})
		if err != nil {
			return
		}
		var authority struct {
			Writable bool `json:"writable"`
		}
		if client.Call(ctx, "settings/describe", nil, &authority) != nil || !authority.Writable || !current() {
			return
		}
		for _, a := range activity.Entries {
			if a.BotID == id && a.SessionID == sid && a.Error == "" && a.ActivitySeq >= 0 {
				_, err = rpc.DshbotMarkRead(ctx, []pluginrpc.DshbotReadMark{{ID: id, ThroughSeq: a.ActivitySeq}})
				if err != nil {
					asst := s.AssistantStore()
					if current() && asst != nil {
						asst.SetLastAction(i18n.T(i18n.KeyCmdConversationOpenedReadMarkerFailed), state.FaultOf(err))
						host.Invalidate(host.Window())
					}
				}
				return
			}
		}
	}()
}
