package assistantctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
	assistantui "whaleisle/internal/ui/assistant"
)

type proactivePanelState struct {
	service                *proactive.Service
	open, busy             bool
	loadError, actionError string
	title, description     string
	selected               string
	autonomy               int
	runner                 *proactive.Runner
	runnerError            string
	monitor                *proactive.Monitor
	monitorError           string
	sourceRepository       string
	sourceError            string
	sourceBusy             bool
	deliveryRoutes         []proactive.DeliveryRoute
	deliveryError          string
	deliveryBusy           bool
}

// ProactiveSnapshot returns a read-only copy of the proactive panel state.
func (s *Service) ProactiveSnapshot() ProactiveStatus {
	if s == nil {
		return ProactiveStatus{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.proactivePanel
	autonomy := p.autonomy
	if autonomy == 0 {
		autonomy = 1
	}
	return ProactiveStatus{
		Open:             p.open,
		Busy:             p.busy,
		SourceBusy:       p.sourceBusy,
		DeliveryBusy:     p.deliveryBusy,
		Title:            p.title,
		Description:      p.description,
		Selected:         p.selected,
		Autonomy:         autonomy,
		Error:            p.actionError,
		LoadError:        p.loadError,
		RunnerError:      p.runnerError,
		MonitorError:     p.monitorError,
		SourceRepository: p.sourceRepository,
		SourceError:      p.sourceError,
		DeliveryError:    p.deliveryError,
		RunnerActive:     p.runner != nil,
		MonitorActive:    p.monitor != nil,
	}
}

// SetProactiveDraft updates the proactive task title and description drafts.
func (s *Service) SetProactiveDraft(title, description string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if !s.proactivePanel.busy {
		s.proactivePanel.title = title
		s.proactivePanel.description = description
	}
	s.mu.Unlock()
}

// ProactiveDeliveryRunning reports whether the proactive runner currently has an
// active delivery worker (counted in local task protection).
func (s *Service) ProactiveDeliveryRunning() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	runner := s.proactivePanel.runner
	s.mu.Unlock()
	return runner != nil && runner.DeliveryRunning()
}

// FreezeProactiveAdmissions freezes new proactive runner admissions during task-
// protected shutdown/restart/import flows. Always returns a non-nil release func.
func (s *Service) FreezeProactiveAdmissions() func() {
	if s == nil {
		return func() {}
	}
	s.mu.Lock()
	runner := s.proactivePanel.runner
	s.mu.Unlock()
	if runner == nil {
		return func() {}
	}
	return runner.FreezeAdmissions()
}

// BindProactiveRunner attaches a Host-scoped proactive runner (or closes the
// previous runner when client is nil).
func (s *Service) BindProactiveRunner(ctx context.Context, client *dsh.Client) {
	if s == nil {
		return
	}
	host := s.safeHost()
	dshHome := host.DshHome()
	s.mu.Lock()
	previous := s.proactivePanel.runner
	s.proactivePanel.runner = nil
	s.proactivePanel.deliveryRoutes = nil
	s.proactivePanel.deliveryBusy = false
	s.proactivePanel.deliveryError = ""
	service := s.proactivePanel.service
	s.mu.Unlock()
	if previous != nil {
		previous.Close()
	}
	if client == nil || service == nil {
		return
	}
	runner := proactive.NewRunner(ctx, service, proactive.Host{Client: client}, dshHome, func(err error) {
		if ctx.Err() != nil {
			return
		}
		s.mu.Lock()
		if err != nil {
			s.proactivePanel.runnerError = err.Error()
		} else {
			s.proactivePanel.runnerError = ""
		}
		s.mu.Unlock()
		s.WakeProactiveMonitor(false)
		s.RefreshWhaleSettingsUI()
	})
	s.mu.Lock()
	if s.proactivePanel.service == service && ctx.Err() == nil {
		s.proactivePanel.runner = runner
		runner = nil
	}
	s.mu.Unlock()
	if runner != nil {
		runner.Close()
	}
}

// StopProactiveRunner closes any active Host-bound proactive runner.
func (s *Service) StopProactiveRunner() {
	s.BindProactiveRunner(context.Background(), nil)
}

// OpenProactive opens the durable proactive task storage under userData.
func (s *Service) OpenProactive(userData string) {
	if s == nil {
		return
	}
	service, err := proactive.Open(userData)
	s.AttachProactive(service, err)
}

// DispatchProactive routes install-control desktop proactive requests to the
// proactive service and notifies runner approval pauses when applicable.
func (s *Service) DispatchProactive(ctx context.Context, action string, payload json.RawMessage) (any, error) {
	if s == nil {
		return nil, errors.New("proactive task storage is not available")
	}
	s.mu.Lock()
	service, loadErr, runner := s.proactivePanel.service, s.proactivePanel.loadError, s.proactivePanel.runner
	s.mu.Unlock()
	if service == nil {
		if loadErr == "" {
			loadErr = "proactive task storage is not available"
		}
		return nil, errors.New(loadErr)
	}
	value, err := service.Dispatch(ctx, action, payload)
	if err == nil && action == "approval/request" && runner != nil {
		if approval, ok := value.(proactive.Approval); ok {
			err = runner.PauseForApproval(ctx, approval.TaskID)
		}
	}
	s.RefreshWhaleSettingsUI()
	return value, err
}

// ProactiveModel builds the UI model for the durable proactive task inbox.
func (s *Service) ProactiveModel() assistantui.ProactiveModel {
	if s == nil {
		return assistantui.ProactiveModel{}
	}
	host := s.safeHost()
	sessions := host.Sessions()
	dshHome := host.DshHome()
	s.mu.Lock()
	p := s.proactivePanel
	s.mu.Unlock()
	m := assistantui.ProactiveModel{
		Open:        p.open,
		Busy:        p.busy,
		Error:       p.actionError,
		LoadError:   p.loadError,
		Title:       p.title,
		Description: p.description,
		Selected:    p.selected,
	}
	m.DeliveryRoutes, m.DeliveryError, m.DeliveryBusy = append([]proactive.DeliveryRoute(nil), p.deliveryRoutes...), p.deliveryError, p.deliveryBusy
	m.OnDeliveryRefresh = s.RefreshProactiveDeliveryRoutes
	m.Autonomy = p.autonomy
	if m.Autonomy == 0 {
		m.Autonomy = 1
	}
	m.OnAutonomy = func(level int) {
		if level < 1 || level > 5 {
			return
		}
		s.mu.Lock()
		if !s.proactivePanel.busy {
			s.proactivePanel.autonomy = level
		}
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}
	if m.Error == "" {
		m.Error = p.runnerError
	}
	if m.Error == "" {
		m.Error = p.monitorError
	}
	m.SourceRepository, m.SourceError, m.SourceBusy = p.sourceRepository, p.sourceError, p.sourceBusy
	m.OnSourceEdit = func(value string) {
		s.mu.Lock()
		if !s.proactivePanel.sourceBusy {
			s.proactivePanel.sourceRepository = value
		}
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}
	m.OnSourceAdd = func() {
		s.mu.Lock()
		repo := s.proactivePanel.sourceRepository
		s.mu.Unlock()
		s.ConfigureProactiveSource(repo, true)
	}
	m.OnSourceToggle = func(repository string, enabled bool) {
		s.proactiveSourceMutation(func(service *proactive.Service) error {
			_, err := service.ConfigureSource(repository, enabled)
			return err
		}, false)
	}
	m.OnSourceRefresh = func(id string) {
		s.proactiveSourceMutation(func(service *proactive.Service) error { return service.RefreshSource(id) }, false)
	}
	m.OnNoticeOpen = s.OpenProactiveNotice
	m.OnNoticeRead = func(id string) {
		s.proactiveNoticeMutation(func(service *proactive.Service) error { return service.ReadNotice(id) })
	}
	m.OnNoticeRetry = func(id string) {
		s.proactiveNoticeMutation(func(service *proactive.Service) error { return service.RetryNotice(id) })
	}
	spec := proactive.RunSpec{HostHome: dshHome}
	if sessions != nil {
		spec.Cwd = sessions.WorkspaceCwd()
		for _, row := range sessions.Snapshot().Workspaces {
			if row.Path == spec.Cwd {
				spec.WorkspaceID = row.WorkspaceID
				break
			}
		}
	}
	m.CanRun, m.WorkingDirectory = p.runner != nil, spec.Cwd
	if p.service != nil {
		m.State = p.service.Snapshot()
	}
	m.OnToggle = func() {
		s.mu.Lock()
		s.proactivePanel.open = !s.proactivePanel.open
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}
	m.OnSelect = func(id string) {
		s.mu.Lock()
		s.proactivePanel.selected = id
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}
	m.OnEdit = func(field, value string) {
		s.mu.Lock()
		if !s.proactivePanel.busy {
			switch field {
			case "title":
				s.proactivePanel.title = value
			case "description":
				s.proactivePanel.description = value
			}
		}
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}
	// Disk commits run off the UI thread. The service serializes concurrent Host
	// proposals, while one UI mutation owns its draft until persistence succeeds.
	mutate := func(run func(*proactive.Service) (string, error), created bool) {
		s.mu.Lock()
		svc := s.proactivePanel.service
		if svc == nil || svc != p.service || s.proactivePanel.busy {
			s.mu.Unlock()
			return
		}
		s.proactivePanel.busy, s.proactivePanel.actionError = true, ""
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
		go func() {
			id, err := run(svc)
			s.mu.Lock()
			if s.proactivePanel.service == svc {
				s.proactivePanel.busy = false
				if err != nil {
					s.proactivePanel.actionError = err.Error()
				} else {
					if id != "" {
						s.proactivePanel.selected = id
					}
					if created {
						s.proactivePanel.title, s.proactivePanel.description = "", ""
					}
				}
			}
			s.mu.Unlock()
			s.RefreshWhaleSettingsUI()
		}()
	}
	m.OnCreate = func() {
		s.mu.Lock()
		title, description := s.proactivePanel.title, s.proactivePanel.description
		autonomy := s.proactivePanel.autonomy
		s.mu.Unlock()
		if autonomy == 0 {
			autonomy = 1
		}
		if strings.TrimSpace(title) == "" {
			return
		}
		mutate(func(service *proactive.Service) (string, error) {
			t, err := service.Create(proactive.CreateInput{Title: title, Description: description, Autonomy: autonomy, Source: "manual"})
			return t.ID, err
		}, true)
	}
	m.OnDelivery = func(taskID, routeID, text string) {
		if p.runner == nil {
			return
		}
		mutate(func(service *proactive.Service) (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_, err := p.runner.ProposeDelivery(ctx, taskID, routeID, text)
			return taskID, err
		}, false)
	}
	m.OnCancel = func(id string) {
		mutate(func(service *proactive.Service) (string, error) {
			if p.runner != nil {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				t, err := p.runner.StopTask(ctx, id)
				return t.ID, err
			}
			t, err := service.Transition(proactive.TransitionInput{TaskID: id, State: "cancelled", Actor: "user", Reason: "Cancelled in task inbox"})
			return t.ID, err
		}, false)
	}
	m.OnDecide = func(id, decision string) {
		mutate(func(service *proactive.Service) (string, error) {
			for _, a := range service.Snapshot().Outbox {
				if a.ID == id && a.Status == "pending" {
					t, err := service.Get(a.TaskID)
					if err != nil {
						return "", err
					}
					if a.ToolCall == nil && t.Execution != nil && t.Execution.Unsettled() {
						if p.runner == nil {
							return "", errors.New(whaleTaskText("连接原 Host 并停止执行后再决定。", "Connect the original Host and stop execution before deciding."))
						}
						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()
						if err := p.runner.PauseForApproval(ctx, a.TaskID); err != nil {
							return "", err
						}
					}
				}
			}
			a, err := service.Decide(id, decision, "")
			return a.TaskID, err
		}, false)
	}
	m.OnRun = func(id string, continuation bool) {
		if p.runner == nil {
			return
		}
		mutate(func(service *proactive.Service) (string, error) {
			selected := spec
			if continuation {
				t, err := service.Get(id)
				if err != nil {
					return "", err
				}
				if t.Execution != nil {
					selected = t.Execution.Spec
				}
			}
			t, err := p.runner.Start(id, selected, continuation)
			return t.ID, err
		}, false)
	}
	m.OnInspect = func(id string) {
		if p.runner != nil {
			p.runner.Observe(id)
		}
	}
	m.OnComplete = func(id string) {
		mutate(func(service *proactive.Service) (string, error) {
			t, err := service.Transition(proactive.TransitionInput{TaskID: id, State: "delivered", Actor: "user", Reason: "Result accepted by user"})
			return t.ID, err
		}, false)
	}
	m.OnSchedule = func(id, at, repeat string) {
		if p.runner == nil {
			return
		}
		mutate(func(service *proactive.Service) (string, error) {
			when, err := time.ParseInLocation("2006-01-02 15:04", strings.TrimSpace(at), time.Local)
			if err != nil {
				return "", errors.New(whaleTaskText("请输入本地时间，例如 2026-11-12 09:00。", "Enter local time, for example 2026-11-12 09:00."))
			}
			var interval *int64
			if strings.TrimSpace(repeat) != "" {
				minutes, err := strconv.ParseInt(strings.TrimSpace(repeat), 10, 64)
				if err != nil || minutes <= 0 || minutes > int64((1<<63-1)/time.Minute) {
					return "", errors.New(whaleTaskText("重复间隔须为正整数分钟。", "Repeat interval must be a positive whole number of minutes."))
				}
				ms := minutes * 60000
				interval = &ms
			}
			_, err = p.runner.Schedule(proactive.ScheduleInput{TaskID: id, At: when.Format(time.RFC3339), IntervalMS: interval}, spec)
			return id, err
		}, false)
	}
	m.OnCancelSchedule = func(id string) {
		mutate(func(service *proactive.Service) (string, error) {
			j, err := service.CancelSchedule(id)
			return j.TaskID, err
		}, false)
	}
	m.OnOpenSession = func(id string) {
		if sessions == nil {
			return
		}
		s.mu.Lock()
		s.open = false
		s.conversationGeneration++
		s.mu.Unlock()
		sessions.Select(id)
		s.RefreshWhaleSettingsUI()
	}
	return m
}

// RefreshProactiveDeliveryRoutes asynchronously fetches delivery routes from the active runner.
func (s *Service) RefreshProactiveDeliveryRoutes() {
	if s == nil {
		return
	}
	s.mu.Lock()
	runner := s.proactivePanel.runner
	if runner == nil || s.proactivePanel.deliveryBusy {
		s.mu.Unlock()
		return
	}
	s.proactivePanel.deliveryBusy = true
	s.proactivePanel.deliveryError = ""
	s.mu.Unlock()
	s.RefreshWhaleSettingsUI()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		routes, err := runner.DeliveryRoutes(ctx)
		s.mu.Lock()
		if s.proactivePanel.runner == runner {
			s.proactivePanel.deliveryBusy = false
			s.proactivePanel.deliveryRoutes = routes
			if err != nil {
				s.proactivePanel.deliveryError = err.Error()
			}
		}
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}()
}

