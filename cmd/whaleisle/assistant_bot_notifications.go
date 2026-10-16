package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/botnotify"
	"whaleisle/internal/dsh"
	"whaleisle/internal/harness"
	assiststore "whaleisle/internal/store/assistant"
)

type botNotificationRun struct {
	window            *mygo.Window
	client            *dsh.Client
	endpoint          harness.Endpoint
	key               string
	cancel            context.CancelFunc
	tracker           botnotify.Tracker // UI-thread only below this line
	contactActivity   []botnotify.ContactActivity
	notices           map[botnotify.Ticket]*mygo.Notification
	pending           []botnotify.Notice
	deliveryBlocked   bool
	navigationMessage string
}

// The caller holds s.mu. Match the native Assistant page's mutually exclusive
// page gates using current state, not the previous frame's rendered snapshot.
// Keep this pure: native window getters synchronously marshal to the UI thread.
func (s *mainState) botNotificationPageActiveLocked() bool {
	return s.assistantOpen && !s.quitting && !s.launcherOpen && !s.settingsOpen && !s.usageOpen && !s.imOpen && !s.marketOpen && !s.scheduleOpen && !s.remoteOpen && !s.webviewOpen && !s.browserPreviewOpen
}

// No native getters are called while holding mainState/controller locks.
func (s *mainState) botNotificationCurrent(run *botNotificationRun) bool {
	if run == nil {
		return false
	}
	s.mu.Lock()
	current := s.botNotifications == run && s.win == run.window && s.botNotificationPageActiveLocked()
	asst := s.assistant
	s.mu.Unlock()
	if !current || run.window == nil || run.window.IsDestroyed() || asst == nil || asst.Snapshot().Mode != assiststore.ModeDshbot {
		return false
	}
	return s.assistantDshbotCurrent(run.endpoint, run.client)
}

// Called by the active Bot page lifecycle. OS hide/minimize does not retire
// this activity observer; page/account/Host departure and OnClosed still do.
// It also persists original read cursors when settings are writable.
func (s *mainState) observeBotNotifications(rendered bool) {
	s.mu.Lock()
	active := s.botNotificationPageActiveLocked()
	win := s.win
	ep := s.bootStatus.Endpoint
	asst := s.assistant
	s.mu.Unlock()
	if !rendered || !active || win == nil || win.IsDestroyed() || asst == nil || asst.Snapshot().Mode != assiststore.ModeDshbot {
		s.stopBotNotifications()
		return
	}
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if !s.assistantDshbotCurrent(ep, client) {
		s.stopBotNotifications()
		return
	}
	owner := assistantTaskScope(ep, client, "notification-catalog")
	key := fmt.Sprintf("%p\x00%s", win, owner.HostKey)
	s.mu.Lock()
	old := s.botNotifications
	s.mu.Unlock()
	if old != nil && old.key == key && s.botNotificationCurrent(old) {
		return
	}
	s.stopBotNotifications()
	ctx, cancel := context.WithCancel(context.Background())
	run := &botNotificationRun{window: win, client: client, endpoint: ep, key: key, cancel: cancel, notices: map[botnotify.Ticket]*mygo.Notification{}}
	s.mu.Lock()
	if !s.botNotificationPageActiveLocked() || s.win != win {
		s.mu.Unlock()
		cancel()
		return
	}
	s.botNotifications = run
	if s.botNotificationReadSlot == nil {
		s.botNotificationReadSlot = make(chan struct{}, 1)
	}
	readSlot := s.botNotificationReadSlot
	s.botNotificationMessage = "Bot 通知观察中：仅当前 Bot 页完整目录 / Watching this Bot page’s complete catalog"
	s.mu.Unlock()
	rpc := s.pluginRPCClient(ep)
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
		win.Update(func() {
			defer close(applied)
			if !s.botNotificationCurrent(run) || win.IsDestroyed() {
				return
			}
			if err != nil {
				run.contactActivity = nil
				run.tracker.Clear()
				run.pending = nil
				for ticket, n := range run.notices {
					n.Close()
					delete(run.notices, ticket)
				}
				s.botNotificationStatus(err.Error())
				return
			}
			run.contactActivity = append([]botnotify.ContactActivity(nil), snapshot.Contacts...)
			s.mu.Lock()
			denied := s.botNotificationDenied
			s.mu.Unlock()
			supported := mygo.NotificationsSupported() && !denied && !run.deliveryBlocked
			notices := run.tracker.Observe(snapshot, win.IsFocused(), supported)
			for ticket, n := range run.notices {
				if !run.tracker.Current(ticket) {
					n.Close()
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
			if !supported {
				s.botNotificationStatus("系统通知不可用或未获允许；Bot 页状态仍可查看 / System notifications unavailable or not allowed")
			} else if snapshot.Unavailable > 0 {
				s.botNotificationStatus(fmt.Sprintf("%d 个 Bot 状态暂不可用；其余继续观察 / Some Bot activity is unavailable; remaining items are monitored", snapshot.Unavailable) + run.navigationMessage)
			} else {
				s.botNotificationStatus("Bot 通知观察中：仅当前 Bot 页完整目录 / Watching this Bot page’s complete catalog" + run.navigationMessage)
			}
			s.deliverBotNotification(run)
		})
		select {
		case <-ctx.Done():
		case <-applied:
		}
	})
}
func (s *mainState) botNotificationStatus(message string) {
	s.mu.Lock()
	changed := s.botNotificationMessage != message
	s.botNotificationMessage = message
	win := s.win
	s.mu.Unlock()
	if changed && win != nil {
		win.Invalidate()
	}
}
func (s *mainState) botNotificationStatusText() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.botNotificationMessage
}

