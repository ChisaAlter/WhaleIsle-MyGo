package testwait

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// recorder collects the failure text so a test can assert what a real timeout
// would have printed, without failing this test to see it.
type recorder struct{ fatals []string }

func (r *recorder) Helper() {}

func (r *recorder) Fatalf(format string, args ...any) {
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

func (r *recorder) last() string {
	if len(r.fatals) == 0 {
		return ""
	}
	return r.fatals[len(r.fatals)-1]
}

func TestUntilReturnsWhenTheCheckHolds(t *testing.T) {
	r := &recorder{}
	ready := make(chan struct{})
	state := func() string { return "unused" }
	go func() { time.Sleep(5 * time.Millisecond); close(ready) }()
	done := make(chan struct{})
	Until(r, Wait{What: "the signal", Producer: &Producer{Name: "the test goroutine", Done: done}, State: state, Budget: 5 * time.Second}, func() bool {
		select {
		case <-ready:
			return true
		default:
			return false
		}
	})
	if len(r.fatals) != 0 {
		t.Fatalf("a satisfied wait failed: %q", r.last())
	}
}

// A producer that stopped is the one case a longer budget cannot fix, so the
// message has to say that and name it, instead of reporting a timeout.
func TestUntilNamesTheProducerThatStopped(t *testing.T) {
	r := &recorder{}
	producerDone := make(chan struct{})
	boom := errors.New("socket closed by the peer")
	time.AfterFunc(10*time.Millisecond, func() { close(producerDone) })
	Until(r, Wait{
		What:     "two pending questions",
		Producer: &Producer{Name: "the events pump", Done: producerDone, Err: func() error { return boom }},
		State:    func() string { return "pending=0 status=\"idle\"" },
		Budget:   5 * time.Second,
	}, func() bool { return false })

	got := r.last()
	for _, want := range []string{
		"waiting for two pending questions",
		"the events pump",
		"exited",
		"socket closed by the peer",
		"a longer budget would not help",
		"state: pending=0 status=\"idle\"",
		"goroutines:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("producer-stopped message is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "still running") {
		t.Fatalf("a stopped producer was reported as slow:\n%s", got)
	}
}

// The budget path has to admit the producer is alive: that is the difference
// between a late state and one that can never arrive.
func TestUntilSaysTheProducerIsStillRunningAtTheBudget(t *testing.T) {
	r := &recorder{}
	producerDone := make(chan struct{})
	defer close(producerDone)
	Until(r, Wait{
		What:     "a stream transition",
		Producer: &Producer{Name: "the session follow loop", Done: producerDone},
		State:    func() string { return "reconnecting=false messages=2" },
		Budget:   20 * time.Millisecond,
	}, func() bool { return false })

	got := r.last()
	for _, want := range []string{
		"not true after 20ms",
		"the session follow loop is still running",
		"late rather than never",
		"state: reconnecting=false messages=2",
		"goroutines:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("budget message is missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "nothing can be") {
		t.Fatalf("a live producer was reported as stopped:\n%s", got)
	}
}

// A wait that cannot observe its producer has to say so; silence would let the
// next reader believe the timeout already ruled the goroutine out.
func TestUntilAdmitsWhenNoProducerWasNamed(t *testing.T) {
	r := &recorder{}
	Until(r, Wait{What: "captured readers", Budget: 10 * time.Millisecond}, func() bool { return false })
	got := r.last()
	for _, want := range []string{"captured readers", "named no producer", "stopped or slow", "goroutines:"} {
		if !strings.Contains(got, want) {
			t.Fatalf("nameless-producer message is missing %q:\n%s", want, got)
		}
	}
}
