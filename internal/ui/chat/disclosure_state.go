package chat

// Shared explicit disclosure choices for transcript groups, retry details and
// usage panels. Workflow runs use their own status-driven disclosure state.
func workflowExpanded(state *map[string]bool, key string, initial bool) bool {
	if state == nil {
		return initial
	}
	if *state == nil {
		*state = map[string]bool{}
	}
	v, ok := (*state)[key]
	if !ok {
		(*state)[key] = initial
		return initial
	}
	return v
}

func workflowToggle(state *map[string]bool, key string, initial bool) {
	if state != nil {
		v := workflowExpanded(state, key, initial)
		(*state)[key] = !v
	}
}
