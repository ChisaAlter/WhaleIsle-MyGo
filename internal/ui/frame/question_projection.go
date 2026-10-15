package frame

import "whaleisle/internal/store/interaction"

// Pending questions remain reachable while history is empty or still loading.
func hasSessionInteractions(store *interaction.Store, sessionID string) bool {
	if store == nil {
		return false
	}
	snap := store.SnapshotForAgent(sessionID)
	return len(snap.Pending) > 0 || snap.QuestionStatus != ""
}
