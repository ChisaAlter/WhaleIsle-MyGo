package session

import "encoding/json"

// A permission value belongs to one materialized session generation. In
// particular, a process catalog or saved default can never initialize it.
type permissionProjectionState struct {
	SessionID  string
	Generation uint64
	Seq        float64
	Value      string
	Present    bool
	Pending    bool
}

func (s *Store) permissionProjectionCurrentLocked() bool {
	p := s.permissionProjection
	return s.selectedID != "" && p.SessionID == s.selectedID &&
		p.Generation == s.followGeneration && p.Present
}

func (s *Store) permissionLabelLocked() string {
	if !s.permissionProjectionCurrentLocked() {
		return ""
	}
	return labelForPresetValue(s.permissionProjection.Value, s.permissionOptions)
}

func (s *Store) applyPermissionValuesLocked(values json.RawMessage, seq float64) {
	var all map[string]json.RawMessage
	if json.Unmarshal(values, &all) != nil {
		return
	}
	// Missing means this session has no permission capability, not that the
	// process default should be displayed as its current permission.
	s.applyPermissionSelectionLocked(all["permissions"], seq)
}

func (s *Store) applyPermissionSelectionLocked(raw json.RawMessage, seq float64) {
	p := &s.permissionProjection
	previous := *p
	if p.SessionID == s.selectedID && p.Generation == s.followGeneration {
		if seq < p.Seq {
			return
		}
	} else {
		*p = permissionProjectionState{SessionID: s.selectedID, Generation: s.followGeneration}
	}
	var selection struct {
		CurrentValue string `json:"currentValue"`
	}
	if len(raw) != 0 && string(raw) != "null" {
		if json.Unmarshal(raw, &selection) != nil || selection.CurrentValue == "" {
			if previous.SessionID != p.SessionID || previous.Generation != p.Generation || previous.Value != p.Value || previous.Present != p.Present || previous.Pending != p.Pending {
				s.invalidateViewLocked()
			}
			return
		}
	}
	p.Seq, p.Value, p.Present = seq, selection.CurrentValue, selection.CurrentValue != ""
	if previous.SessionID != p.SessionID || previous.Generation != p.Generation || previous.Value != p.Value || previous.Present != p.Present || previous.Pending != p.Pending {
		s.invalidateViewLocked()
	}
}

func (s *Store) applyPermissionControlLocked(raw json.RawMessage) {
	var frame struct {
		Type, SessionID, Key string
		Seq                  float64
		Value                json.RawMessage
	}
	if json.Unmarshal(raw, &frame) != nil || frame.Type != "projection" ||
		frame.SessionID != s.selectedID || frame.Key != "permissions" {
		return
	}
	s.applyPermissionSelectionLocked(frame.Value, frame.Seq)
}