func (s *mainState) stopBotNotifications() {
	s.mu.Lock()
	run := s.botNotifications
	s.botNotifications = nil
	s.mu.Unlock()
	if run == nil {
		return
	}
	run.cancel()
	run.tracker.Clear()
	run.pending = nil
	for ticket, n := range run.notices {
		n.Close()
		delete(run.notices, ticket)
	}
}
func (s *mainState) clearFocusedBotNotifications() {
	s.mu.Lock()
	run := s.botNotifications
	s.mu.Unlock()
	if run == nil {
		return
	}
	// Some OS backends focus the window before delivering notification click.
	// Retain shown notification identity so that click can still select its
	// exact target; discard only unsubmitted reminders while foregrounded.
	run.pending = nil
}

func (s *mainState) deliverBotNotification(run *botNotificationRun) {
	if !s.botNotificationCurrent(run) || run.window.IsDestroyed() || run.window.IsFocused() {
		run.pending = nil
		return
	}
	s.mu.Lock()
	busy, denied := s.botNotificationShowing, s.botNotificationDenied
	s.mu.Unlock()
	if busy || denied || run.deliveryBlocked {
		return
	}
	for len(run.pending) > 0 {
		notice := run.pending[0]
		run.pending = run.pending[1:]
		if !run.tracker.Current(notice.Ticket) {
			continue
		}
		n := mygo.NewNotification(mygo.NotificationOptions{Title: notice.Name, Body: strings.Join(notice.Events, "\n"), Group: "whaleisle-bot"})
		run.notices[notice.Ticket] = n
		n.OnClick(func() {
			current := func() bool {
				return s.botNotificationCurrent(run) && run.notices[notice.Ticket] == n && run.tracker.Current(notice.Ticket)
			}
			if !current() {
				return
			}
			// Hidden/minimized windows retain this read-only observer. Restore
			// its exact window before selecting or revealing management UI.
			// Native callbacks may change page/account ownership at each step.
			run.window.Show()
			if !current() {
				return
			}
			run.window.Restore()
			if !current() {
				return
			}
			run.window.Focus()
			if !current() {
				return
			}
			if !run.window.IsVisible() || run.window.IsMinimized() {
				s.botNotificationStatus("无法显示通知目标窗口 / Could not restore the notification target window")
				return
			}
			// The latest accepted catalog/activity pair owns this exact item,
			// including rooms and hidden profiles. No cached/current selection.
			target, valid := run.tracker.Target(notice.Ticket)
			s.mu.Lock()
			asst := s.assistant
			s.mu.Unlock()
			if !valid || asst == nil {
				return
			}
			// Hidden or newly-created items need their exact management view;
			// never insert a selection that the normal roster cannot represent.
			s.mu.Lock()
			roster := s.dshbotRoster
			s.mu.Unlock()
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
				if !s.revealAssistantBotVisibility(run.endpoint, run.client, target.ID, target.SessionID) {
					return
				}
				run.navigationMessage = " · 通知目标：" + target.Name + "；请在隐藏/显示管理块核对 / Notification target: verify in visibility management"
				s.botNotificationStatus(run.navigationMessage)
			} else {
				asst.SelectDshbot(notice.Ticket.BotID, run.endpoint.Origin)
			}

			n.Close()
			delete(run.notices, notice.Ticket)
			run.window.Invalidate()
			// Navigation only: no bot/open, session prompt, task, answer, or routine run.
		})
		s.mu.Lock()
		s.botNotificationShowing = true
		s.mu.Unlock()
		go func() {
			// A worker can start after a same-frame page/focus/account change.
			var err error
			if !s.botNotificationCurrent(run) || run.window.IsDestroyed() || run.window.IsFocused() {
				err = context.Canceled
			} else {
				err = n.Show() // one outstanding native permission/submission
			}
			// Release the shared slot even when the old window cannot execute
			// Update anymore. A replacement cannot start a second permission job.
			s.mu.Lock()
			s.botNotificationShowing = false
			if err == mygo.ErrNotificationsDenied {
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
			currentRun.window.Update(func() {
				if !s.botNotificationCurrent(run) || !run.tracker.Current(notice.Ticket) || run.notices[notice.Ticket] != n || run.window.IsFocused() {
					n.Close()
					delete(run.notices, notice.Ticket)
				} else if err != nil {
					n.Close()
					delete(run.notices, notice.Ticket)
					if err != context.Canceled {
						run.deliveryBlocked = true
						run.pending = nil
						if err == mygo.ErrNotificationsDenied {
							s.mu.Lock()
							s.botNotificationDenied = true
							s.mu.Unlock()
						}
						s.botNotificationStatus("系统通知未能投递；请检查系统通知权限 / System notification delivery failed; check OS notification permission")
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
