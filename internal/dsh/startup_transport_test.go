package dsh

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/coder/websocket"
)

var fastRetry = RetryPolicy{Initial: 5 * time.Millisecond, Max: 20 * time.Millisecond, Budget: 2 * time.Second}

func TestRetryTransientClassification(t *testing.T) {
	cases := []struct {
		name  string
		err   error
		retry bool
	}{
		{"reset", &TransportError{Op: "session/list", Err: syscall.ECONNRESET}, true},
		{"forcibly closed", &TransportError{Op: "session/list", Err: errors.New("wsarecv: An existing connection was forcibly closed by the remote host.")}, true},
		{"eof", &TransportError{Op: "session/list", Err: io.ErrUnexpectedEOF}, true},
		{"endpoint not ready", &Error{Code: "gateway/invocation-unavailable"}, true},
		{"service unavailable", &Error{Code: "gateway/service-unavailable"}, true},
		{"rate limited", &HTTPError{Op: "session/list", Status: 429}, true},
		{"server error", &HTTPError{Op: "session/list", Status: 503}, true},
		{"bad request", &HTTPError{Op: "session/list", Status: 400}, false},
		{"rejected", &Error{Code: "session/not-found"}, false},
		{"plain", errors.New("decode value"), false},
		{"cancelled", context.Canceled, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			err := RetryTransient(context.Background(), fastRetry, "session/list", func(context.Context) error {
				calls++
				if calls == 1 {
					return tc.err
				}
				return nil
			})
			if tc.retry && (err != nil || calls != 2) {
				t.Fatalf("transient: err=%v calls=%d, want a second successful attempt", err, calls)
			}
			if !tc.retry && (err == nil || calls != 1) {
				t.Fatalf("permanent: err=%v calls=%d, want one attempt and the error", err, calls)
			}
		})
	}
}

func TestRetryTransientBackoffIsCappedAndBudgeted(t *testing.T) {
	policy := RetryPolicy{Initial: 10 * time.Millisecond, Max: 40 * time.Millisecond, Budget: 200 * time.Millisecond}
	var stamps []time.Time
	start := time.Now()
	err := RetryTransient(context.Background(), policy, "session/list", func(context.Context) error {
		stamps = append(stamps, time.Now())
		return &HTTPError{Op: "session/list", Status: 503}
	})
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("a never-healing read must return its last error")
	}
	if elapsed > policy.Budget+100*time.Millisecond {
		t.Fatalf("retries ran %s, past the %s budget", elapsed, policy.Budget)
	}
	if len(stamps) < 4 {
		t.Fatalf("only %d attempts inside the budget", len(stamps))
	}
	for i := 1; i < len(stamps); i++ {
		gap := stamps[i].Sub(stamps[i-1])
		want := min(policy.Initial<<(i-1), policy.Max)
		if gap < want-2*time.Millisecond {
			t.Fatalf("attempt %d came after %s, want at least %s", i, gap, want)
		}
		if gap > policy.Max+60*time.Millisecond {
			t.Fatalf("attempt %d waited %s, past the %s cap", i, gap, policy.Max)
		}
	}
}

func TestRetryTransientHonorsContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	policy := RetryPolicy{Initial: time.Second, Max: time.Second, Budget: time.Minute}
	var calls atomic.Int32
	done := make(chan error, 1)
	go func() {
		done <- RetryTransient(ctx, policy, "session/list", func(context.Context) error {
			calls.Add(1)
			return &Error{Code: "gateway/invocation-unavailable"}
		})
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err == nil || calls.Load() != 1 {
			t.Fatalf("err=%v calls=%d", err, calls.Load())
		}
	case <-time.After(time.Second):
		t.Fatal("RetryTransient kept waiting after cancel")
	}
}

func TestRetryTransientNeverRetriesMutations(t *testing.T) {
	for _, method := range []string{"session/create", "session/submit", "workspace/initializeDefault", "$events/result", "unknown/read"} {
		var calls int
		err := RetryTransient(context.Background(), fastRetry, method, func(context.Context) error {
			calls++
			return &TransportError{Op: method, Err: syscall.ECONNRESET}
		})
		if err == nil || calls != 1 {
			t.Fatalf("%s: calls=%d err=%v; a method outside the allowlist runs exactly once", method, calls, err)
		}
		if IdempotentRead(method) {
			t.Fatalf("%s must not be on the read allowlist", method)
		}
	}
}

