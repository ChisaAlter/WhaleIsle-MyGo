package main

import (
	"context"
	"sync"
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/bridgealerts"
	"whaleisle/internal/desktop/components"
	"whaleisle/internal/paths"
	"whaleisle/internal/ui/launcher"
)

type bridgeAlertUI struct {
	noticesMu   sync.Mutex
	monitor     *bridgealerts.Monitor
	window      *mygo.Window
	cancel      context.CancelFunc
	notices     map[bridgealerts.Ticket]*mygo.Notification
	attempted   map[bridgealerts.Ticket]bool
	actionError string
}

func (s *mainState) startWhaleBridgeAlerts(win *mygo.Window) {
	s.closeWhaleBridgeAlerts()
	if win == nil || !components.WhaleBridgeManagementSupported() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &bridgeAlertUI{monitor: components.NewWhaleBridgeAlerts(paths.UserDataDir()), window: win, cancel: cancel, notices: map[bridgealerts.Ticket]*mygo.Notification{}, attempted: map[bridgealerts.Ticket]bool{}}
	s.mu.Lock()
	s.bridgeAlerts = run
	s.mu.Unlock()
	win.OnClosed(func() {
		s.mu.Lock()
		current := s.bridgeAlerts == run
		s.mu.Unlock()
		if current {
			s.closeWhaleBridgeAlerts()
		}
	})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			run.monitor.Poll()
			win.Update(func() { s.deliverWhaleBridgeAlerts(run) })
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *mainState) closeWhaleBridgeAlerts() {
	s.mu.Lock()
	run := s.bridgeAlerts
	s.bridgeAlerts = nil
	s.mu.Unlock()
	if run == nil {
		return
	}
	run.cancel()
	run.monitor.Close()
	// Dispose immediately, even when the application event loop is exiting.
	run.noticesMu.Lock()
	notices := make([]*mygo.Notification, 0, len(run.notices))
	for _, n := range run.notices {
		notices = append(notices, n)
	}
	clear(run.notices)
	clear(run.attempted)
	run.noticesMu.Unlock()
	for _, n := range notices {
		n.Close()
	}
}
func (run *bridgeAlertUI) ownsNotice(ticket bridgealerts.Ticket, n *mygo.Notification) bool {
	run.noticesMu.Lock()
	defer run.noticesMu.Unlock()
	return run.notices[ticket] == n
}
func (s *mainState) bridgeAlertUICurrent(run *bridgeAlertUI) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return run != nil && s.bridgeAlerts == run && s.win == run.window && !s.quitting && !run.window.IsDestroyed()
}
func (s *mainState) observeWhaleBridgeAlerts() {
	s.mu.Lock()
	run := s.bridgeAlerts
	s.mu.Unlock()
	if run == nil {
		return
	}
	go func() { run.monitor.Observe(); run.window.Update(func() { s.deliverWhaleBridgeAlerts(run) }) }()
}
func (s *mainState) deliverWhaleBridgeAlerts(run *bridgeAlertUI) {
	if !s.bridgeAlertUICurrent(run) {
		return
	}
	snapshot := run.monitor.Snapshot()
	run.noticesMu.Lock()
	for ticket, n := range run.notices {
		if !run.monitor.Current(ticket) {
			n.Close()
			delete(run.notices, ticket)
		}
	}
	for ticket := range run.attempted {
		if !run.monitor.Current(ticket) {
			delete(run.attempted, ticket)
		}
	}
	run.noticesMu.Unlock()
	for _, notice := range snapshot.Alerts {
		ticket := notice.Ticket
		run.noticesMu.Lock()
		attempted := run.attempted[ticket]
		if !attempted {
			run.attempted[ticket] = true
		}
		run.noticesMu.Unlock()
		if attempted || !run.monitor.Current(ticket) {
			continue
		}
		if !systemNotificationsSupported() {
			run.monitor.DeliveryFailed(ticket)
			run.window.FlashFrame(true)
			continue
		}
		n := mygo.NewNotification(mygo.NotificationOptions{Title: notice.Title, Body: notice.Body, Silent: true, Group: "whalebridge-quota"})
		run.noticesMu.Lock()
		run.notices[ticket] = n
		run.noticesMu.Unlock()
		n.OnClick(func() {
			if !s.bridgeAlertUICurrent(run) || !run.ownsNotice(ticket, n) || !run.monitor.Current(ticket) {
				return
			}
			s.openWhaleBridgeAlert(run, ticket)
		})
		go func() {
			if !s.bridgeAlertUICurrent(run) || !run.monitor.Current(ticket) {
				return
			}
			err := n.Show()
			if !s.bridgeAlertUICurrent(run) || !run.monitor.Current(ticket) {
				n.Close()
				return
			}
			run.window.Update(func() {
				if !s.bridgeAlertUICurrent(run) || !run.ownsNotice(ticket, n) || !run.monitor.Current(ticket) {
					n.Close()
					return
				}
				if err != nil {
					n.Close()
					run.noticesMu.Lock()
					delete(run.notices, ticket)
					run.noticesMu.Unlock()
					run.monitor.DeliveryFailed(ticket)
					run.window.FlashFrame(true)
				}
			})
		}()
	}
	if snapshot.Paused {
		run.window.FlashFrame(true)
	}
}
func (s *mainState) openWhaleBridgeAlert(run *bridgeAlertUI, ticket bridgealerts.Ticket) {
	if !s.bridgeAlertUICurrent(run) {
		return
	}
	go func() {
		err := run.monitor.Open(ticket)
		run.window.Update(func() {
			if !s.bridgeAlertUICurrent(run) {
				return
			}
			if err != nil {
				run.actionError = i18n.T(i18n.KeyCmdTheWhalebridgeInstanceChangedOrIsUnava)
			} else {
				run.actionError = ""
			}
		})
	}()
}
func (s *mainState) withWhaleBridgeAlerts(m launcher.Model) launcher.Model {
	s.mu.Lock()
	run := s.bridgeAlerts
	s.mu.Unlock()
	if !s.bridgeAlertUICurrent(run) {
		return m
	}
	m.BridgeAlerts = launcher.BridgeAlertsModel{Snapshot: run.monitor.Snapshot(), ActionError: run.actionError,
		OnOpen: func(t bridgealerts.Ticket) { s.openWhaleBridgeAlert(run, t) },
		OnContinue: func(t bridgealerts.Ticket) {
			if !s.bridgeAlertUICurrent(run) {
				return
			}
			go func() {
				err := run.monitor.Continue(t)
				run.window.Update(func() {
					if !s.bridgeAlertUICurrent(run) {
						return
					}
					if err != nil {
						run.actionError = i18n.T(i18n.KeyCmdTheWhalebridgeInstanceChangedReviewIts)
					} else {
						run.actionError = ""
					}
				})
			}()
		},
	}
	return m
}
