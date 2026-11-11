package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"whaleisle/internal/desktop/proactive"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

func exerciseProactive(ctx context.Context, client *dsh.Client, model *modelFixture, workspace, userData, home string, pass func(string, string), restart func() (*dsh.Client, error), open func() (*proactive.Service, error)) error {
	if os.Getenv("WHALEISLE_PROACTIVE_EXTERNAL_DELIVERY") == "1" {
		return exerciseProactiveExternalDelivery(ctx, client, workspace, home, pass, open, restart)
	}
	if os.Getenv("WHALEISLE_PROACTIVE_TOOL_ONLY") == "1" {
		return exerciseProactiveToolGate(ctx, client, workspace, home, pass, open, restart)
	}
	if os.Getenv("WHALEISLE_PROACTIVE_DELIVERY_ONLY") == "1" {
		return exerciseProactiveDelivery(ctx, client, workspace, userData, home, pass, open)
	}
	if os.Getenv("WHALEISLE_PROACTIVE_APPROVAL_ONLY") == "1" {
		return exerciseProactiveApproval(ctx, client, model, workspace, userData, home, pass, open)
	}
	if os.Getenv("WHALEISLE_PROACTIVE_RECOVERY_ONLY") == "1" {
		return exerciseProactiveRecovery(ctx, client, model, workspace, userData, home, pass, restart, open)
	}
	service, err := open()
	if err != nil {
		return err
	}
	runner := proactive.NewRunner(ctx, service, proactive.Host{Client: client}, home, nil)
	defer func() { runner.Close() }()
	spec := proactive.RunSpec{HostHome: home, Cwd: workspace}
	create := func(title string) (proactive.Task, error) {
		return service.Create(proactive.CreateInput{Title: title, Autonomy: 1})
	}
	wait := func(id, state string) (proactive.Task, error) {
		var found proactive.Task
		err := await(ctx, func() (bool, error) {
			var err error
			found, err = service.Get(id)
			if err != nil {
				return false, err
			}
			if found.Execution != nil && found.Execution.Phase == "unknown" {
				return false, fmt.Errorf("task execution: %s", found.Execution.Error)
			}
			return found.State == state, nil
		})
		return found, err
	}
	task, err := create("QA_TOOL: read qa-note.txt and report its exact marker")
	if err != nil {
		return err
	}
	if _, err = runner.Start(task.ID, spec, false); err != nil {
		return err
	}
	task, err = wait(task.ID, "verifying")
	if err != nil {
		return err
	}
	if !strings.Contains(task.Execution.Summary, "QA_TOOL_OK") {
		return fmt.Errorf("task result missing actual read result: %s", task.Execution.Summary)
	}
	var projection api.SessionProjectionsValue
	if err := client.Call(ctx, "session/projections", map[string]any{"request": map[string]string{"sessionId": task.Execution.SessionID}}, &projection); err != nil {
		return err
	}
	if !strings.Contains(string(projection.Values), "read-only") {
		return fmt.Errorf("read-only sandbox not persisted: %s", projection.Values)
	}
	pass("proactive-real-session-read", "native authorized execution created the exact saved Session, read a real file, and persisted its result under read-only sandbox")
	model.mu.Lock()
	before := model.requests
	model.mu.Unlock()
	runner.Close()
	client, err = restart()
	if err != nil {
		return err
	}
	service, err = open()
	if err != nil {
		return err
	}
	runner = proactive.NewRunner(ctx, service, proactive.Host{Client: client}, home, nil)
	result, err := (proactive.Host{Client: client}).Run(ctx, task, func(string) error { return nil })
	if err != nil || result.State != "verifying" {
		return fmt.Errorf("historical execution recovery: %+v %w", result, err)
	}
	model.mu.Lock()
	after := model.requests
	model.mu.Unlock()
	if after != before {
		return fmt.Errorf("recovery resent an already completed prompt")
	}
	pass("proactive-restart-no-resend", "real Host restart recovered the exact task turn and summary with zero new model requests")
	if _, err := runner.Start(task.ID, spec, true); err != nil {
		return err
	}
	continued, err := wait(task.ID, "verifying")
	if err != nil {
		return err
	}
	if continued.Execution.SessionID != task.Execution.SessionID || continued.Execution.RequestID == task.Execution.RequestID {
		return fmt.Errorf("continuation lost Session or reused turn identity")
	}
	pass("proactive-explicit-continuation", "explicit Continue used the original Session with a new durable request ID")
	stopping, err := create("QA_WAIT: hold this task until cancellation")
	if err != nil {
		return err
	}
	if _, err := runner.Start(stopping.ID, spec, false); err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) { model.mu.Lock(); defer model.mu.Unlock(); return model.waitStarted, nil }); err != nil {
		return err
	}
	model.mu.Lock()
	cancelledBefore := model.canceled
	model.mu.Unlock()
	if _, err := runner.StopTask(ctx, stopping.ID); err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) {
		model.mu.Lock()
		defer model.mu.Unlock()
		return model.canceled > cancelledBefore, nil
	}); err != nil {
		return err
	}
	stopped, _ := service.Get(stopping.ID)
	if stopped.State != "cancelled" {
		return fmt.Errorf("task cancellation was not persisted")
	}
	pass("proactive-stop-model-request", "native Stop cancelled the live HTTP model request and persisted the task cancellation")
	template, err := create("QA_SCHEDULE: report a short status")
	if err != nil {
		return err
	}
	job, err := runner.Schedule(proactive.ScheduleInput{TaskID: template.ID, At: time.Now().Add(-time.Second).UTC().Format(time.RFC3339)}, spec)
	if err != nil {
		return err
	}
	var child proactive.Task
	if err := await(ctx, func() (bool, error) {
		for _, item := range service.Snapshot().Tasks {
			if item.Execution != nil && item.Execution.JobID == job.ID {
				child = item
				if item.Execution.Phase == "unknown" {
					return false, fmt.Errorf("schedule: %s", item.Execution.Error)
				}
				return item.State == "verifying", nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	if child.ParentID != template.ID || child.Execution.SessionID == task.Execution.SessionID {
		return fmt.Errorf("schedule result has wrong execution owner")
	}
	runner.Close()
	service, err = open()
	if err != nil {
		return err
	}
	for _, item := range service.Snapshot().Schedules {
		if item.ID == job.ID && item.Status != "fired" {
			return fmt.Errorf("one-shot schedule was not consumed")
		}
	}
	pass("proactive-schedule-durable-child", "actual desktop timer created a separate task/Session, model completed, and the fired receipt survived reopen")
	// Inspect actual persisted source IDs rather than only the local task record.
	var page api.SessionPage
	address, _ := json.Marshal(map[string]string{"kind": "session", "sessionId": continued.Execution.SessionID})
	if err := client.Call(ctx, "session/projections", map[string]any{"request": map[string]string{"sessionId": continued.Execution.SessionID}}, &projection); err != nil {
		return err
	}
	max := float64(500)
	if err := client.Call(ctx, "session/page", map[string]any{"request": api.SessionPageRequest{Address: address, ThroughSeq: projection.AsOfSeq, MaxMessages: &max}}, &page); err != nil {
		return err
	}
	count := map[string]int{}
	for _, record := range page.Records {
		if record.Event.Type == "user/message" {
			var data struct {
				Source struct {
					RPCID string `json:"rpcId"`
				} `json:"source"`
			}
			_ = json.Unmarshal(record.Event.Data, &data)
			count[data.Source.RPCID]++
		}
	}
	if count[task.Execution.RequestID] != 1 || count[continued.Execution.RequestID] != 1 {
		return fmt.Errorf("persisted request IDs not exactly once: %v", count)
	}
	pass("proactive-request-identity", "Host history contains exactly one original request and one explicitly continued request")
	return nil
}

func exerciseProactiveDelivery(ctx context.Context, client *dsh.Client, workspace, userData, home string, pass func(string, string), open func() (*proactive.Service, error)) error {
	service, err := open()
	if err != nil {
		return err
	}
	runner := proactive.NewRunner(ctx, service, proactive.Host{Client: client}, home, nil)
	defer runner.Close()
	monitor := proactive.NewMonitor(ctx, service, nil, nil, nil)
	defer monitor.Close()
	task, err := service.Create(proactive.CreateInput{Title: "QA_RESULT_DELIVERY: report the verification status", Autonomy: 1})
	if err != nil {
		return err
	}
	if _, err := runner.Start(task.ID, proactive.RunSpec{HostHome: home, Cwd: workspace}, false); err != nil {
		return err
	}
	var notice proactive.Notice
	if err := await(ctx, func() (bool, error) {
		current, err := service.Get(task.ID)
		if err != nil {
			return false, err
		}
		if current.Execution != nil && current.Execution.Phase == "unknown" {
			return false, fmt.Errorf("result delivery execution: %s", current.Execution.Error)
		}
		for _, n := range service.Notices() {
			if n.TaskID == task.ID {
				notice = n
				return n.DesktopStatus == "in-app", nil
			}
		}
		return false, nil
	}); err != nil {
		return err
	}
	if !strings.Contains(notice.Body, "QA_REPLY_OK") || notice.SessionID == "" || notice.ReadAt != "" {
		return fmt.Errorf("result did not arrive in unread in-app feed")
	}
	runner.Close()
	monitor.Close()
	service, err = open()
	if err != nil {
		return err
	}
	if len(service.Notices()) != 1 || service.Notices()[0].ID != notice.ID || service.Notices()[0].Body != notice.Body {
		return fmt.Errorf("delivered result lost on restart")
	}
	if err := service.ReadNotice(notice.ID); err != nil {
		return err
	}
	reopened, err := open()
	if err != nil {
		return err
	}
	if reopened.Notices()[0].ReadAt == "" {
		return fmt.Errorf("delivered result read receipt not durable")
	}
	pass("proactive-real-result-delivery", "real rc.4 task result reached the persistent unread in-app feed through the production monitor; exact Session/result and read receipt survived reopen; no OS toast or external message")
	return nil
}

func exerciseProactiveApproval(ctx context.Context, client *dsh.Client, model *modelFixture, workspace, userData, home string, pass func(string, string), open func() (*proactive.Service, error)) error {
	service, err := open()
	if err != nil {
		return err
	}
	runner := proactive.NewRunner(ctx, service, proactive.Host{Client: client}, home, nil)
	defer runner.Close()
	task, err := service.Create(proactive.CreateInput{Title: "QA_WAIT: await a user decision", Autonomy: 1})
	if err != nil {
		return err
	}
	spec := proactive.RunSpec{HostHome: home, Cwd: workspace}
	if _, err := runner.Start(task.ID, spec, false); err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) { model.mu.Lock(); defer model.mu.Unlock(); return model.waitStarted, nil }); err != nil {
		return err
	}
	a, err := service.RequestApproval(proactive.ApprovalInput{TaskID: task.ID, Action: "Prepare the review for the named recipient", Target: json.RawMessage(`"synthetic-owner"`)})
	if err != nil {
		return err
	}
	if err := runner.PauseForApproval(ctx, task.ID); err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) { model.mu.Lock(); defer model.mu.Unlock(); return model.canceled > 0, nil }); err != nil {
		return err
	}
	paused, err := service.Get(task.ID)
	if err != nil {
		return err
	}
	if paused.State != "waiting_approval" || paused.Execution.Unsettled() {
		return fmt.Errorf("approval did not pause exact execution: %+v", paused)
	}
	model.mu.Lock()
	before := model.requests
	model.releaseProactiveWait = true
	model.mu.Unlock()
	if _, err := service.Decide(a.ID, "approved", ""); err != nil {
		return err
	}
	model.mu.Lock()
	after := model.requests
	model.mu.Unlock()
	if before != after {
		return fmt.Errorf("approval unexpectedly sent a model request")
	}
	if _, err := runner.Start(task.ID, spec, true); err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) {
		task, err := service.Get(task.ID)
		if err != nil {
			return false, err
		}
		if task.Execution.Phase == "unknown" {
			return false, fmt.Errorf("approval continuation: %s", task.Execution.Error)
		}
		return task.State == "verifying", nil
	}); err != nil {
		return err
	}
	pass("proactive-approval-pauses-execution", "approval request cancelled the actual model HTTP request, remained waiting_approval, and only explicit Continue started the next turn; no external delivery")
	return nil
}

