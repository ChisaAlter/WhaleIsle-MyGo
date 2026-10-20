package bridgealerts

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"whaleisle/internal/store/storelocale"
)

type fakeSource struct {
	mu               sync.Mutex
	scope            string
	batch            Batch
	err              error
	calls, opens     int
	started, release chan struct{}
}

func (s *fakeSource) Current(context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scope, nil
}
func (s *fakeSource) Claim(ctx context.Context, key string) (Batch, error) {
	s.mu.Lock()
	s.calls++
	b, e, start, release := s.batch, s.err, s.started, s.release
	s.mu.Unlock()
	if start != nil {
		close(start)
		select {
		case <-release:
		case <-ctx.Done():
			return Batch{}, ctx.Err()
		}
	}
	return b, e
}
func (s *fakeSource) Open(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ctx.Err() != nil || key != s.scope {
		return errors.New("stale")
	}
	s.opens++
	return nil
}
func alertBatch(seq int64) Batch {
	return Batch{Sequence: seq, Alerts: []Record{{Sequence: seq, Alert: Alert{Name: "Fixture", User: "Synthetic", Kind: "renews", Window: "Week", Used: scalar("42.5"), ResetsAt: "2030-01-01T00:00:00Z"}}}}
}
func scalar(s string) Scalar { return Scalar(s) }
func TestBridgeAlertsLifecycleFixture(t *testing.T) {
	s := &fakeSource{scope: "process-one", batch: alertBatch(2)}
	m := New(s)
	m.Poll()
	a := m.Snapshot()
	if len(a.Alerts) != 1 || !strings.Contains(a.Alerts[0].Body, "42.5%") {
		t.Fatalf("missing real fields: %#v", a)
	}
	ticket := a.Alerts[0].Ticket
	m.Poll()
	if len(m.Snapshot().Alerts) != 1 {
		t.Fatal("duplicate sequence")
	}
	m.DeliveryFailed(ticket)
	if !m.Snapshot().Fallback {
		t.Fatal("notification failure lost visible fallback")
	}
	if err := m.Open(ticket); err != nil || s.opens != 1 {
		t.Fatalf("same process open: %v", err)
	}
	s.scope = "process-two"
	calls := s.calls
	m.Observe()
	if m.Current(ticket) || s.calls != calls {
		t.Fatal("read-only observation failed to retire old process")
	}
	s.batch = alertBatch(1)
	m.Poll()
	if len(m.Snapshot().Alerts) != 1 || m.Snapshot().Alerts[0].Ticket.Sequence != 1 {
		t.Fatal("PID replacement did not reset sequence")
	}
	if err := m.Open(ticket); err == nil || s.opens != 1 {
		t.Fatal("old click retargeted replacement")
	}
	s.scope = ""
	m.Poll()
	if len(m.Snapshot().Alerts) != 0 {
		t.Fatal("stopped instance retained live notices")
	}
	s.scope = "process-two"
	m.Poll()
	if len(m.Snapshot().Alerts) != 0 {
		t.Fatal("temporary missing state replayed claimed sequence")
	}
	m.Close()
	m.Poll()
	if err := m.Open(m.Snapshot().Manage); err == nil {
		t.Fatal("closed window accepted ticket")
	}
}
func TestBridgeAlertsUncertainAndInterruptedFixture(t *testing.T) {
	s := &fakeSource{scope: "one", err: errors.New("claim outcome unknown")}
	m := New(s)
	m.Poll()
	m.Poll()
	if s.calls != 1 || !m.Snapshot().Paused || m.Snapshot().Status == "" {
		t.Fatal("uncertain claim replayed or invisible")
	}
	paused := m.Snapshot().Manage
	s.scope = "replacement"
	if m.Continue(paused) == nil {
		t.Fatal("continue adopted replacement")
	}
	s.scope = "one"
	if m.Continue(paused) != nil || m.Snapshot().Paused {
		t.Fatal("explicit continue failed")
	}
	s.scope = "two"
	s.err = ErrNotClaimed
	m.Poll()
	m.Poll()
	if s.calls != 3 || m.Snapshot().Paused {
		t.Fatal("preflight failure must remain safely retryable")
	}
	s.err = nil
	s.batch = alertBatch(1)
	s.started = make(chan struct{})
	s.release = make(chan struct{})
	done := make(chan struct{})
	go func() { m.Poll(); close(done) }()
	<-s.started
	m.Poll()
	if s.calls != 4 {
		t.Fatal("concurrent claim")
	}
	s.mu.Lock()
	s.scope = "three"
	s.mu.Unlock()
	close(s.release)
	<-done
	if len(m.Snapshot().Alerts) != 0 {
		t.Fatal("late previous process result published")
	}
	s.started = nil
	s.release = nil
	m.Poll()
	n := m.Snapshot()
	if len(n.Alerts) != 1 {
		t.Fatal("new process failed to reset")
	}
	m.Close()
	if m.Current(n.Alerts[0].Ticket) {
		t.Fatal("close left callback live")
	}
}
func TestBridgeAlertsFormatAndMalformedFixture(t *testing.T) {
	for _, tc := range []struct {
		a    Alert
		want string
	}{
		{Alert{Kind: "expires", Credits: scalar("3"), ResetsAt: "2030-02-03T04:05:06Z"}, "3 次额度重置"},
		{Alert{Balance: scalar("$2.10")}, "$2.10"},
		{Alert{Window: "Day", Used: scalar("0")}, "0%"},
		{Alert{Window: "Day"}, storelocale.T(storelocale.KeyStoreBridgealertsUsageNotProvided)},
	} {
		_, body := Format(tc.a)
		if !strings.Contains(body, tc.want) {
			t.Fatalf("%q lacks %q", body, tc.want)
		}
	}
	s := &fakeSource{scope: "one", batch: Batch{Sequence: 1, Alerts: []Record{{Sequence: 2}}}}
	m := New(s)
	m.Poll()
	m.Poll()
	if !m.Snapshot().Paused || s.calls != 1 {
		t.Fatal("malformed consumed queue was replayed")
	}
}
