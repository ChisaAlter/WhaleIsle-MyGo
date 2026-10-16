package terminal

import "whaleisle/internal/store/hostterminal"

func (h *HostState) Occurrence(key string) *HostState {
	if h.occurrences == nil {
		h.occurrences = make(map[string]*HostState)
	}
	if h.occurrences[key] == nil {
		h.occurrences[key] = &HostState{}
	}
	return h.occurrences[key]
}
func (h *HostState) hideOccurrences(store *hostterminal.Store, visible map[string]bool) {
	for key, child := range h.occurrences {
		if !visible[key] {
			if owner := store.ExistingOccurrence(key); owner != nil {
				child.Observe(owner, "", false)
			}
		}
	}
}

// Hiding a leaf releases only its output attachment, retaining the Host process.
func (h *HostState) ReconcileOccurrences(store *hostterminal.Store, keys []string) {
	visible := make(map[string]bool, len(keys))
	for _, key := range keys {
		visible[key] = true
	}
	h.hideOccurrences(store, visible)
	store.Show("", false)
	h.drop()
}
