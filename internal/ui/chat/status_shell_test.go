package chat

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/store/session"
	"whaleisle/internal/ui/i18n"
)

func TestStatusSurvivesEmptyConversationAndReconnect(t *testing.T) {
	reconnect := 0
	m := Model{Snapshot: session.ViewState{LastAction: "history unavailable"}, ConnectionBanner: "connection offline", OnReconnect: func() { reconnect++ }}
	tt := ui.NewTester(func(c *ui.Context) { ui.Column(c).Fill().Children(func() { StatusView(c, m) }) }, 600, 300)
	for _, text := range []string{"history unavailable", "connection offline"} {
		if !strings.Contains(strings.Join(tt.Texts(), "\n"), text) {
			t.Fatalf("empty conversation hid %q", text)
		}
	}
	if err := tt.Click(i18n.T(i18n.KeyChatReconnect)); err != nil || reconnect != 1 {
		t.Fatalf("reconnect callback: calls=%d err=%v", reconnect, err)
	}
}

func TestShellOwnedStatusDoesNotDuplicateInsideTranscript(t *testing.T) {
	m := Model{Snapshot: session.ViewState{LastAction: "one action"}, ConnectionBanner: "one connection"}
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Children(func() {
			StatusView(c, m)
			m.SuppressStatus = true
			ui.Box(c).Grow(1).Children(func() { View(c, m) })
		})
	}, 600, 300)
	for _, text := range []string{"one action", "one connection"} {
		count := 0
		for _, got := range tt.Texts() {
			if got == text {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("%q rendered %d times", text, count)
		}
	}
}
