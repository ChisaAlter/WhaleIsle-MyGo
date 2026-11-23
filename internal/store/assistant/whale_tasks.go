package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/state"
)

type WhaleTaskSnapshot struct {
	Host                                string
	Loaded, Busy, CreateUnknown         bool
	Goal, Files, Criteria, BotID, Actor string
	Catalog                             pluginrpc.WhaleTasks
	Error                               *state.Fault
	Notice                              string
}
type WhaleTaskTicket struct {
	Host       string
	Generation uint64
	lease      state.Lease
}
type WhaleTaskPanel struct {
	mu    sync.Mutex
	state WhaleTaskSnapshot
	op    state.LeaseOwner
	opL   state.Lease
	pub   state.Published[WhaleTaskSnapshot]
}

// publishLocked deep-copies the catalog, which the task list mutates by
// reference through pluginrpc values.
func (s *WhaleTaskPanel) publishLocked() {
	v := s.state
	data, _ := json.Marshal(v.Catalog)
	v.Catalog = pluginrpc.WhaleTasks{}
	_ = json.Unmarshal(data, &v.Catalog)
	s.pub.Publish(v)
}

func (s *WhaleTaskPanel) Bind(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Host == host {
		return
	}
	s.opL = s.op.Renew(context.Background())
	s.state = WhaleTaskSnapshot{Host: host}
	s.publishLocked()
}
func (s *WhaleTaskPanel) Suspend() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.Busy {
		return
	}
	s.opL = s.op.Renew(context.Background())
	s.state.Busy = false
	s.state.Loaded = false
	s.publishLocked()
}
func (s *WhaleTaskPanel) Cancel(t WhaleTaskTicket) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.current(t) {
		return
	}
	s.opL = s.op.Renew(context.Background())
	s.state.Busy = false
	s.state.Loaded = false
	s.publishLocked()
}
func (s *WhaleTaskPanel) SetError(host string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == s.state.Host && err != nil {
		s.state.Error = state.FaultOf(err)
		s.publishLocked()
	}
}
func (s *WhaleTaskPanel) Snapshot() *state.Version[WhaleTaskSnapshot] {
	if v := s.pub.Load(); v != nil {
		return v
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pub.Load() == nil {
		s.pub.Publish(s.state)
	}
	return s.pub.Load()
}
func (s *WhaleTaskPanel) Edit(host, field, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Host != host || s.state.Busy {
		return
	}
	switch field {
	case "goal":
		s.state.Goal = value
	case "files":
		s.state.Files = value
	case "criteria":
		s.state.Criteria = value
	case "bot":
		s.state.BotID = value
	case "actor":
		s.state.Actor = value
	default:
		return
	}
	s.publishLocked()
}
func taskLines(value string) []string {
	var result []string
	for _, line := range strings.Split(value, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	return result
}
func (v WhaleTaskSnapshot) Draft() pluginrpc.WhaleTaskDraft {
	return pluginrpc.WhaleTaskDraft{Goal: strings.TrimSpace(v.Goal), Files: taskLines(v.Files), Criteria: taskLines(v.Criteria)}
}
func (s *WhaleTaskPanel) Begin(host, action string) (WhaleTaskTicket, WhaleTaskSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.state.Host || s.state.Busy || action == "start" && s.state.CreateUnknown {
		return WhaleTaskTicket{}, WhaleTaskSnapshot{}, false
	}
	s.opL = s.op.Renew(context.Background())
	s.state.Busy = true
	s.state.Error = nil
	s.state.Notice = ""
	if action == "start" {
		s.state.CreateUnknown = true
	}
	s.publishLocked()
	return WhaleTaskTicket{host, s.opL.Generation(), s.opL}, s.state, true
}
func (s *WhaleTaskPanel) Current(t WhaleTaskTicket) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current(t)
}
func (s *WhaleTaskPanel) current(t WhaleTaskTicket) bool {
	return s.state.Busy && t.Host == s.state.Host && t.lease == s.opL && s.opL.Valid()
}
func (s *WhaleTaskPanel) Finish(t WhaleTaskTicket, action string, catalog *pluginrpc.WhaleTasks, accepted bool, notice string, err error) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.current(t) {
		return false
	}
	s.state.Busy = false
	if catalog != nil {
		s.state.Catalog = *catalog
		s.state.Loaded = true
	}
	if action == "refresh" && err == nil {
		s.state.CreateUnknown = false
	}
	if accepted && action == "start" {
		s.state.Goal, s.state.Files, s.state.Criteria = "", "", ""
		s.state.CreateUnknown = false
	}
	if err != nil {
		s.state.Error = state.FaultOf(err)
	} else {
		s.state.Notice = notice
	}
	s.publishLocked()
	return true
}
