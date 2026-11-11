package proactive

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
	"whaleisle/internal/dsh"
)

// Runner belongs to one connected desktop Host. Closing a panel does not stop
// it; losing that Host cancels observation and leaves durable execution IDs.
type Runner struct {
	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	service         *Service
	host            Host
	home            string
	active          map[string]context.CancelFunc
	workers         map[string]chan struct{}
	notify          func(error)
	done            chan struct{}
	frozen          int
	deliveryRunning bool
}

func NewRunner(parent context.Context, service *Service, host Host, hostHome string, notify func(error)) *Runner {
	ctx, cancel := context.WithCancel(parent)
	r := &Runner{ctx: ctx, cancel: cancel, service: service, host: host, home: hostHome, active: map[string]context.CancelFunc{}, workers: map[string]chan struct{}{}, notify: notify, done: make(chan struct{})}
	go r.loop()
	return r
}

func (r *Runner) changed(err error) {
	if r.notify != nil {
		r.notify(err)
	}
}

func (r *Runner) loop() {
	defer close(r.done)
	if err := r.service.recoverDeliveries(r.home); err != nil {
		r.changed(err)
		return
	}
	for _, task := range r.service.Snapshot().Tasks {
		if task.Execution != nil && task.Execution.Spec.HostHome == r.home && task.Execution.Unsettled() && !task.Terminal() {
			if task.ApprovalID != "" && !r.service.IsToolApproval(task.ApprovalID) {
				r.changed(r.PauseForApproval(r.ctx, task.ID))
				continue
			}
			r.Observe(task.ID)
		}
	}
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		select {
		case <-r.ctx.Done():
			return
		case at := <-timer.C:
			r.mu.Lock()
			if r.frozen > 0 {
				r.mu.Unlock()
				continue
			}
			tasks, err := r.service.claimDue(r.home, at)
			r.mu.Unlock()
			if err != nil {
				r.changed(err)
				return
			}
			for _, task := range tasks {
				r.Observe(task.ID)
			}
			if len(tasks) > 0 {
				r.changed(nil)
			}
			r.dispatchDeliveries()
		}
	}
}

func (r *Runner) Close() {
	r.cancel()
	<-r.done
	r.mu.Lock()
	workers := make([]chan struct{}, 0, len(r.workers))
	for _, done := range r.workers {
		workers = append(workers, done)
	}
	r.mu.Unlock()
	for _, done := range workers {
		<-done
	}
}

func (r *Runner) Start(id string, spec RunSpec, continuation bool) (Task, error) {
	if r.ctx.Err() != nil {
		return Task{}, r.ctx.Err()
	}
	if spec.HostHome != r.home {
		return Task{}, fmt.Errorf("task belongs to another Host")
	}
	r.mu.Lock()
	if r.frozen > 0 {
		r.mu.Unlock()
		return Task{}, errors.New("desktop lifecycle operation is in progress")
	}
	if r.active[id] != nil {
		r.mu.Unlock()
		return Task{}, errors.New("the previous execution is still settling; try again when it finishes")
	}
	task, err := r.service.PrepareExecution(id, spec, continuation)
	r.mu.Unlock()
	if err != nil {
		return Task{}, err
	}
	r.Observe(id)
	r.changed(nil)
	return task, nil
}

// FreezeAdmissions stops the local scheduler while the existing Host lifecycle
// gate inspects/drains work. Cancelling the lifecycle action restores it.
func (r *Runner) FreezeAdmissions() func() {
	r.mu.Lock()
	r.frozen++
	r.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { r.mu.Lock(); r.frozen--; r.mu.Unlock() }) }
}