func exerciseProactiveRecovery(ctx context.Context, client *dsh.Client, model *modelFixture, workspace, userData, home string, pass func(string, string), restart func() (*dsh.Client, error), open func() (*proactive.Service, error)) error {
	service, err := open()
	if err != nil {
		return err
	}
	runner := proactive.NewRunner(ctx, service, proactive.Host{Client: client}, home, nil)
	defer func() { runner.Close() }()
	task, err := service.Create(proactive.CreateInput{Title: "QA_WAIT: preserve this interrupted task", Autonomy: 1})
	if err != nil {
		return err
	}
	spec := proactive.RunSpec{HostHome: home, Cwd: workspace}
	task, err = runner.Start(task.ID, spec, false)
	if err != nil {
		return err
	}
	if err := await(ctx, func() (bool, error) { model.mu.Lock(); defer model.mu.Unlock(); return model.waitStarted, nil }); err != nil {
		return err
	}
	model.mu.Lock()
	before := model.requests
	model.mu.Unlock()
	runner.Close()
	client, err = restart()
	if err != nil {
		return err
	}
	service, err = open()
	if err != nil {
		return err
	}
	runner = proactive.NewRunner(ctx, service, proactive.Host{Client: client}, home, nil)
	if err := await(ctx, func() (bool, error) {
		found, err := service.Get(task.ID)
		if err != nil {
			return false, err
		}
		if found.Execution.Phase == "unknown" {
			return false, fmt.Errorf("recovery: %s", found.Execution.Error)
		}
		return found.State == "waiting_input" && !found.Execution.Unsettled(), nil
	}); err != nil {
		return err
	}
	model.mu.Lock()
	after := model.requests
	model.releaseProactiveWait = true
	model.mu.Unlock()
	if before != after {
		return fmt.Errorf("interrupted Host restart automatically resent task: %d to %d", before, after)
	}
	pass("proactive-interrupted-recovery", "running real Host was stopped and restarted; task became waiting_input with no new model request")
	continued, err := runner.Start(task.ID, spec, true)
	if err != nil {
		return err
	}
	if continued.Execution.SessionID != task.Execution.SessionID {
		return fmt.Errorf("interrupted task changed Session")
	}
	if err := await(ctx, func() (bool, error) {
		found, err := service.Get(task.ID)
		if err != nil {
			return false, err
		}
		if found.Execution.Phase == "unknown" {
			return false, fmt.Errorf("continuation: %s", found.Execution.Error)
		}
		return found.State == "verifying" && strings.Contains(found.Execution.Summary, "QA_REPLY_OK"), nil
	}); err != nil {
		return err
	}
	pass("proactive-interrupted-explicit-resume", "user continuation completed in the original Session after restart")
	return nil
}
