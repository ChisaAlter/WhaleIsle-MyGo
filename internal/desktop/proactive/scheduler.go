package proactive

import (
	"crypto/rand"
	"fmt"
	"reflect"
	"time"
)

func (s *Service) Schedule(in ScheduleInput) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[in.TaskID]
	if !ok {
		return Job{}, ErrNotFound
	}
	if t.Terminal() {
		return Job{}, fmt.Errorf("%w: cannot schedule a terminal task", ErrConflict)
	}
	at, err := time.Parse(time.RFC3339Nano, in.At)
	if err != nil {
		return Job{}, fmt.Errorf("schedule at must be an ISO date with a time zone: %w", err)
	}
	if in.IntervalMS != nil && (*in.IntervalMS <= 0 || *in.IntervalMS > int64((1<<63-1)/time.Millisecond)) {
		return Job{}, fmt.Errorf("intervalMs is outside the positive duration range")
	}
	if in.ID == "" {
		in.ID = rand.Text()
	}
	j := Job{ID: in.ID, TaskID: in.TaskID, At: timestamp(at), IntervalMS: in.IntervalMS, Metadata: in.Metadata, Status: "scheduled"}
	if prior, exists := s.jobs[j.ID]; exists {
		if reflect.DeepEqual(prior, j) {
			return clone(prior), nil
		}
		return Job{}, fmt.Errorf("%w: schedule identity already exists", ErrConflict)
	}
	if err := s.commit(Event{Type: "schedule.created", Job: &j}); err != nil {
		return Job{}, err
	}
	return clone(j), nil
}

func (s *Service) CancelSchedule(id string) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if j.Status != "scheduled" {
		return clone(j), nil
	}
	if err := s.commit(Event{Type: "schedule.cancelled", JobID: id}); err != nil {
		return Job{}, err
	}
	return clone(s.jobs[id]), nil
}

// Poll is the explicit consume operation from the desktop proactive protocol.
// Merely listing jobs never fires them. The caller owns execution of the returned
// due occurrences; repeating jobs advance from their previous due date.
func (s *Service) Poll(at time.Time) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = s.now()
	}
	due := []Job{}
	var events []Event
	for _, id := range s.jobOrder {
		j := s.jobs[id]
		if j.Status != "scheduled" {
			continue
		}
		// Executable schedules are consumed atomically with their child task by
		// the native runner; a Host poll cannot steal these occurrences.
		if j.Execution != nil {
			continue
		}
		t, exists := s.tasks[j.TaskID]
		if !exists || t.Terminal() {
			events = append(events, Event{Type: "schedule.cancelled", JobID: j.ID})
			continue
		}
		when, err := time.Parse(time.RFC3339Nano, j.At)
		if err != nil {
			return nil, fmt.Errorf("stored schedule %s: %w", j.ID, err)
		}
		if when.After(at) {
			continue
		}
		// An approval is never consumed by a timer or used as permission to run.
		if t.ApprovalID != "" || t.State == "waiting_approval" {
			continue
		}
		due = append(due, clone(j))
		e := Event{Type: "schedule.fired", JobID: j.ID}
		if j.IntervalMS != nil {
			e.Type = "schedule.rescheduled"
			e.DueAt = timestamp(when.Add(time.Duration(*j.IntervalMS) * time.Millisecond))
		}
		events = append(events, e)
	}
	if err := s.commit(events...); err != nil {
		return nil, err
	}
	return due, nil
}
