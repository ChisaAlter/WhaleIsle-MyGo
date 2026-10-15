package session

import (
	"context"
	"fmt"
	"strings"
	"whaleisle/internal/dsh/api"
)

// SessionRenameView owns the editor across paints and asynchronous acknowledgements.
type SessionRenameView struct {
	SessionID, Draft, Error string
	Busy                    bool
}

func (s *Store) SessionRename() SessionRenameView { s.mu.Lock(); defer s.mu.Unlock(); return s.rename }
func (s *Store) invalidateRenameLocked()          { s.renameEpoch++; s.rename = SessionRenameView{} }
func (s *Store) BeginSessionRename(id, title string) {
	s.mu.Lock()
	if !s.rename.Busy && id != "" {
		s.renameEpoch++
		s.rename = SessionRenameView{SessionID: id, Draft: title}
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) EditSessionRename(title string) {
	s.mu.Lock()
	if !s.rename.Busy {
		s.rename.Draft = title
		s.rename.Error = ""
	}
	s.mu.Unlock()
}
func (s *Store) CancelSessionRename() {
	s.mu.Lock()
	if !s.rename.Busy {
		s.invalidateRenameLocked()
	}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) SubmitSessionRename(id, title string) {
	s.RenameSession(context.Background(), id, title)
}

// RenameSession closes only the submitted editor after a matching Host acknowledgement.
func (s *Store) RenameSession(ctx context.Context, sessionID, title string) {
	title = strings.TrimSpace(title)
	s.mu.Lock()
	if sessionID == "" || title == "" || s.rename.Busy {
		s.mu.Unlock()
		return
	}
	client, epoch, hostEpoch := s.client, s.renameEpoch, s.eventsGeneration
	editing := s.rename.SessionID == sessionID
	if client == nil {
		if editing {
			s.rename.Error = "Host is not connected"
		}
		s.mu.Unlock()
		s.fire()
		return
	}
	if editing {
		s.rename.Busy = true
		s.rename.Error = ""
	}
	s.mu.Unlock()
	s.fire()
	go func() {
		var out api.SessionRenameValue
		err := client.Call(ctx, "session/rename", map[string]any{"request": api.SessionRenameRequest{SessionId: sessionID, Title: title}}, &out)
		if err == nil && strings.TrimSpace(out.Title) == "" {
			err = fmt.Errorf("rename: missing acknowledged title")
		}
		s.mu.Lock()
		if s.client != client || s.renameEpoch != epoch {
			s.mu.Unlock()
			return
		}
		if editing {
			s.rename.Busy = false
		}
		if err != nil {
			s.lastAction = err.Error()
			if editing {
				s.rename.Error = err.Error()
			}
			s.mu.Unlock()
			s.fire()
			return
		}
		for i := range s.items {
			if s.items[i].SessionID == sessionID {
				s.items[i].Title = out.Title
				break
			}
		}
		if editing {
			s.invalidateRenameLocked()
		}
		s.lastAction = "session renamed"
		s.mu.Unlock()
		s.fire()
		// Presentation belongs to the same Host as the acknowledged rename.
		// UI notification can synchronously replace the Host before this hop.
		s.mu.Lock()
		currentHost := s.client == client && s.eventsGeneration == hostEpoch
		s.mu.Unlock()
		if !currentHost {
			return
		}
		req := api.SessionPresentationRequest{SessionId: sessionID, Presentation: api.SessionPresentationRequestPresentation{Owner: "whaleisle", Title: out.Title}}
		var presentation api.SessionPresentationValue
		// Non-fatal follow-up; never let a late result mutate the current editor.
		_ = client.Call(ctx, "session/setPresentation", map[string]any{"request": req}, &presentation)
	}()
}
