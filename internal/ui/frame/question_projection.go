package frame

import "whaleisle/internal/store/interaction"

// Pending questions remain reachable while history is empty or still loading.
func hasSessionInteractions(store *interaction.Store, sessionID string) bool {
	if store == nil {
		return false
	}
	snap := store.SnapshotForAgent(sessionID)
	return len(snap.Pending) > 0 || len(snap.QuestionHistory) > 0 || snap.QuestionStatus != ""
}

// A continued late-answer card is history, not a currently blocked response.
func hasPendingStopInteraction(store *interaction.Store, sessionID string) bool {
	if store == nil {
		return false
	}
	for _, pending := range store.SnapshotForAgent(sessionID).Pending {
		if !pending.Continued {
			return true
		}
	}
	return false
}