// StartProactiveMonitor starts the background source/notice monitor if proactive storage is attached.
func (s *Service) StartProactiveMonitor() {
	if s == nil {
		return
	}
	if s.safeHost().Quitting() {
		return
	}
	s.mu.Lock()
	if s.proactivePanel.monitor != nil || s.proactivePanel.service == nil {
		s.mu.Unlock()
		return
	}
	service := s.proactivePanel.service
	monitor := proactive.NewMonitor(context.Background(), service, proactive.GitHubReader{}, s.DeliverProactiveNotice, func(err error) {
		s.mu.Lock()
		if err != nil {
			s.proactivePanel.monitorError = err.Error()
		} else {
			s.proactivePanel.monitorError = ""
		}
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	})
	s.proactivePanel.monitor = monitor
	s.mu.Unlock()
}

// StopProactiveMonitor stops and waits for the background proactive monitor.
func (s *Service) StopProactiveMonitor() {
	if s == nil {
		return
	}
	s.mu.Lock()
	monitor := s.proactivePanel.monitor
	s.proactivePanel.monitor = nil
	s.mu.Unlock()
	if monitor != nil {
		monitor.Close()
	}
}

// CancelProactiveMonitor cancels the proactive monitor without blocking the main thread on quit.
func (s *Service) CancelProactiveMonitor() {
	if s == nil {
		return
	}
	s.mu.Lock()
	monitor := s.proactivePanel.monitor
	s.mu.Unlock()
	if monitor != nil {
		monitor.Cancel()
	}
}

