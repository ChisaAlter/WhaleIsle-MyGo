package main

import (
	"testing"

	"whaleisle/internal/desktop/taskcontrol"
)

func TestQuitSkipsConfirmationOnlyForAnUninspectableHost(t *testing.T) {
	clean := taskcontrol.Inspection{OK: true, HostGeneration: "g1", Coverage: map[string]string{"sessions": "ok"}}
	uninspectable := taskcontrol.Inspection{OK: false, Unknown: "Host task-control inspect unavailable"}
	work := taskcontrol.Inspection{OK: true, HostGeneration: "g1", ActiveWork: []taskcontrol.Work{{Kind: "turn"}}}

	if quitSkipsConfirmation("quit", uninspectable, 0) != true {
		t.Fatal("an uninspectable Host with no local work must not hold the close")
	}
	if quitSkipsConfirmation("quit", uninspectable, 2) {
		t.Fatal("local running work still needs the answer")
	}
	if quitSkipsConfirmation("quit", clean, 0) {
		t.Fatal("a clean inspection never reaches the confirmation path")
	}
	if quitSkipsConfirmation("quit", work, 0) {
		t.Fatal("Host work must still be confirmed")
	}
	for _, operation := range []string{"stop", "restart", "update", "component"} {
		if quitSkipsConfirmation(operation, uninspectable, 0) {
			t.Fatalf("%s must keep its confirmation", operation)
		}
	}
}
