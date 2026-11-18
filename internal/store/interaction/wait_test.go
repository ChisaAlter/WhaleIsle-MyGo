package interaction

import (
	"testing"
	"time"
)

// stateCeiling bounds how long a test watches for a state the store reaches on
// its own. It is a stuck detector, not a timing budget: a loaded race-detector
// runner is slower than a developer machine by an order of magnitude, so a
// ceiling that doubles as an assertion fails on scheduling, and the failure then
// reads as a product bug. Sixty seconds is far past anything these fixtures need
// while still ending a wedged test.
const stateCeiling = 60 * time.Second

// statePoll covers transitions that do not fire the store's refresh hook.
const statePoll = 50 * time.Millisecond

// awaitState blocks until condition holds, driven by the store's own refresh
// hook rather than by elapsed time. describe supplies the failure text.
func awaitState(t *testing.T, s *Store, describe func() string, condition func() bool) {
	t.Helper()
	if condition() {
		return
	}
	changed := make(chan struct{}, 1)
	s.Attach(func() {
		select {
		case changed <- struct{}{}:
		default:
		}
	})
	t.Cleanup(func() { s.Attach(nil) })
	ceiling := time.NewTimer(stateCeiling)
	defer ceiling.Stop()
	poll := time.NewTicker(statePoll)
	defer poll.Stop()
	for !condition() {
		select {
		case <-changed:
		case <-poll.C:
		case <-ceiling.C:
			t.Fatalf("state never arrived within %s, which is a stuck detector rather than a timeout budget: %s", stateCeiling, describe())
		}
	}
}

// awaitQuestionFixture waits for a question state the fixture drives
// asynchronously. It polls because its callers do not all name the store, so the
// ceiling is the only time bound and it is deliberately generous.
func awaitQuestionFixture(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(stateCeiling)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatalf("question condition never arrived within %s, which is a stuck detector rather than a timeout budget", stateCeiling)
		}
		time.Sleep(time.Millisecond)
	}
}
