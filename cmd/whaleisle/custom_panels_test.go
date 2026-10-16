package main

import (
	"testing"
	panel "whaleisle/internal/store/custompanel"
	dshbotstore "whaleisle/internal/store/dshbot"
	"whaleisle/internal/store/session"
)

func TestCustomPanelCanonicalBindingAndCompletedReply(t *testing.T) {
	roster := dshbotstore.Snapshot{Origin: "host", Rows: []dshbotstore.Row{{ID: "a", SessionID: "session-a"}, {ID: "b", SessionID: "session-b"}}}
	scope, _ := customPanelBotScope("stable-host", "host", "session-b", roster)
	if scope.Bot != "b" {
		t.Fatal(scope)
	}
	scope, _ = customPanelBotScope("stable-host", "host", "unrelated", roster)
	if scope.Bot != "" {
		t.Fatal("unrelated chat inherited Bot panels")
	}
	scope, _ = customPanelBotScope("stable-host", "other-host", "session-a", roster)
	if scope.Bot != "" {
		t.Fatal("Host crossing")
	}
	// The existing workspace namespace survives a supervised Host listen-port change.
	before, _ := customPanelBotScope("stable-host", "host", "session-a", roster)
	roster.Origin = "restarted-host-port"
	after, _ := customPanelBotScope("stable-host", "restarted-host-port", "session-a", roster)
	if before != after || after.Host != "stable-host" {
		t.Fatal("ephemeral endpoint changed persisted Panel identity", before, after)
	}
	stale, _ := customPanelBotScope("stable-host", "host", "session-a", roster)
	if stale.Bot != "" {
		t.Fatal("stale runtime endpoint accepted")
	}
	messages := []session.Message{{Role: "assistant", Kind: session.MessageKindText, Body: panel.Example}, {Role: "assistant", Kind: session.MessageKindText, Body: "partial", Streaming: true}}
	d, err := latestCustomPanelReply(messages)
	if err != nil || d.Title != "Daily dashboard" {
		t.Fatal(d, err)
	}
}
