package assistant

import (
	"encoding/json"
	"strings"
	"sync"

	"whaleisle/internal/desktop/pluginrpc"
)

type WhaleTaskSnapshot struct {
	Host                                string
	Loaded, Busy, CreateUnknown         bool
	Goal, Files, Criteria, BotID, Actor string
	Catalog                             pluginrpc.WhaleTasks
	Error, Notice                       string
}
type WhaleTaskTicket struct {
	Host       string
	Generation uint64
}
type WhaleTaskPanel struct {
	mu         sync.Mutex
	state      WhaleTaskSnapshot
	generation uint64
}

func (s *WhaleTaskPanel) Bind(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Host == host {
		return
	}
	s.generation++
	s.state = WhaleTaskSnapshot{Host: host}
}
func (s *WhaleTaskPanel) Suspend() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.state.Busy {
		return
	}
	s.generation++
	s.state.Busy = false
	s.state.Loaded = false
}
func (s *WhaleTaskPanel) Cancel(t WhaleTaskTicket) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.current(t) {
		return
	}
	s.generation++
	s.state.Busy = false
	s.state.Loaded = false
}
func (s *WhaleTaskPanel) SetError(host string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == s.state.Host && err != nil {
		s.state.Error = err.Error()
	}
}
func (s *WhaleTaskPanel) Snapshot() WhaleTaskSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.state
	data, _ := json.Marshal(v.Catalog)
	v.Catalog = pluginrpc.WhaleTasks{}
	_ = json.Unmarshal(data, &v.Catalog)
	return v
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
	}
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
	s.generation++
	s.state.Busy = true
	s.state.Error = ""
	s.state.Notice = ""
	if action == "start" {
		s.state.CreateUnknown = true
	}
	return WhaleTaskTicket{host, s.generation}, s.state, true
}
func (s *WhaleTaskPanel) Current(t WhaleTaskTicket) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current(t)
}
func (s *WhaleTaskPanel) current(t WhaleTaskTicket) bool {
	return s.state.Busy && t.Host == s.state.Host && t.Generation == s.generation
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
		s.state.Error = err.Error()
	} else {
		s.state.Notice = notice
	}
	return true
}
