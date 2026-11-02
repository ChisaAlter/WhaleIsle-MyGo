package session

import "encoding/json"

type ComposerImageLimits struct {
	MediaTypes           []string `json:"mediaTypes"`
	MaxImagesPerMessage  int64    `json:"maxImagesPerMessage"`
	MaxImageBytes        int64    `json:"maxImageBytes"`
	MaxMessageImageBytes int64    `json:"maxMessageImageBytes"`
}
type composerImageLimitState struct {
	value      *ComposerImageLimits
	sessionID  string
	generation uint64
	seq        float64
}

func (s *Store) ComposerImageLimits(id string) *ComposerImageLimits {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.imageLimits
	if id == "" || id != s.selectedID || p.sessionID != id || p.generation != s.followGeneration || p.value == nil {
		return nil
	}
	out := *p.value
	out.MediaTypes = append([]string(nil), out.MediaTypes...)
	return &out
}
func (s *Store) applyImageLimitValuesLocked(values json.RawMessage, seq float64) {
	var all map[string]json.RawMessage
	if json.Unmarshal(values, &all) != nil {
		return
	}
	s.applyImageLimitsLocked(all["imageLimits"], seq)
}
func (s *Store) applyImageLimitsLocked(raw json.RawMessage, seq float64) {
	s.invalidateViewLocked()
	p := &s.imageLimits
	if p.sessionID != s.selectedID || p.generation != s.followGeneration {
		*p = composerImageLimitState{sessionID: s.selectedID, generation: s.followGeneration}
	}
	if seq < p.seq {
		return
	}
	if len(raw) == 0 || string(raw) == "null" {
		p.value = nil
		p.seq = seq
		return
	}
	var shape struct {
		MediaTypes                                               []string
		MaxImagesPerMessage, MaxImageBytes, MaxMessageImageBytes *int64
	}
	if json.Unmarshal(raw, &shape) != nil || shape.MediaTypes == nil || shape.MaxImagesPerMessage == nil || shape.MaxImageBytes == nil || shape.MaxMessageImageBytes == nil {
		return
	}
	if *shape.MaxImagesPerMessage < 0 || *shape.MaxImageBytes < 0 || *shape.MaxMessageImageBytes < 0 {
		return
	}
	p.value = &ComposerImageLimits{shape.MediaTypes, *shape.MaxImagesPerMessage, *shape.MaxImageBytes, *shape.MaxMessageImageBytes}
	p.seq = seq
}
func (s *Store) applyImageLimitControlLocked(raw json.RawMessage) {
	s.invalidateViewLocked()
	var f struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &f) != nil || f.Type != "projection" || f.Key != "imageLimits" || f.SessionID != s.selectedID {
		return
	}
	s.applyImageLimitsLocked(f.Value, f.Seq)
}
