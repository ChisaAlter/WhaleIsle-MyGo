package session

import (
	"bytes"
	"encoding/json"
	"strings"
)

// SessionPresentation is the Host's sessionListMetadata.presentation. It is
// navigation/composer presentation only, never execution or file authority.
type SessionPresentation struct {
	Owner            string `json:"owner"`
	Title            string `json:"title"`
	WorkingDirectory string `json:"workingDirectory,omitempty"`
	Composer         string `json:"composer,omitempty"`
}

func presentationFromValues(raw json.RawMessage) (*SessionPresentation, bool) {
	if !bytes.Contains(raw, []byte(`"sessionListMetadata"`)) {
		return nil, false
	}
	var values map[string]json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return nil, false
	}
	metadata, present := values["sessionListMetadata"]
	if !present {
		return nil, false // Sparse availability hints do not withdraw known data.
	}
	return presentationFromMetadata(metadata)
}

func presentationFromMetadata(raw json.RawMessage) (*SessionPresentation, bool) {
	var metadata struct {
		Presentation *SessionPresentation `json:"presentation"`
	}
	if json.Unmarshal(raw, &metadata) != nil {
		return nil, false
	}
	return metadata.Presentation, true
}

func mergeSessionPresentation(next *Item, old Item) {
	// Cached watermarks belong to persisted storage, not the live connection's
	// sequence space. A cold list hint must not overwrite connected authority.
	keep := !next.presentationKnown || old.presentationKnown &&
		(old.presentationKind == "sequenced" && next.presentationKind == "cached" ||
			old.presentationKind == next.presentationKind && old.presentationSeq > next.presentationSeq)
	if keep {
		next.Presentation = old.Presentation
		next.presentationKnown, next.presentationKind, next.presentationSeq = old.presentationKnown, old.presentationKind, old.presentationSeq
	}
}

func (s *Store) applySessionPresentationValuesLocked(id, kind string, seq float64, raw json.RawMessage) {
	presentation, known := presentationFromValues(raw)
	if known {
		s.applySessionPresentationLocked(id, kind, seq, presentation)
	}
}

func (s *Store) applySessionPresentationLocked(id, kind string, seq float64, presentation *SessionPresentation) {
	s.invalidateViewLocked()
	for i := range s.items {
		if s.items[i].SessionID != id {
			continue
		}
		next := Item{Presentation: presentation, presentationKnown: true, presentationKind: kind, presentationSeq: seq}
		mergeSessionPresentation(&next, s.items[i])
		s.items[i].Presentation = next.Presentation
		s.items[i].presentationKnown, s.items[i].presentationKind, s.items[i].presentationSeq = next.presentationKnown, next.presentationKind, next.presentationSeq
		if s.items[i].Cwd == "" && next.Presentation != nil && strings.TrimSpace(next.Presentation.WorkingDirectory) != "" {
			s.items[i].Cwd = strings.TrimSpace(next.Presentation.WorkingDirectory)
		}
		if s.items[i].Title == "" && next.Presentation != nil && strings.TrimSpace(next.Presentation.Title) != "" {
			s.items[i].Title = strings.TrimSpace(next.Presentation.Title)
		}
		return
	}
	if id != "" && presentation != nil {
		s.items = append(s.items, Item{
			SessionID:         id,
			Title:             strings.TrimSpace(presentation.Title),
			Cwd:               strings.TrimSpace(presentation.WorkingDirectory),
			Presentation:      presentation,
			presentationKnown: true,
			presentationKind:  kind,
			presentationSeq:   seq,
		})
	}
}

// EnsureSessionPresentation seeds or updates a session's presentation in the
// store before session/list completes, so selecting a known plugin-owned
// session (such as the Whale Assistant) renders its header and composer posture
// immediately even while a large session directory is still being listed.
func (s *Store) EnsureSessionPresentation(id, owner, title, cwd string) {
	id = strings.TrimSpace(id)
	owner = strings.TrimSpace(owner)
	if s == nil || id == "" || owner == "" {
		return
	}
	pres := &SessionPresentation{
		Owner:            owner,
		Title:            strings.TrimSpace(title),
		WorkingDirectory: strings.TrimSpace(cwd),
	}
	s.mu.Lock()
	s.applySessionPresentationLocked(id, "cached", 0, pres)
	s.mu.Unlock()
	s.fire()
}

func (s *Store) applySessionPresentationControlLocked(raw json.RawMessage) {
	s.invalidateViewLocked()
	var frame struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil {
		return
	}
	if frame.Type == "projection" && frame.Key == "sessionListMetadata" {
		if presentation, ok := presentationFromMetadata(frame.Value); ok {
			s.applySessionPresentationLocked(frame.SessionID, "sequenced", frame.Seq, presentation)
		}
	}
	if frame.Type == "baseline" {
		var baseline struct {
			Projections map[string]agentNavigationProjection
		}
		if json.Unmarshal(frame.Value, &baseline) == nil {
			for id, projection := range baseline.Projections {
				if projection.AsOfSeq != nil {
					s.applySessionPresentationValuesLocked(id, "sequenced", *projection.AsOfSeq, projection.Values)
				}
			}
		}
	}
}

func (s *Store) resetSessionPresentationsLocked() {
	s.invalidateViewLocked()
	for i := range s.items {
		s.items[i].Presentation = nil
		s.items[i].presentationKnown, s.items[i].presentationKind, s.items[i].presentationSeq = false, "", 0
	}
}
