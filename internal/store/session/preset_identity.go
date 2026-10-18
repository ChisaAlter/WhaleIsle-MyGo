package session

func permissionOptionByValue(value string, options []PermissionOption) (PermissionOption, bool) {
	for _, option := range options {
		if option.Value == value {
			return option, true
		}
	}
	return PermissionOption{}, false
}
func agentPresetOptionByID(id string, options []AgentPresetOption) (AgentPresetOption, bool) {
	for _, option := range options {
		if option.ID == id {
			return option, true
		}
	}
	return AgentPresetOption{}, false
}

// PresetPickerOptions returns catalog identities separately from display labels.
// Permission availability remains gated by the selected session projection.
func (s *Store) PresetPickerOptions(sessionID string, developerTools bool) ([]PermissionOption, []AgentPresetOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.selectedID != sessionID {
		return nil, nil
	}
	var permission []PermissionOption
	if s.permissionProjectionCurrentLocked() {
		permission = append(permission, s.permissionOptions...)
	}
	var agents []AgentPresetOption
	for _, o := range s.agentPresetOptions {
		if developerTools || !o.RequiresCodingTools {
			agents = append(agents, o)
		}
	}
	return permission, agents
}
