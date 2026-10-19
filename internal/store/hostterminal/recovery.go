package hostterminal

import (
	"context"
	"errors"
	"regexp"
	"time"
)

// RecoveryTarget identifies an existing Host process. It contains no transport,
// input capability, terminal output, command or local shell launch request.
type RecoveryTarget struct {
	Authority  string `json:"authority"`
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
	Title      string `json:"title,omitempty"`
}

var recoveryTerminalID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func (s *Store) RecoverySnapshot(authority, session string) *RecoveryTarget {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.sessions[session]
	if authority == "" || st == nil || st.active == "" {
		return nil
	}
	e := s.entries[terminalKey{session, st.active}]
	if e == nil || (e.phase == "closed" && !e.desired) {
		return nil
	}
	return &RecoveryTarget{Authority: authority, SessionID: session, TerminalID: e.key.id, Title: e.info.Title}
}

// AdoptRecovery is called for a displayed occurrence after its trusted Host is
// bound. Discovery must find the saved identity before retain/attach; recovery
// never calls terminal/create and never substitutes a missing process.
// A failed lookup stays visible for explicit Refresh/Connect, without retries.
func (s *Store) AdoptRecovery(owner uint64, authority, session string, saved *RecoveryTarget) bool {
	if saved == nil || authority == "" || saved.Authority != authority || saved.SessionID != session || !recoveryTerminalID.MatchString(saved.TerminalID) {
		return false
	}
	s.mu.Lock()
	if s.epoch != owner || s.remote == nil || !s.ready || s.visible != session || s.ctx == nil || s.ctx.Err() != nil {
		s.mu.Unlock()
		return false
	}
	st := s.sessionLocked(session)
	if st.recoveredID == saved.TerminalID || st.active != "" || st.busy {
		s.mu.Unlock()
		return false
	}
	id := saved.TerminalID
	st.recoveredID, st.active, st.busy, st.error = id, id, true, ""
	st.operation++
	op, epoch, remote, parent := st.operation, s.epoch, s.remote, s.ctx
	e := s.entryLocked(terminalKey{session, id})
	e.info.Title, e.phase = saved.Title, "loading"
	st.items = append(st.items, e.info)
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(parent, 15*time.Second)
		defer cancel()
		var environment Environment
		err := remote.Call(ctx, "terminal/environment", map[string]any{"agentId": session}, &environment)
		if err == nil {
			err = validEnvironment(environment)
		}
		var items []Info
		if err == nil {
			err = remote.Call(ctx, "terminal/list", map[string]any{"sessionId": session}, &items)
		}
		var found *Info
		for i := range items {
			if items[i].ID == id {
				found = &items[i]
				break
			}
		}
		if err == nil && found == nil {
			err = errors.New("Saved Host terminal is no longer available; no replacement was created")
		}
		s.mu.Lock()
		if s.epoch != epoch || st.operation != op || s.entries[e.key] != e || parent.Err() != nil {
			s.mu.Unlock()
			return
		}
		st.busy = false
		if err != nil {
			e.phase, e.error = "failed", err.Error()
		} else {
			st.environment, st.items = environment, items
			e.info, e.desired, e.error = *found, true, ""
			s.startHoldLocked(e)
		}
		s.mu.Unlock()
		s.fire()
	}()
	return true
}
