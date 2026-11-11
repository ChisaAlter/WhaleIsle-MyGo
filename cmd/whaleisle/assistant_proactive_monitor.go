package main

import (
	"context"
	"fmt"

	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/proactive"
)

func (s *mainState) startProactiveMonitor() {
	s.mu.Lock()
	if s.proactivePanel.monitor != nil || s.proactivePanel.service == nil || s.quitting {
		s.mu.Unlock()
		return
	}
	service := s.proactivePanel.service
	s.proactivePanel.monitor = proactive.NewMonitor(context.Background(), service, proactive.GitHubReader{}, s.deliverProactiveNotice, func(err error) {
		s.mu.Lock()
		if err != nil {
			s.proactivePanel.monitorError = err.Error()
		} else {
			s.proactivePanel.monitorError = ""
		}
		s.mu.Unlock()
		s.refreshWhaleSettingsUI()
	})
	s.mu.Unlock()
}

func (s *mainState) stopProactiveMonitor() {
	s.mu.Lock()
	monitor := s.proactivePanel.monitor
	s.proactivePanel.monitor = nil
	s.mu.Unlock()
	if monitor != nil {
		monitor.Close()
	}
}

// Application OnQuit may run on the native thread. Do not wait there for a
// notification API whose own callback requires that thread to finish.
func (s *mainState) cancelProactiveMonitor() {
	s.mu.Lock()
	monitor := s.proactivePanel.monitor
	s.mu.Unlock()
	if monitor != nil {
		monitor.Cancel()
	}
}

func (s *mainState) wakeProactiveMonitor(sourcesChanged bool) {
	s.mu.Lock()
	monitor := s.proactivePanel.monitor
	s.mu.Unlock()
	if monitor != nil {
		if sourcesChanged {
			monitor.SourcesChanged()
		} else {
			monitor.Wake()
		}
	}
}

func (s *mainState) openProactiveNotice(notice proactive.Notice) {
	s.mu.Lock()
	service := s.proactivePanel.service
	win := s.win
	quitting := s.quitting
	s.mu.Unlock()
	if service == nil || quitting {
		return
	}
	found := false
	for _, current := range service.Notices() {
		if current.ID == notice.ID {
			notice = current
			found = true
			break
		}
	}
	if !found {
		return
	}
	s.revealMainConversation()
	s.selectMainPanel("assistant")
	s.mu.Lock()
	s.proactivePanel.open = true
	s.proactivePanel.selected = notice.TaskID
	s.mu.Unlock()
	if win != nil {
		win.Show()
		win.Restore()
		win.Focus()
	}
	go func() {
		err := service.ReadNotice(notice.ID)
		s.mu.Lock()
		if err != nil {
			s.proactivePanel.actionError = err.Error()
		}
		s.mu.Unlock()
		s.refreshWhaleSettingsUI()
	}()
}

func (s *mainState) deliverProactiveNotice(ctx context.Context, notice proactive.Notice) (string, error) {
	s.mu.Lock()
	win := s.win
	quitting := s.quitting
	s.mu.Unlock()
	if quitting || ctx.Err() != nil {
		return "", context.Canceled
	}
	if win == nil || win.IsDestroyed() || !systemNotificationsSupported() {
		return "unavailable", nil
	}
	if win.IsFocused() {
		return "in-app", nil
	}
	title, body := notice.Title, whaleTaskText("任务已有结果，打开收件箱核对。", "Task results are ready. Open the inbox to review.")
	if notice.Kind == "discovery" {
		body = fmt.Sprintf(whaleTaskText("发现 %s 项待处理事项。", "Found %s new items to review."), notice.Body)
	}
	// Full output remains in the durable feed, not in an OS lock-screen toast.
	if len([]rune(title)) > 120 {
		title = string([]rune(title)[:120]) + "…"
	}
	n := mygo.NewNotification(mygo.NotificationOptions{Title: title, Body: body, Group: "whaleisle-proactive"})
	n.OnClick(func() {
		if ctx.Err() == nil {
			mygo.RunOnMain(func() { s.openProactiveNotice(notice); n.Close() })
		}
	})
	// macOS may await notification consent. Quitting must be able to cancel
	// this observer without waiting for a user response in that system dialog.
	shown := make(chan error, 1)
	go func() { shown <- n.Show() }()
	select {
	case <-ctx.Done():
		n.Close()
		return "", ctx.Err()
	case err := <-shown:
		if err != nil {
			n.Close()
			return "failed", fmt.Errorf("%s", whaleTaskText("桌面提醒未显示；结果仍保留在收件箱。", "Desktop notification could not be shown; the result remains in the inbox."))
		}
		return "shown", nil
	}
}

func (s *mainState) configureProactiveSource(repository string, enabled bool) {
	s.proactiveSourceMutation(func(service *proactive.Service) error {
		_, err := service.ConfigureSource(repository, enabled)
		return err
	}, true)
}

func (s *mainState) proactiveNoticeMutation(action func(*proactive.Service) error) {
	s.mu.Lock()
	service := s.proactivePanel.service
	if service == nil || s.proactivePanel.sourceBusy {
		s.mu.Unlock()
		return
	}
	s.proactivePanel.sourceBusy = true
	s.proactivePanel.actionError = ""
	s.mu.Unlock()
	s.refreshWhaleSettingsUI()
	go func() {
		err := action(service)
		s.mu.Lock()
		s.proactivePanel.sourceBusy = false
		if err != nil {
			s.proactivePanel.actionError = err.Error()
		}
		s.mu.Unlock()
		if err == nil {
			s.wakeProactiveMonitor(false)
		}
		s.refreshWhaleSettingsUI()
	}()
}

func (s *mainState) proactiveSourceMutation(action func(*proactive.Service) error, clearDraft bool) {
	s.mu.Lock()
	service := s.proactivePanel.service
	if service == nil || s.proactivePanel.sourceBusy {
		s.mu.Unlock()
		return
	}
	s.proactivePanel.sourceBusy = true
	s.proactivePanel.sourceError = ""
	s.mu.Unlock()
	s.refreshWhaleSettingsUI()
	go func() {
		err := action(service)
		s.mu.Lock()
		s.proactivePanel.sourceBusy = false
		if err != nil {
			s.proactivePanel.sourceError = err.Error()
		} else if clearDraft {
			s.proactivePanel.sourceRepository = ""
		}
		s.mu.Unlock()
		if err == nil {
			s.wakeProactiveMonitor(true)
		}
		s.refreshWhaleSettingsUI()
	}()
}