func (r *Runner) PauseForApproval(ctx context.Context, id string) error {
	task, err := r.service.Get(id)
	if err != nil {
		return err
	}
	if task.Execution == nil || !task.Execution.Unsettled() {
		return nil
	}
	if task.Execution.Spec.HostHome != r.home {
		return errors.New("approval execution belongs to another Host")
	}
	r.mu.Lock()
	if cancel := r.active[id]; cancel != nil {
		cancel()
	}
	done := r.workers[id]
	r.mu.Unlock()
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if err := r.host.Stop(ctx, task.Execution.SessionID); err != nil {
		var remote *dsh.Error
		if !errors.As(err, &remote) || remote.Code != "session/not-found" {
			return err
		}
	}
	_, err = r.service.updateExecution(id, task.Execution.RequestID, "interrupted", "waiting_input", "", "Paused for user approval")
	r.changed(err)
	return err
}

func (r *Runner) Observe(id string) {
	r.mu.Lock()
	if r.ctx.Err() != nil || r.active[id] != nil {
		r.mu.Unlock()
		return
	}
	task, err := r.service.Get(id)
	if err != nil || task.Execution == nil || task.Execution.Spec.HostHome != r.home || !task.Execution.Unsettled() || task.Terminal() || task.ApprovalID != "" && !r.service.IsToolApproval(task.ApprovalID) {
		r.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(r.ctx)
	r.active[id] = cancel
	done := make(chan struct{})
	r.workers[id] = done
	r.mu.Unlock()
	go func() {
		defer func() {
			cancel()
			r.mu.Lock()
			delete(r.active, id)
			delete(r.workers, id)
			close(done)
			r.mu.Unlock()
		}()
		progress := func(phase string) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			_, err := r.service.updateExecution(id, task.Execution.RequestID, phase, "running", "", "")
			r.changed(err)
			return err
		}
		result, err := r.host.Run(ctx, task, progress)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			_, saveErr := r.service.updateExecution(id, task.Execution.RequestID, "unknown", "waiting_input", "", err.Error())
			if saveErr != nil && !errors.Is(saveErr, ErrConflict) {
				r.changed(saveErr)
			} else {
				r.changed(err)
			}
			return
		}
		_, err = r.service.updateExecution(id, task.Execution.RequestID, result.Phase, result.State, result.Summary, result.Error)
		r.changed(err)
	}()
}

func (r *Runner) StopTask(ctx context.Context, id string) (Task, error) {
	task, err := r.service.Get(id)
	if err != nil {
		return Task{}, err
	}
	if task.Execution == nil {
		return r.service.Transition(TransitionInput{TaskID: id, State: "cancelled", Actor: "user", Reason: "Cancelled in task inbox"})
	}
	if task.Execution.Spec.HostHome != r.home {
		return Task{}, fmt.Errorf("stop this task on its original Host")
	}
	if !task.Execution.Unsettled() {
		return r.service.Transition(TransitionInput{TaskID: id, State: "cancelled", Actor: "user", Reason: "Cancelled in task inbox"})
	}
	r.mu.Lock()
	if cancel := r.active[id]; cancel != nil {
		cancel()
	}
	done := r.workers[id]
	r.mu.Unlock()
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return Task{}, ctx.Err()
		}
	}
	if task.Execution.Unsettled() {
		if err := r.host.Stop(ctx, task.Execution.SessionID); err != nil {
			var remote *dsh.Error
			if !errors.As(err, &remote) || remote.Code != "session/not-found" {
				return Task{}, err
			}
		}
	}
	if _, err := r.service.updateExecution(id, task.Execution.RequestID, "stopped", "waiting_input", "", "Stopped by user"); err != nil {
		return Task{}, err
	}
	result, err := r.service.Transition(TransitionInput{TaskID: id, State: "cancelled", Actor: "user", Reason: "Cancelled in task inbox"})
	r.changed(err)
	return result, err
}

func (r *Runner) Schedule(in ScheduleInput, spec RunSpec) (Job, error) {
	if r.ctx.Err() != nil {
		return Job{}, r.ctx.Err()
	}
	if spec.HostHome != r.home {
		return Job{}, fmt.Errorf("schedule belongs to another Host")
	}
	j, err := r.service.ScheduleExecution(in, spec)
	r.changed(err)
	return j, err
}
