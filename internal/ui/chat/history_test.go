package chat

import (
	"testing"

	"whaleisle/internal/store/session"
)

func TestShowLoadEarlier(t *testing.T) {
	if ShowLoadEarlier(session.ViewState{}) {
		t.Fatal("empty snap should hide")
	}
	if !ShowLoadEarlier(session.ViewState{HistoryHasMore: true}) {
		t.Fatal("hasMore should show")
	}
	if !ShowLoadEarlier(session.ViewState{HistoryPageBusy: true}) {
		t.Fatal("busy should show")
	}
}