func TestStreamFailureEndpointNotReady(t *testing.T) {
	if !errors.Is(&StreamFailure{Code: "gateway/invocation-unavailable"}, ErrEndpointNotReady) {
		t.Fatal("invocation-unavailable error frame must read as ErrEndpointNotReady")
	}
	if errors.Is(&StreamFailure{Code: "session/not-found"}, ErrEndpointNotReady) {
		t.Fatal("an ordinary stream failure is not endpoint-not-ready")
	}
	for _, err := range []error{&StreamFailure{Code: "gateway/service-unavailable"}, ErrStreamEnd, ErrStreamOverflow} {
		if !Resubscribable(err) {
			t.Fatalf("%v should re-subscribe", err)
		}
	}
	if Resubscribable(ErrCarrierLost) || Resubscribable(&StreamFailure{Code: "x"}) {
		t.Fatal("carrier loss and real stream failures are not re-subscribed in place")
	}
}

func TestGenerationClientOwnsTransportAndClosesIdle(t *testing.T) {
	var mu sync.Mutex
	states := map[net.Conn]http.ConnState{}
	srv := httptest.NewUnstartedServer(rpcOK())
	srv.Config.ConnState = func(c net.Conn, s http.ConnState) {
		mu.Lock()
		states[c] = s
		mu.Unlock()
	}
	srv.Start()
	defer srv.Close()

	a := NewGenerationClient(Endpoint{Origin: srv.URL})
	b := NewGenerationClient(Endpoint{Origin: srv.URL})
	ta, tb := a.HTTP.Transport.(*http.Transport), b.HTTP.Transport.(*http.Transport)
	if ta == tb || ta == defaultHTTPClient.Transport {
		t.Fatal("each Host generation must own its transport")
	}
	if ta.IdleConnTimeout <= 0 || ta.IdleConnTimeout >= 5*time.Second {
		t.Fatalf("IdleConnTimeout %s must stay under Node's 5 s keep-alive", ta.IdleConnTimeout)
	}
	if dt := defaultHTTPClient.Transport.(*http.Transport).IdleConnTimeout; dt >= 5*time.Second {
		t.Fatalf("default IdleConnTimeout %s must stay under Node's 5 s keep-alive", dt)
	}
	if err := a.Call(context.Background(), "session/list", nil, nil); err != nil {
		t.Fatal(err)
	}
	idle := func() int {
		mu.Lock()
		defer mu.Unlock()
		n := 0
		for _, s := range states {
			if s == http.StateIdle {
				n++
			}
		}
		return n
	}
	deadline := time.Now().Add(time.Second)
	for idle() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if idle() == 0 {
		t.Fatal("the generation's connection should be pooled idle after a call")
	}
	a.Close()
	deadline = time.Now().Add(time.Second)
	for idle() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if idle() != 0 {
		t.Fatal("Close must drop the generation's idle connections")
	}
}

// TestCallReadHealsResetOnFirstPost is the sidebar's failure: the first POST
// lands on a socket the server resets. A read on the allowlist retries it.
func TestCallReadHealsResetOnFirstPost(t *testing.T) {
	var posts atomic.Int32
	ok := rpcOK()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if posts.Add(1) == 1 {
			resetRequest(t, w)
			return
		}
		ok.ServeHTTP(w, r)
	}))
	defer srv.Close()
	c := NewGenerationClient(Endpoint{Origin: srv.URL})
	defer c.Close()
	var out struct {
		Items []any `json:"items"`
	}
	if err := c.CallRead(context.Background(), "session/list", nil, &out); err != nil {
		t.Fatalf("reset first POST should heal: %v", err)
	}
	if posts.Load() != 2 {
		t.Fatalf("posts=%d, want one reset and one retry", posts.Load())
	}
	posts.Store(0)
	if err := c.CallRead(context.Background(), "session/create", nil, nil); err == nil {
		t.Fatal("a mutation must surface the reset instead of retrying")
	}
	if posts.Load() != 1 {
		t.Fatalf("mutation posts=%d, want 1", posts.Load())
	}
}

func TestMuxDialRetriesResetBeforeRouteRegistered(t *testing.T) {
	var upgrades atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if upgrades.Add(1) == 1 {
			resetRequest(t, w)
			return
		}
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		for {
			if _, _, err := c.Read(r.Context()); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	if err := mux.Dial(context.Background()); err != nil {
		t.Fatalf("dial should retry the reset upgrade: %v", err)
	}
	defer mux.Close()
	if upgrades.Load() != 2 {
		t.Fatalf("upgrades=%d, want 2", upgrades.Load())
	}
}
