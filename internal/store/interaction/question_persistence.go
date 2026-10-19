package interaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
)

type savedQuestionAnswer struct {
	ID       string   `json:"id"`
	Selected []string `json:"selected,omitempty"`
	Custom   string   `json:"custom,omitempty"`
	Skipped  bool     `json:"skipped,omitempty"`
}

type savedQuestionDraft struct {
	SessionID   string                `json:"sessionId"`
	CallID      string                `json:"callId"`
	Fingerprint string                `json:"fingerprint"`
	Index       int                   `json:"index"`
	Answers     []savedQuestionAnswer `json:"answers"`
	Wait        string                `json:"wait,omitempty"`
}

type questionDraftEnvelope struct {
	Version int                           `json:"version"`
	Owner   string                        `json:"owner"`
	Drafts  map[string]savedQuestionDraft `json:"drafts"`
}

type questionDraftStorage struct {
	owner    string
	restored bool
	deferred map[string]savedQuestionDraft
	save     func(json.RawMessage) error
	last     json.RawMessage
	err      string
	epoch    uint64
}

// Question draft writes have a separate lock so no filesystem callback runs
// under Store.mu, and older writes cannot finish after newer ones.
type questionDraftWriter struct{ mu sync.Mutex }

func questionDraftOwner(owner string) string {
	if strings.TrimSpace(owner) == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(owner))
	return hex.EncodeToString(sum[:])
}

func questionDraftFingerprint(req userQuestionsRequest) string {
	raw, _ := json.Marshal(req.Questions)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func snapshotQuestionDraft(sessionID, callID string, draft *draftSession) savedQuestionDraft {
	out := savedQuestionDraft{SessionID: sessionID, CallID: callID, Fingerprint: questionDraftFingerprint(draft.req), Index: draft.index}
	for _, q := range draft.req.Questions {
		answer := draft.answers[q.ID]
		if answer != nil {
			out.Answers = append(out.Answers, savedQuestionAnswer{ID: q.ID, Selected: append([]string(nil), answer.selected...), Custom: answer.custom, Skipped: answer.skipped})
		}
	}
	if draft.wait.held {
		out.Wait = "waiting"
	} else if draft.wait.editing {
		out.Wait = "editing"
	}
	return out
}

func (s *Store) retainQuestionDraftsLocked() {
	if s.questionStorage.owner == "" {
		return
	}
	if s.questionStorage.deferred == nil {
		s.questionStorage.deferred = make(map[string]savedQuestionDraft)
	}
	for _, p := range s.pending {
		if draft := s.questionDrafts[p.EventID]; p.Event == EventUserQuestions && p.CallID != "" && draft != nil && planReviewOf(draft.req.Questions) == nil {
			s.questionStorage.deferred[questionCardID(p.AgentID, p.CallID)] = snapshotQuestionDraft(p.AgentID, p.CallID, draft)
		}
	}
}

func (s *Store) exportQuestionDraftsLocked() json.RawMessage {
	if s.questionStorage.owner == "" {
		return nil
	}
	envelope := questionDraftEnvelope{Version: 1, Owner: s.questionStorage.owner, Drafts: make(map[string]savedQuestionDraft)}
	for key, draft := range s.questionStorage.deferred {
		envelope.Drafts[key] = draft
	}
	for _, p := range s.pending {
		if draft := s.questionDrafts[p.EventID]; p.Event == EventUserQuestions && p.CallID != "" && draft != nil && planReviewOf(draft.req.Questions) == nil {
			envelope.Drafts[questionCardID(p.AgentID, p.CallID)] = snapshotQuestionDraft(p.AgentID, p.CallID, draft)
		}
	}
	raw, _ := json.Marshal(envelope)
	return raw
}

// ExportQuestionDrafts returns only an explicitly bound Host/profile's drafts.
// The owner is a caller-provided stable profile identity, never a session ID,
// transient port, cookie, or inferred global Host.
func (s *Store) ExportQuestionDrafts(owner string) json.RawMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	if questionDraftOwner(owner) != s.questionStorage.owner {
		return nil
	}
	return s.exportQuestionDraftsLocked()
}

func (s *Store) bindQuestionDraftOwnerLocked(owner string) {
	if owner != s.questionStorage.owner {
		s.questionStorage = questionDraftStorage{owner: owner, epoch: s.questionStorage.epoch + 1}
	}
}

// SetQuestionDraftPersistence installs the application's existing resume writer.
// Only explicit draft mutations schedule writes; rendering never calls it.
func (s *Store) SetQuestionDraftPersistence(owner string, save func(json.RawMessage) error) {
	s.questionWriter.mu.Lock()
	defer s.questionWriter.mu.Unlock()
	s.mu.Lock()
	s.bindQuestionDraftOwnerLocked(questionDraftOwner(owner))
	s.questionStorage.save = save
	s.mu.Unlock()
}

