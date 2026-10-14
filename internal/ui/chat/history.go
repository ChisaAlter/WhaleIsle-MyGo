package chat

import "whaleisle/internal/store/session"

// ShowLoadEarlier reports whether the chat column should offer session/page.
func ShowLoadEarlier(snap session.ViewState) bool {
	return snap.HistoryHasMore || snap.HistoryPageBusy
}
