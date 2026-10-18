package assistant

import (
	"fmt"
	"sync"
	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh/api"
)

type WhalePreferencesSnapshot struct {
	Host                       string
	Loaded, Busy               bool
	Server                     pluginrpc.WhalePreferences
	Route                      pluginrpc.WhaleModelRoute
	IMDefault                  bool
	Catalog                    api.ModelCatalog
	ModelsError, Error, Notice string
}
type WhalePreferencesEditor struct {
	mu         sync.Mutex
	state      WhalePreferencesSnapshot
	generation uint64
}
type WhalePreferencesTicket struct {
	Host       string
	Generation uint64
}
type WhalePreferencesCommand struct {
	Action, Skill string
	Enabled       bool
	Route         pluginrpc.WhaleModelRoute
	IMDefault     bool
}

func (s *WhalePreferencesEditor) Bind(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Host == host {
		return
	}
	s.generation++
	s.state = WhalePreferencesSnapshot{Host: host}
}
func (s *WhalePreferencesEditor) Suspend() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.generation++
	if s.state.Busy {
		s.state.Loaded = false
		s.state.Notice = "操作中断，请刷新 / Interrupted; refresh"
	}
	s.state.Busy = false
}
func (s *WhalePreferencesEditor) Snapshot() WhalePreferencesSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.state
	v.Server.Skills = append([]pluginrpc.WhaleSkill(nil), v.Server.Skills...)
	return v
}
func (s *WhalePreferencesEditor) Edit(host string, route pluginrpc.WhaleModelRoute, im bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host != s.state.Host || s.state.Busy || !s.state.Loaded {
		return
	}
	s.state.Route = route
	s.state.IMDefault = im
	s.state.Notice = ""
}
func (s *WhalePreferencesEditor) Begin(host string, write bool) (WhalePreferencesTicket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.state.Host || s.state.Busy || (write && !s.state.Loaded) {
		return WhalePreferencesTicket{}, false
	}
	s.generation++
	s.state.Busy = true
	s.state.Error = ""
	s.state.Notice = ""
	return WhalePreferencesTicket{host, s.generation}, true
}
func (s *WhalePreferencesEditor) Current(t WhalePreferencesTicket) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current(t)
}
func (s *WhalePreferencesEditor) current(t WhalePreferencesTicket) bool {
	return t.Host == s.state.Host && t.Generation == s.generation && s.state.Busy
}
func (s *WhalePreferencesEditor) Finish(t WhalePreferencesTicket, v pluginrpc.WhalePreferences, cat *api.ModelCatalog, modelErr string, cmd *WhalePreferencesCommand, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.current(t) {
		return
	}
	s.state.Busy = false
	if err != nil {
		s.state.Error = err.Error()
		s.state.Loaded = false
		if cmd != nil {
			s.state.Notice = "保存结果未核实，请刷新后核对 / Save not verified; refresh to compare"
		}
		return
	}
	old := s.state.Server
	reset := !s.state.Loaded || old.HomeDir != v.HomeDir || old.SessionID != v.SessionID
	// A refresh after an uncertain write keeps the user's selection for comparison.
	if old.HomeDir != "" && old.HomeDir == v.HomeDir && old.SessionID == v.SessionID {
		reset = false
	}
	if reset || s.state.Route == old.Route() {
		s.state.Route = v.Route()
	}
	if reset || s.state.IMDefault == old.IMDefault {
		s.state.IMDefault = v.IMDefault
	}
	s.state.Server = v
	s.state.Loaded = true
	if cat != nil {
		s.state.Catalog = *cat
		s.state.ModelsError = modelErr
	}
	s.state.Notice = "已读取 Whale 设置 / Whale settings loaded"
	if cmd != nil {
		matches := true
		switch cmd.Action {
		case "model":
			matches = v.Route() == cmd.Route
			if matches {
				s.state.Route = v.Route()
			}
		case "im":
			matches = v.IMDefault == cmd.IMDefault
			if matches {
				s.state.IMDefault = v.IMDefault
			}
		case "skill":
			matches = false
			for _, skill := range v.Skills {
				if skill.Name == cmd.Skill {
					matches = skill.Enabled == cmd.Enabled
					break
				}
			}
		}
		if !matches {
			s.state.Error = fmt.Sprintf("Host 已返回，但 %s 读回与选择不一致，请核对 / Read-back differs", cmd.Action)
			return
		}
		s.state.Notice = "已保存并读回 Whale 设置 / Saved and read back"
	}
}
