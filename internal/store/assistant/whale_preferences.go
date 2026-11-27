package assistant

import (
	"context"
	"fmt"
	"sync"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/state"
	"whaleisle/internal/store/storelocale"
)

type WhalePreferencesSnapshot struct {
	Host         string
	Loaded, Busy bool
	Server       pluginrpc.WhalePreferences
	Route        pluginrpc.WhaleModelRoute
	IMDefault    bool
	Catalog      api.ModelCatalog
	ModelsError  *state.Fault
	Error        *state.Fault
	Notice       string
}
type WhalePreferencesEditor struct {
	mu    sync.Mutex
	state WhalePreferencesSnapshot
	op    state.LeaseOwner
	opL   state.Lease
	pub   state.Published[WhalePreferencesSnapshot]
}
type WhalePreferencesTicket struct {
	Host       string
	Generation uint64
	lease      state.Lease
}
type WhalePreferencesCommand struct {
	Action, Skill string
	Enabled       bool
	Route         pluginrpc.WhaleModelRoute
	IMDefault     bool
}

func (s *WhalePreferencesEditor) publishLocked() {
	v := s.state
	v.Server.Skills = append([]pluginrpc.WhaleSkill(nil), v.Server.Skills...)
	s.pub.Publish(v)
}

func (s *WhalePreferencesEditor) Bind(host string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Host == host {
		return
	}
	s.opL = s.op.Renew(context.Background())
	s.state = WhalePreferencesSnapshot{Host: host}
	s.publishLocked()
}
func (s *WhalePreferencesEditor) Suspend() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opL = s.op.Renew(context.Background())
	if s.state.Busy {
		s.state.Loaded = false
		s.state.Notice = storelocale.T(storelocale.KeyStoreAssistantInterruptedRefresh)
	}
	s.state.Busy = false
	s.publishLocked()
}
func (s *WhalePreferencesEditor) Snapshot() *state.Version[WhalePreferencesSnapshot] {
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
func (s *WhalePreferencesEditor) Edit(host string, route pluginrpc.WhaleModelRoute, im bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host != s.state.Host || s.state.Busy || !s.state.Loaded {
		return
	}
	s.state.Route = route
	s.state.IMDefault = im
	s.state.Notice = ""
	s.publishLocked()
}
func (s *WhalePreferencesEditor) Begin(host string, write bool) (WhalePreferencesTicket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if host == "" || host != s.state.Host || s.state.Busy || (write && !s.state.Loaded) {
		return WhalePreferencesTicket{}, false
	}
	s.opL = s.op.Renew(context.Background())
	s.state.Busy = true
	s.state.Error = nil
	s.state.Notice = ""
	s.publishLocked()
	return WhalePreferencesTicket{host, s.opL.Generation(), s.opL}, true
}
func (s *WhalePreferencesEditor) Current(t WhalePreferencesTicket) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current(t)
}
func (s *WhalePreferencesEditor) current(t WhalePreferencesTicket) bool {
	return t.Host == s.state.Host && t.lease == s.opL && s.opL.Valid() && s.state.Busy
}
func (s *WhalePreferencesEditor) Finish(t WhalePreferencesTicket, v pluginrpc.WhalePreferences, cat *api.ModelCatalog, modelErr *state.Fault, cmd *WhalePreferencesCommand, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.current(t) {
		return
	}
	s.state.Busy = false
	if err != nil {
		s.state.Error = state.FaultOf(err)
		if cmd != nil && s.state.Loaded {
			// Keep the loaded page interactive after a rejected write and roll back
			// the unpersisted draft control to the last confirmed server value.
			switch cmd.Action {
			case "model":
				s.state.Route = s.state.Server.Route()
			case "im":
				s.state.IMDefault = s.state.Server.IMDefault
			}
			s.state.Notice = ""
		} else {
			s.state.Loaded = false
		}
		s.publishLocked()
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
	s.state.Notice = storelocale.T(storelocale.KeyStoreAssistantWhaleSettingsLoaded)
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
			s.state.Error = state.FaultText(fmt.Sprintf(storelocale.T(storelocale.KeyStoreAssistantReadbackDiffers), cmd.Action))
			s.publishLocked()
			return
		}
		s.state.Notice = storelocale.T(storelocale.KeyStoreAssistantSavedAndReadBack)
	}
	s.publishLocked()
}
