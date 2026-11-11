package proactive

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Service is the sole journal writer in a desktop profile. The application's
// existing profile lock owns interprocess exclusion; mu serializes UI and HTTP.
type Service struct {
	mu                                 sync.Mutex
	path                               string
	now                                func() time.Time
	events                             []json.RawMessage
	disk                               []byte
	diskExists                         bool
	tasks                              map[string]Task
	approvals                          map[string]Approval
	jobs                               map[string]Job
	taskOrder, approvalOrder, jobOrder []string
	transitionKeys                     map[string]TransitionInput
	createKeys                         map[string]string
	sources                            map[string]Source
	sourceOrder                        []string
	notices                            map[string]Notice
	noticeOrder                        []string
	tools                              map[string]ToolOperation
}

func Open(userData string) (*Service, error) {
	if userData == "" {
		return nil, fmt.Errorf("proactive user data directory is required")
	}
	s := &Service{path: filepath.Join(userData, "assistant", "proactive-events.ndjson"), now: time.Now,
		tasks: map[string]Task{}, approvals: map[string]Approval{}, jobs: map[string]Job{},
		transitionKeys: map[string]TransitionInput{}, createKeys: map[string]string{}, sources: map[string]Source{}, notices: map[string]Notice{}, tools: map[string]ToolOperation{}}
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s.diskExists = true
	r := bufio.NewReader(f)
	line := 0
	for {
		raw, readErr := r.ReadBytes('\n')
		s.disk = append(s.disk, raw...)
		line++
		if len(bytes.TrimSpace(raw)) > 0 {
			var event Event
			if err := json.Unmarshal(raw, &event); err != nil {
				return nil, fmt.Errorf("proactive journal line %d: %w; original file preserved", line, err)
			}
			if err := validateStoredEvent(event); err != nil {
				return nil, fmt.Errorf("proactive journal line %d: %w; original file preserved", line, err)
			}
			s.events = append(s.events, bytes.Clone(bytes.TrimSpace(raw)))
			s.apply(event)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	return s, nil
}

// commit writes the entire transaction before publishing any in-memory state.
// A same-directory replacement prevents a crash between approval/task/schedule
// events from exposing half a decision. Existing unknown event fields survive.
func (s *Service) commit(events ...Event) error {
	if len(events) == 0 {
		return nil
	}
	encoded := make([]json.RawMessage, len(events))
	var next bytes.Buffer
	for _, raw := range s.events {
		next.Write(raw)
		next.WriteByte('\n')
	}
	for i := range events {
		events[i].EventID = rand.Text()
		events[i].At = timestamp(s.now())
		events[i].Sequence = uint64(len(s.events) + i + 1)
		raw, err := json.Marshal(events[i])
		if err != nil {
			return err
		}
		encoded[i] = raw
		next.Write(raw)
		next.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".proactive-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	write := func() error {
		if _, err := f.Write(next.Bytes()); err != nil {
			return err
		}
		return f.Sync()
	}
	err = write()
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	current, err := os.ReadFile(s.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if s.diskExists != (err == nil) || !bytes.Equal(s.disk, current) {
		return fmt.Errorf("%w: task journal changed on disk; reopen it before writing", ErrConflict)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	s.disk, s.diskExists = next.Bytes(), true
	for i, event := range events {
		s.events = append(s.events, encoded[i])
		s.apply(event)
	}
	return nil
}

func validateStoredEvent(e Event) error {
	switch e.Type {
	case "delivery.updated":
		if e.ApprovalID == "" || e.Delivery == nil {
			return fmt.Errorf("invalid persisted delivery")
		}
	case "tool.operation":
		if e.Tool == nil || e.Tool.Key == "" || e.Tool.Call.CallID == "" {
			return fmt.Errorf("invalid persisted tool operation")
		}
	case "source.configured", "source.checked":
		if e.Source == nil || e.Source.ID == "" {
			return fmt.Errorf("invalid persisted source")
		}
		if _, err := NormalizeRepository(e.Source.Repository); err != nil {
			return err
		}
	case "task.source":
		if e.TaskID == "" || e.SourceItem == nil {
			return fmt.Errorf("invalid persisted source item")
		}
	case "notice.created", "notice.updated":
		if e.Notice == nil || e.Notice.ID == "" {
			return fmt.Errorf("invalid persisted notice")
		}
	case "":
		return fmt.Errorf("event type is required")
	case "task.created":
		if e.Task == nil || e.Task.ID == "" || !validState(e.Task.State) || e.Task.Autonomy < 1 || e.Task.Autonomy > 5 {
			return fmt.Errorf("invalid persisted task")
		}
		if e.Task.Execution != nil {
			return validateExecution(*e.Task.Execution)
		}
	case "task.execution":
		if e.TaskID == "" || e.Execution == nil {
			return fmt.Errorf("invalid persisted task execution")
		}
		return validateExecution(*e.Execution)
	case "task.transitioned":
		if e.TaskID == "" || !validState(e.State) {
			return fmt.Errorf("invalid persisted transition")
		}
	case "approval.requested":
		if e.Approval == nil || e.Approval.ID == "" || e.Approval.TaskID == "" {
			return fmt.Errorf("invalid persisted approval")
		}
	case "schedule.created":
		if e.Job == nil || e.Job.ID == "" || e.Job.TaskID == "" {
			return fmt.Errorf("invalid persisted schedule")
		}
		if _, err := time.Parse(time.RFC3339Nano, e.Job.At); err != nil {
			return fmt.Errorf("invalid schedule date: %w", err)
		}
		if e.Job.IntervalMS != nil && (*e.Job.IntervalMS <= 0 || *e.Job.IntervalMS > int64((1<<63-1)/time.Millisecond)) {
			return fmt.Errorf("invalid persisted repeat interval")
		}
		if e.Job.Execution != nil {
			return validateRunSpec(*e.Job.Execution)
		}
	}
	return nil
}

func (s *Service) apply(e Event) {
	switch e.Type {
	case "delivery.updated":
		if a, ok := s.approvals[e.ApprovalID]; ok {
			a.Delivery = clone(e.Delivery)
			a.Effect, a.EffectError = e.Delivery.Status, e.Delivery.Error
			s.approvals[a.ID] = a
		}
	case "tool.operation":
		op := clone(*e.Tool)
		s.tools[op.Key] = op
		if a, ok := s.approvals[op.ApprovalID]; ok {
			a.Effect, a.EffectError = op.Status, op.Error
			s.approvals[a.ID] = a
		}
	case "source.configured", "source.checked":
		if _, ok := s.sources[e.Source.ID]; !ok {
			s.sourceOrder = append(s.sourceOrder, e.Source.ID)
		}
		s.sources[e.Source.ID] = *e.Source
	case "notice.created", "notice.updated":
		if _, ok := s.notices[e.Notice.ID]; !ok {
			s.noticeOrder = append(s.noticeOrder, e.Notice.ID)
		}
		s.notices[e.Notice.ID] = *e.Notice
	case "task.source":
		if t, ok := s.tasks[e.TaskID]; ok {
			t.SourceItem = clone(e.SourceItem)
			if t.Execution == nil && t.State == "discovered" {
				t.Title = e.SourceItem.Title
				t.Description = e.SourceItem.Description
				t.Labels = clone(e.SourceItem.Labels)
				t.SourceURL = e.SourceItem.URL
			}
			s.tasks[t.ID] = t
		}
	case "task.execution":
		if e.Execution == nil {
			return
		}
		if t, exists := s.tasks[e.TaskID]; exists {
			t.Execution = clone(e.Execution)
			t.UpdatedAt = e.At
			s.tasks[t.ID] = t
		}
	case "task.created":
		if e.Task == nil {
			return
		}
		t := clone(*e.Task)
		if _, exists := s.tasks[t.ID]; !exists {
			s.taskOrder = append(s.taskOrder, t.ID)
		}
		s.tasks[t.ID] = t
		if t.IdempotencyKey != "" {
			s.createKeys[t.IdempotencyKey] = t.ID
		}
	case "task.transitioned":
		t, ok := s.tasks[e.TaskID]
		if !ok {
			return
		}
		t.State, t.UpdatedAt, t.LastReason, t.LastEvidence = e.State, e.At, e.Reason, clone(e.Evidence)
		s.tasks[t.ID] = t
		if e.IdempotencyKey != "" {
			s.transitionKeys[e.TaskID+"\x00"+e.IdempotencyKey] = TransitionInput{TaskID: e.TaskID, State: e.State, Reason: e.Reason}
		}
	case "approval.requested":
		if e.Approval == nil {
			return
		}
		a := clone(*e.Approval)
		if _, exists := s.approvals[a.ID]; !exists {
			s.approvalOrder = append(s.approvalOrder, a.ID)
		}
		s.approvals[a.ID] = a
		if t, ok := s.tasks[a.TaskID]; ok {
			t.ApprovalID = a.ID
			s.tasks[t.ID] = t
		}
	case "approval.decided":
		a, ok := s.approvals[e.ApprovalID]
		if !ok {
			return
		}
		a.Decision, a.Note = e.Decision, e.Note
		s.approvals[a.ID] = a
	case "outbox.updated":
		a, ok := s.approvals[e.ApprovalID]
		if !ok {
			return
		}
		a.Status, a.DecidedAt, a.DecidedBy = e.Status, e.DecidedAt, e.DecidedBy
		s.approvals[a.ID] = a
		if t, ok := s.tasks[a.TaskID]; ok && t.ApprovalID == a.ID && a.Status != "pending" {
			t.ApprovalID = ""
			s.tasks[t.ID] = t
		}
	case "schedule.created":
		if e.Job == nil {
			return
		}
		j := clone(*e.Job)
		if _, exists := s.jobs[j.ID]; !exists {
			s.jobOrder = append(s.jobOrder, j.ID)
		}
		s.jobs[j.ID] = j
	case "schedule.cancelled", "schedule.rescheduled", "schedule.fired":
		j, ok := s.jobs[e.JobID]
		if !ok {
			return
		}
		switch e.Type {
		case "schedule.cancelled":
			j.Status = "cancelled"
		case "schedule.fired":
			j.Status = "fired"
		case "schedule.rescheduled":
			j.At = e.DueAt
		}
		s.jobs[j.ID] = j
	}
}

func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := Snapshot{Tasks: []Task{}, Outbox: []Approval{}, Schedules: []Job{}}
	for _, id := range s.taskOrder {
		out.Tasks = append(out.Tasks, s.tasks[id])
	}
	for _, id := range s.approvalOrder {
		out.Outbox = append(out.Outbox, s.approvals[id])
	}
	for _, id := range s.jobOrder {
		out.Schedules = append(out.Schedules, s.jobs[id])
	}
	for _, id := range s.sourceOrder {
		out.Sources = append(out.Sources, s.sources[id])
	}
	for _, id := range s.noticeOrder {
		out.Notices = append(out.Notices, s.notices[id])
	}
	return clone(out)
}

func (s *Service) Get(id string) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return clone(t), nil
}
