package harness

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartingLogNotificationsAreCoalesced(t *testing.T) {
	var notified atomic.Int32
	s := &Supervisor{opts: Options{OnChange: func(Status) { notified.Add(1) }}}
	s.status = Status{State: Starting}
	stop := time.Now().Add(time.Second)
	lines := 0
	for time.Now().Before(stop) {
		s.log("plugin line")
		lines++
		time.Sleep(time.Millisecond)
	}
	time.Sleep(2 * progressInterval)
	got := notified.Load()
	if got == 0 {
		t.Fatal("log lines while Starting must still reach the boot page")
	}
	// One second of logging at ≤5/s, plus the trailing flush.
	if got > 7 {
		t.Fatalf("%d notifications for %d lines in 1 s; want at most 5 per second", got, lines)
	}
	if logs := s.Logs(); len(logs) == 0 || logs[len(logs)-1] != "plugin line" {
		t.Fatal("coalescing must not drop the log itself")
	}
}

func TestStartingNotificationStopsAfterStateChange(t *testing.T) {
	var notified atomic.Int32
	s := &Supervisor{opts: Options{OnChange: func(st Status) {
		if st.State == Starting {
			notified.Add(1)
		}
	}}}
	s.status = Status{State: Starting}
	s.log("late line")
	s.mu.Lock()
	s.status = Status{State: Stopped}
	s.mu.Unlock()
	time.Sleep(2 * progressInterval)
	if notified.Load() != 0 {
		t.Fatal("a pending progress notification must not resurrect Starting after the state changed")
	}
}

func TestRedeemRetriesTransientFailures(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch hits.Add(1) {
		case 1:
			hj, _ := w.(http.Hijacker)
			conn, _, _ := hj.Hijack()
			if tcp, ok := conn.(*net.TCPConn); ok {
				_ = tcp.SetLinger(0)
			}
			_ = conn.Close()
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable)
		default:
			http.SetCookie(w, &http.Cookie{Name: cookiePrefix + "x", Value: "v"})
			w.WriteHeader(http.StatusFound)
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ep, err := redeemWithRetry(ctx, srv.Client(), srv.URL+"/?token=abc")
	if err != nil {
		t.Fatalf("a reset then a 503 should heal within three retries: %v", err)
	}
	if ep.Cookie != cookiePrefix+"x=v" || hits.Load() != 3 {
		t.Fatalf("cookie %q after %d exchanges", ep.Cookie, hits.Load())
	}
}

func TestRedeemDoesNotRetryRejection(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	if _, err := redeemWithRetry(context.Background(), srv.Client(), srv.URL+"/?token=abc"); err == nil {
		t.Fatal("a rejected token must fail the start")
	}
	if hits.Load() != 1 {
		t.Fatalf("rejected token exchanged %d times, want 1", hits.Load())
	}
}

func TestRedeemGivesUpAfterThreeRetries(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	if _, err := redeemWithRetry(context.Background(), srv.Client(), srv.URL+"/?token=abc"); err == nil {
		t.Fatal("a Host that never answers with a cookie must fail the start")
	}
	if hits.Load() != 4 {
		t.Fatalf("exchanges=%d, want the first plus three retries", hits.Load())
	}
}
