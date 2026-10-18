package hostterminal

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
)

// RenameView is an identity-owned editor; a failed or uncertain RPC leaves the
// original draft intact for inspection and an explicit retry.
type RenameView struct {
	Editing, Saving bool
	Draft, Error    string
}
type renameState struct {
	view     RenameView
	revision uint64
}

func (s *Store) renameEntryLocked(owner uint64, session, id string) *entry {
	if s.epoch != owner || s.remote == nil || s.visible != session {
		return nil
	}
	st := s.sessions[session]
	e := s.entries[terminalKey{session, id}]
	if st == nil || st.active != id || e == nil || e.closing || e.phase == "closed" {
		return nil
	}
	return e
}
func (s *Store) RenameSnapshot(owner uint64, session, id string) RenameView {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.epoch != owner {
		return RenameView{}
	}
	if e := s.entries[terminalKey{session, id}]; e != nil {
		return e.rename.view
	}
	return RenameView{}
}
func (s *Store) BeginRename(owner uint64, session, id string) {
	s.mu.Lock()
	e := s.renameEntryLocked(owner, session, id)
	if e == nil || e.rename.view.Editing {
		s.mu.Unlock()
		return
	}
	e.rename.view = RenameView{Editing: true, Draft: e.info.Title}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) EditRename(owner uint64, session, id, draft string) {
	s.mu.Lock()
	e := s.renameEntryLocked(owner, session, id)
	if e == nil || !e.rename.view.Editing || e.rename.view.Saving {
		s.mu.Unlock()
		return
	}
	e.rename.view.Draft = draft
	e.rename.view.Error = ""
	s.mu.Unlock()
	s.fire()
}
func (s *Store) CancelRename(owner uint64, session, id string) {
	s.mu.Lock()
	e := s.renameEntryLocked(owner, session, id)
	if e == nil || e.rename.view.Saving {
		s.mu.Unlock()
		return
	}
	e.rename.revision++
	e.rename.view = RenameView{}
	s.mu.Unlock()
	s.fire()
}
func (s *Store) cancelRenameLocked(e *entry) {
	e.rename.revision++
	if e.rename.view.Saving {
		e.rename.view.Saving = false
		e.rename.view.Error = "Terminal close started; the name draft is preserved."
	}
}
func (s *Store) SaveRename(owner uint64, session, id string) {
	s.mu.Lock()
	e := s.renameEntryLocked(owner, session, id)
	if e == nil || !e.rename.view.Editing || e.rename.view.Saving {
		s.mu.Unlock()
		return
	}
	title := strings.TrimFunc(e.rename.view.Draft, func(r rune) bool { return r == '\ufeff' || unicode.IsSpace(r) })
	if !s.ready {
		e.rename.view.Error = "Host disconnected; the name draft is preserved."
		s.mu.Unlock()
		s.fire()
		return
	}
	if title == "" || len(utf16.Encode([]rune(title))) > 120 {
		e.rename.view.Error = "Terminal title must contain 1–120 UTF-16 characters."
		s.mu.Unlock()
		s.fire()
		return
	}
	if title == e.info.Title {
		e.rename.view = RenameView{}
		s.mu.Unlock()
		s.fire()
		return
	}
	e.rename.revision++
	revision := e.rename.revision
	e.rename.view.Saving = true
	e.rename.view.Error = ""
	remote, ctx, epoch := s.remote, s.ctx, s.epoch
	s.mu.Unlock()
	s.fire()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		err := remote.Call(ctx, "terminal/rename", map[string]any{"agentId": session, "id": id, "title": title}, nil)
		s.mu.Lock()
		if s.epoch != epoch || s.entries[e.key] != e || e.rename.revision != revision {
			s.mu.Unlock()
			return
		}
		e.rename.view.Saving = false
		if err != nil {
			e.rename.view.Error = "Rename was not confirmed; the name draft is preserved. " + err.Error()
		} else {
			e.info.Title = title
			e.rename.view = RenameView{}
			if st := s.sessions[session]; st != nil {
				for i := range st.items {
					if st.items[i].ID == id {
						st.items[i].Title = title
					}
				}
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}