// WakeProactiveMonitor notifies the running monitor of source or delivery state changes.
func (s *Service) WakeProactiveMonitor(sourcesChanged bool) {
	if s == nil {
		return
	}
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

// OpenProactiveNotice reveals the main window, navigates to the task inbox, and marks the notice read.
func (s *Service) OpenProactiveNotice(notice proactive.Notice) {
	if s == nil {
		return
	}
	host := s.safeHost()
	quitting := host.Quitting()
	win := host.Window()
	s.mu.Lock()
	service := s.proactivePanel.service
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
	host.RevealMainConversation()
	host.SelectMainPanel("assistant")
	s.mu.Lock()
	s.proactivePanel.open = true
	s.proactivePanel.selected = notice.TaskID
	s.mu.Unlock()
	if win != nil {
		host.ShowWindow(win)
		host.RestoreWindow(win)
		host.FocusWindow(win)
	}
	go func() {
		err := service.ReadNotice(notice.ID)
		s.mu.Lock()
		if err != nil {
			s.proactivePanel.actionError = err.Error()
		}
		s.mu.Unlock()
		s.RefreshWhaleSettingsUI()
	}()
}

// DeliverProactiveNotice shows an OS toast when unfocused or keeps the notice in-app.
func (s *Service) DeliverProactiveNotice(ctx context.Context, notice proactive.Notice) (string, error) {
	if s == nil {
		return "unavailable", nil
	}
	host := s.safeHost()
	quitting := host.Quitting()
	if quitting || ctx.Err() != nil {
		return "", context.Canceled
	}
	win := host.Window()
	winSt := host.WindowStatus(win)
	if !winSt.Exists || winSt.Destroyed || !host.SystemNotificationsSupported() {
		return "unavailable", nil
	}
	if winSt.Focused {
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
	n := host.NewNotification(NotificationOptions{Title: title, Body: body, Group: "whaleisle-proactive"})
	if n == nil {
		return "unavailable", nil
	}
	n.OnClick(func() {
		if ctx.Err() == nil {
			host.RunOnMain(func() {
				s.OpenProactiveNotice(notice)
				n.Close()
			})
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

// ConfigureProactiveSource enables, adds, or disables a proactive source repository.
func (s *Service) ConfigureProactiveSource(repository string, enabled bool) {
	if s == nil {
		return
	}
	s.proactiveSourceMutation(func(service *proactive.Service) error {
		_, err := service.ConfigureSource(repository, enabled)
		return err
	}, true)
}

func (s *Service) proactiveNoticeMutation(action func(*proactive.Service) error) {
	s.mu.Lock()
	service := s.proactivePanel.service
	if service == nil || s.proactivePanel.sourceBusy {
		s.mu.Unlock()
		return
	}
	s.proactivePanel.sourceBusy = true
	s.proactivePanel.actionError = ""
	s.mu.Unlock()
	s.RefreshWhaleSettingsUI()
	go func() {
		err := action(service)
		s.mu.Lock()
		s.proactivePanel.sourceBusy = false
		if err != nil {
			s.proactivePanel.actionError = err.Error()
		}
		s.mu.Unlock()
		if err == nil {
			s.WakeProactiveMonitor(false)
		}
		s.RefreshWhaleSettingsUI()
	}()
}

func (s *Service) proactiveSourceMutation(action func(*proactive.Service) error, clearDraft bool) {
	s.mu.Lock()
	service := s.proactivePanel.service
	if service == nil || s.proactivePanel.sourceBusy {
		s.mu.Unlock()
		return
	}
	s.proactivePanel.sourceBusy = true
	s.proactivePanel.sourceError = ""
	s.mu.Unlock()
	s.RefreshWhaleSettingsUI()
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
			s.WakeProactiveMonitor(true)
		}
		s.RefreshWhaleSettingsUI()
	}()
}
