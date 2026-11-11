package main

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
	"whaleisle/internal/paths"
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

func (s *mainState) bindProactiveRunner(ctx context.Context, client *dsh.Client) {
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
	s.mu.Lock()
	if client != nil && service != nil {
		s.proactivePanel.runner = proactive.NewRunner(ctx, service, proactive.Host{Client: client}, paths.DshHome(), func(err error) {
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
			s.wakeProactiveMonitor(false)
			s.refreshWhaleSettingsUI()
		})
	}
	s.mu.Unlock()
}

func (s *mainState) openProactive(userData string) {
	service, err := proactive.Open(userData)
	s.mu.Lock()
	s.proactivePanel.service = service
	if err != nil {
		s.proactivePanel.loadError = err.Error()
	}
	s.mu.Unlock()
}

func (s *mainState) dispatchProactive(ctx context.Context, action string, payload json.RawMessage) (any, error) {
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
	s.refreshWhaleSettingsUI()
	return value, err
}

func (s *mainState) proactiveModel() assistantui.ProactiveModel {
	s.mu.Lock()
	p := s.proactivePanel
	sessions := s.sessions
	s.mu.Unlock()
	m := assistantui.ProactiveModel{Open: p.open, Busy: p.busy, Error: p.actionError, LoadError: p.loadError, Title: p.title, Description: p.description, Selected: p.selected}
	m.DeliveryRoutes, m.DeliveryError, m.DeliveryBusy = p.deliveryRoutes, p.deliveryError, p.deliveryBusy
	m.OnDeliveryRefresh = s.refreshProactiveDeliveryRoutes
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
		s.refreshWhaleSettingsUI()
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
		s.refreshWhaleSettingsUI()
	}
	m.OnSourceAdd = func() { s.configureProactiveSource(p.sourceRepository, true) }
	m.OnSourceToggle = func(repository string, enabled bool) {
		s.proactiveSourceMutation(func(service *proactive.Service) error {
			_, err := service.ConfigureSource(repository, enabled)
			return err
		}, false)
	}
	m.OnSourceRefresh = func(id string) {
		s.proactiveSourceMutation(func(service *proactive.Service) error { return service.RefreshSource(id) }, false)
	}
	m.OnNoticeOpen = s.openProactiveNotice
	m.OnNoticeRead = func(id string) {
		s.proactiveNoticeMutation(func(service *proactive.Service) error { return service.ReadNotice(id) })
	}
	m.OnNoticeRetry = func(id string) {
		s.proactiveNoticeMutation(func(service *proactive.Service) error { return service.RetryNotice(id) })
	}
	spec := proactive.RunSpec{HostHome: paths.DshHome()}
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
		s.refreshWhaleSettingsUI()
	}
	m.OnSelect = func(id string) {
		s.mu.Lock()
		s.proactivePanel.selected = id
		s.mu.Unlock()
		s.refreshWhaleSettingsUI()
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
		s.refreshWhaleSettingsUI()
	}
	// Disk commits run off the UI thread. The service serializes concurrent Host
	// proposals, while one UI mutation owns its draft until persistence succeeds.
	mutate := func(run func(*proactive.Service) (string, error), created bool) {
		s.mu.Lock()
		if p.service == nil || s.proactivePanel.service != p.service || s.proactivePanel.busy {
			s.mu.Unlock()
			return
		}
		s.proactivePanel.busy, s.proactivePanel.actionError = true, ""
		s.mu.Unlock()
		s.refreshWhaleSettingsUI()
		go func() {
			id, err := run(p.service)
			s.mu.Lock()
			if s.proactivePanel.service == p.service {
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
			s.refreshWhaleSettingsUI()
		}()
	}
	m.OnCreate = func() {
		if strings.TrimSpace(p.title) == "" {
			return
		}
		mutate(func(service *proactive.Service) (string, error) {
			t, err := service.Create(proactive.CreateInput{Title: p.title, Description: p.description, Autonomy: m.Autonomy, Source: "manual"})
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
		s.assistantOpen = false
		s.assistantConversationGeneration++
		s.mu.Unlock()
		sessions.Select(id)
		s.refreshWhaleSettingsUI()
	}
	return m
}
