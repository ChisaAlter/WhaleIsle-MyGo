// Package testwait bounds a test's wait on asynchronous work, and makes the
// failure say which of two things happened: the producer stopped, so the state
// can never arrive, or the budget expired while the producer was still running,
// so the state may still arrive.
//
// A hand-rolled `deadline := time.Now().Add(N * time.Second)` picks N as a
// multiple of one observed passing run, so its failure reports "timed out"
// whether the goroutine died or the machine was slow, and every occurrence
// needs a second investigation to tell them apart.
//
// Imported by tests only.
package testwait

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// TestingT is the part of *testing.T this package needs, so its messages can be
// asserted with a recorder instead of by failing a real test.
type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Producer is the goroutine whose progress a wait depends on. Done closes when
// it exits, for any reason; Err names why, when the test can report it.
type Producer struct {
	Name string
	Done <-chan struct{}
	Err  func() error
}

// Wait describes one expectation before it is checked.
type Wait struct {
	// What the test waits for, in the language of the fixture: "two pending
	// questions", never "condition".
	What string
	// Producer is the goroutine that must make the check true. Nil when the
	// test cannot observe one; the failure then says so instead of claiming a
	// distinction it did not have.
	Producer *Producer
	// State returns the predicate's inputs, never a verdict.
	State func() string
	// Budget is how long this wait may take. It is a parameter, never a
	// package default, so a test cannot inherit a bound by accident.
	Budget time.Duration
}

const poll = 2 * time.Millisecond

// Until blocks until check holds, the producer exits, or the budget expires.
// Both failures print the state the check was reading and every goroutine: the
// two questions a stalled wait leaves are "who stopped" and "what was it
// looking at".
func Until(t TestingT, w Wait, check func() bool) {
	t.Helper()
	if check() {
		return
	}
	var done <-chan struct{}
	if w.Producer != nil {
		done = w.Producer.Done
	}
	timer := time.NewTimer(w.Budget)
	defer timer.Stop()
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			if check() {
				return
			}
			t.Fatalf("%s", report(w, producerStopped))
			return
		case <-timer.C:
			if check() {
				return
			}
			t.Fatalf("%s", report(w, budgetSpent))
			return
		case <-ticker.C:
			if check() {
				return
			}
		}
	}
}

type outcome int

const (
	producerStopped outcome = iota
	budgetSpent
)

func report(w Wait, o outcome) string {
	var b strings.Builder
	switch o {
	case producerStopped:
		fmt.Fprintf(&b, "waiting for %s: nothing was produced and nothing can be, because %s exited", w.What, w.Producer.Name)
		if err := producerErr(w.Producer); err != nil {
			fmt.Fprintf(&b, ": %v", err)
		} else {
			b.WriteString(" without reporting an error")
		}
		b.WriteString(" (a longer budget would not help)")
	case budgetSpent:
		if w.Producer == nil {
			fmt.Fprintf(&b, "waiting for %s: not true after %s, and this wait named no producer, so the test cannot say whether the goroutine that must make it true is stopped or slow", w.What, w.Budget)
		} else {
			fmt.Fprintf(&b, "waiting for %s: not true after %s, but %s is still running, so this is late rather than never", w.What, w.Budget, w.Producer.Name)
		}
	}
	if w.State != nil {
		fmt.Fprintf(&b, "\nstate: %s", w.State())
	}
	buf := make([]byte, 1<<20)
	fmt.Fprintf(&b, "\ngoroutines:\n%s", buf[:runtime.Stack(buf, true)])
	return b.String()
}

func producerErr(p *Producer) error {
	if p == nil || p.Err == nil {
		return nil
	}
	return p.Err()
}