// RestoreQuestionDrafts imports once per explicit owner. It never creates a
// pending interaction and cannot overwrite a draft already edited in memory.
func (s *Store) RestoreQuestionDrafts(owner string, raw json.RawMessage) {
	s.questionWriter.mu.Lock()
	defer s.questionWriter.mu.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	key := questionDraftOwner(owner)
	s.bindQuestionDraftOwnerLocked(key)
	if key == "" || s.questionStorage.restored {
		return
	}
	s.questionStorage.restored = true
	var envelope questionDraftEnvelope
	if json.Unmarshal(raw, &envelope) != nil || envelope.Version != 1 || envelope.Owner != key {
		return
	}
	if s.questionStorage.deferred == nil {
		s.questionStorage.deferred = make(map[string]savedQuestionDraft)
	}
	for id, saved := range envelope.Drafts {
		if saved.SessionID == "" || saved.CallID == "" || id != questionCardID(saved.SessionID, saved.CallID) {
			continue
		}
		exists := false
		for _, p := range s.pending {
			exists = exists || p.AgentID == saved.SessionID && p.CallID == saved.CallID
		}
		if _, retained := s.questionStorage.deferred[id]; !exists && !retained {
			s.questionStorage.deferred[id] = saved
		}
	}
	s.questionStorage.last = append(json.RawMessage(nil), raw...)
}

func (s *Store) restoreQuestionDraftLocked(sessionID, callID string, draft *draftSession) {
	key := questionCardID(sessionID, callID)
	saved, ok := s.questionStorage.deferred[key]
	if !ok {
		return
	}
	delete(s.questionStorage.deferred, key)
	if saved.Fingerprint != questionDraftFingerprint(draft.req) || len(saved.Answers) != len(draft.req.Questions) || saved.Index < 0 || saved.Index >= len(draft.req.Questions) {
		return
	}
	for i, q := range draft.req.Questions {
		a := saved.Answers[i]
		if a.ID != q.ID || !q.MultiSelect && len(a.Selected) > 1 {
			return
		}
		for _, selected := range a.Selected {
			found := false
			for _, option := range q.Options {
				found = found || option.Label == selected
			}
			if !found {
				return
			}
		}
	}
	draft.index = saved.Index
	draft.wait.held, draft.wait.editing = saved.Wait == "waiting", saved.Wait == "editing"
	for i, q := range draft.req.Questions {
		a, value := draft.answers[q.ID], saved.Answers[i]
		a.selected, a.custom, a.skipped = append([]string(nil), value.Selected...), value.Custom, value.Skipped
		a.selectWire = ""
		if len(a.selected) > 0 {
			a.selectWire = a.selected[0]
		}
		for label, checked := range a.checks {
			*checked = containsLabel(a.selected, label)
		}
	}
}

func (s *Store) persistQuestionDrafts() {
	go func() { _ = s.FlushQuestionDraftPersistence() }()
}

// FlushQuestionDraftPersistence serializes with mutation writes and exports
// only after taking that lock. Queued older mutations therefore write the
// latest snapshot, never an earlier captured value after the final quit save.
// Call before the application's final ExportQuestionDrafts/resume transaction.
func (s *Store) FlushQuestionDraftPersistence() error {
	s.questionWriter.mu.Lock()
	defer s.questionWriter.mu.Unlock()
	s.mu.Lock()
	storage := &s.questionStorage
	raw, save, epoch := s.exportQuestionDraftsLocked(), storage.save, storage.epoch
	if save == nil || len(raw) == 0 || bytes.Equal(raw, storage.last) {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	err := save(raw)
	s.mu.Lock()
	if s.questionStorage.epoch != epoch {
		s.mu.Unlock()
		return err
	}
	previous := s.questionStorage.err
	s.questionStorage.err = ""
	if err != nil {
		s.questionStorage.err = "Could not save question draft: " + err.Error()
	} else {
		s.questionStorage.last = append(json.RawMessage(nil), raw...)
	}
	changed := previous != s.questionStorage.err
	s.mu.Unlock()
	if changed {
		s.fire()
	}
	return err
}

func (s *Store) pruneQuestionDraftsLocked(sessionID string, active map[string]bool) {
	changed := false
	for key, saved := range s.questionStorage.deferred {
		if saved.SessionID != sessionID || active[saved.CallID] {
			continue
		}
		live := false
		for _, pending := range s.pending {
			live = live || pending.AgentID == sessionID && pending.CallID == saved.CallID
		}
		if !live {
			delete(s.questionStorage.deferred, key)
			changed = true
		}
	}
	if changed {
		s.persistQuestionDrafts()
	}
}
