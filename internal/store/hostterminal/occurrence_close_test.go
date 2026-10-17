package hostterminal

import (
	"context"
	"errors"
	"testing"
	"time"
)

func closeFixture(t *testing.T) (*Store, *Store, *fixtureRemote) {
	t.Helper()
	r := fixture()
	root := &Store{}
	root.bind("host", r, true, nil)
	child := root.Occurrence("pane", "", "session")
	child.mu.Lock()
	e := child.entryLocked(terminalKey{"session", "selected"})
	e.desired = true
	e.phase = "connected"
	child.sessionLocked("session").active = "selected"
	_, cancel := context.WithCancel(context.Background())
	e.holdCancel = cancel
	child.mu.Unlock()
	t.Cleanup(func() { root.bind("disposed", nil, false, nil); child.bind("disposed", nil, false, nil) })
	return root, child, r
}
func awaitClose(t *testing.T, request *OccurrenceClose) error {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		done, err := request.Result()
		if done {
			return err
		}
		select {
		case <-deadline:
			t.Fatal("close did not finish")
			return nil
		case <-time.After(time.Millisecond):
		}
	}
}
func TestOccurrenceCloseExplicitRetiresOnlyTarget(t *testing.T) {
	root, child, r := closeFixture(t)
	sibling := root.Occurrence("other", "", "session")
	child.Show("", false)
	if r.count("terminal/close") != 0 {
		t.Fatal("hiding terminated process")
	}
	v := child.Snapshot("session")
	request := root.BeginOccurrenceClose("pane", child, v.Owner, "session")
	if err := awaitClose(t, request); err != nil {
		t.Fatal(err)
	}
	if root.ExistingOccurrence("pane") != nil || root.ExistingOccurrence("other") != sibling {
		t.Fatal("wrong occurrence retired")
	}
	if r.count("terminal/close") != 1 {
		t.Fatal("close must be one-shot")
	}
	r.mu.Lock()
	calls := append([]fixtureCall(nil), r.calls...)
	r.mu.Unlock()
	for _, c := range calls {
		if c.method == "terminal/close" && (c.args["agentId"] != "session" || c.args["id"] != "selected") {
			t.Fatalf("wrong target: %v", c.args)
		}
	}
	child.mu.Lock()
	e := child.entries[terminalKey{"session", "selected"}]
	held := e.desired || e.holdCancel != nil || e.attachment != nil
	child.mu.Unlock()
	if held {
		t.Fatal("closed occurrence retained hold or attachment")
	}
	if err := awaitClose(t, root.BeginOccurrenceClose("pane", child, v.Owner, "session")); err == nil {
		t.Fatal("stale pane may not close a replacement")
	}
	if r.count("terminal/close") != 1 {
		t.Fatal("stale close repeated RPC")
	}
}
func TestOccurrenceCloseFailureKeepsRetryablePaneWithoutHold(t *testing.T) {
	root, child, r := closeFixture(t)
	r.closeErr = errors.New("offline")
	v := child.Snapshot("session")
	if err := awaitClose(t, root.BeginOccurrenceClose("pane", child, v.Owner, "session")); err == nil {
		t.Fatal("missing failure")
	}
	if root.ExistingOccurrence("pane") != child {
		t.Fatal("failed close lost retry owner")
	}
	child.mu.Lock()
	e := child.entries[terminalKey{"session", "selected"}]
	held := e.desired || e.holdCancel != nil
	child.mu.Unlock()
	if held {
		t.Fatal("failed close kept heartbeat alive")
	}
	r.mu.Lock()
	r.closeErr = nil
	r.mu.Unlock()
	if err := awaitClose(t, root.BeginOccurrenceClose("pane", child, v.Owner, "session")); err != nil {
		t.Fatal(err)
	}
	if r.count("terminal/close") != 2 {
		t.Fatal("explicit retry missing")
	}
}
func TestOccurrenceCloseStaleHostAndBusyFailClosed(t *testing.T) {
	root, child, r := closeFixture(t)
	v := child.Snapshot("session")
	if err := awaitClose(t, root.BeginOccurrenceClose("pane", child, v.Owner+1, "session")); err == nil {
		t.Fatal("stale Host accepted")
	}
	child.mu.Lock()
	child.sessionLocked("session").busy = true
	child.mu.Unlock()
	if err := awaitClose(t, root.BeginOccurrenceClose("pane", child, v.Owner, "session")); err == nil {
		t.Fatal("in-flight creation may not be discarded")
	}
	if r.count("terminal/close") != 0 {
		t.Fatal("invalid request reached Host")
	}
}
