package dsh

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestBootstrapFailureRetainsExplicitRetryOwner(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mux := &Mux{}
		loads := make(chan int, 2)
		failed := make(chan error, 1)
		done := make(chan error, 1)
		attempt := 0
		go func() {
			done <- maintainMux(ctx, mux, Backoff{}, RecoveryConfig{}, true, func(context.Context) error { t.Error("invalid bootstrap dialed"); return nil }, nil, nil, func(ctx context.Context) (RecoveryConfig, error) {
				attempt++
				loads <- attempt
				if attempt == 1 {
					return RecoveryConfig{}, nil
				} // invalid returned policy must not become defaults
				<-ctx.Done()
				return RecoveryConfig{}, ctx.Err()
			}, func(err error) {
				if err != nil {
					failed <- err
				}
			})
		}()
		<-loads
		<-failed
		synctest.Wait()
		if attempt != 1 {
			t.Fatalf("bootstrap silently retried %d times", attempt)
		}
		if !mux.RequestReconnect() {
			t.Fatal("bootstrap failure lost its Retry owner")
		}
		if n := <-loads; n != 2 {
			t.Fatal(n)
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
		if mux.RequestReconnect() {
			t.Fatal("retired owner accepted Retry")
		}
	})
}

func testReadinessOwner(ctx context.Context, config RecoveryConfig) (*Mux, *muxMaintainer, *muxAttempt) {
	child, cancel := context.WithCancelCause(ctx)
	a := &muxAttempt{ctx: child, cancel: cancel, ready: make(chan struct{}), watchDone: make(chan struct{}), started: time.Now()}
	a.deadline = a.started.Add(time.Duration(config.GenerationReadyTimeoutMS) * time.Millisecond)
	o := &muxMaintainer{ctx: ctx, attempt: a, requests: make(chan struct{}, 1), networkChanged: make(chan struct{}, 1)}
	return &Mux{maintainer: o}, o, a
}

func TestGenerationReadyDeadlineIncludesTimeBeforeWatcher(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		config := RecoveryConfig{500, 2, 10000, 30, 100}
		mux, o, a := testReadinessOwner(context.Background(), config)
		defer a.cancel(context.Canceled)
		warnings := make(chan error, 2)
		time.Sleep(60 * time.Millisecond) // Dial already consumed part of the generation budget.
		go mux.watchReadiness(o, a, config, func(err error) { warnings <- err })
		synctest.Wait()
		if len(warnings) != 1 {
			t.Fatalf("overdue warning missing: %d", len(warnings))
		}
		time.Sleep(40 * time.Millisecond)
		synctest.Wait()
		if a.ctx.Err() == nil || len(warnings) != 2 {
			t.Fatalf("deadline restarted after Dial: cause=%v warnings=%d", context.Cause(a.ctx), len(warnings))
		}
		if mux.MarkReady(a.ctx) {
			t.Fatal("late Ready revived expired generation")
		}
		<-a.watchDone
	})
}

func TestGenerationReadyAndRetirementRejectStaleAuthority(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mux, o, old := testReadinessOwner(context.Background(), DefaultRecoveryConfig())
		generationDone := make(chan struct{})
		if !mux.BindGeneration(old.ctx, generationDone) || mux.BindGeneration(old.ctx, generationDone) {
			t.Fatal("generation registration was not single owner")
		}
		if mux.MarkReady(context.Background()) || !mux.MarkReady(old.ctx) || mux.MarkReady(old.ctx) {
			t.Fatal("Ready did not require the current first events acknowledgement")
		}
		old.cancel(context.Canceled)
		_, _, next := testReadinessOwner(context.Background(), DefaultRecoveryConfig())
		defer next.cancel(context.Canceled)
		o.attempt = next
		if mux.MarkReady(old.ctx) || mux.GenerationLost(old.ctx, errors.New("old events")) || mux.BindGeneration(old.ctx, make(chan struct{})) {
			t.Fatal("retired events changed the replacement generation")
		}
		if next.ctx.Err() != nil || !mux.MarkReady(next.ctx) {
			t.Fatal("replacement events lost authority")
		}
		close(generationDone)
	})
}

func TestNetworkPauseManualOnceAndOnlineResetsRetryCap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		o := &muxMaintainer{ctx: ctx, requests: make(chan struct{}, 1), networkChanged: make(chan struct{}, 1)}
		mux := &Mux{maintainer: o}
		mux.SetNetworkAvailability(true, false)
		manual := make(chan bool, 1)
		go func() { m, _ := mux.waitForNetwork(ctx, o); manual <- m }()
		synctest.Wait()
		select {
		case <-manual:
			t.Fatal("offline automatically resumed")
		default:
		}
		if !mux.RequestReconnect() {
			t.Fatal("manual offline retry rejected")
		}
		if !<-manual {
			t.Fatal("manual request lost its one-attempt override")
		}
		go func() { m, _ := mux.waitForNetwork(ctx, o); manual <- m }()
		synctest.Wait()
		select {
		case <-manual:
			t.Fatal("manual override leaked into another offline retry")
		default:
		}
		mux.SetNetworkAvailability(true, true)
		if <-manual {
			t.Fatal("OS online transition became a manual request")
		}
		result := make(chan time.Duration, 1)
		go func() {
			_, cap, err := mux.waitNetworkRetry(ctx, o, 10*time.Second, 100*time.Millisecond)
			if err != nil {
				t.Error(err)
			}
			result <- cap
		}()
		synctest.Wait()
		// A newer network transition supersedes the old delay and manual state.
		mux.SetNetworkAvailability(true, false)
		mux.SetNetworkAvailability(true, true)
		synctest.Wait()
		time.Sleep(100 * time.Millisecond)
		if cap := <-result; cap != 100*time.Millisecond {
			t.Fatalf("online retained failed cap %v", cap)
		}
	})
}

func TestRecoveryBackoffCapsAndJitter(t *testing.T) {
	b := DefaultRecoveryConfig().backoff()
	for cur, want := range map[time.Duration]time.Duration{500 * time.Millisecond: time.Second, 8 * time.Second: 10 * time.Second, 10 * time.Second: 10 * time.Second} {
		if got := nextDelay(cur, b); got != want {
			t.Fatalf("%v => %v want %v", cur, got, want)
		}
	}
	if got := nextDelay(time.Second, Backoff{Initial: time.Millisecond, Max: 10 * time.Second, Factor: math.MaxFloat64}); got != 10*time.Second {
		t.Fatal("factor overflow escaped cap", got)
	}
	for range 100 {
		d := jitterDelay(time.Second)
		if d < 500*time.Millisecond || d >= time.Second {
			t.Fatal("jitter outside 50–100% interval", d)
		}
	}
}

func TestLateCarrierFailureCannotRetireReplacement(t *testing.T) {
	old, current := &websocket.Conn{}, &websocket.Conn{}
	oldDone, currentDone := make(chan struct{}), make(chan struct{})
	stream := &logicalStream{inbox: make(chan ServerFrame, 1)}
	calls := 0
	mux := &Mux{conn: current, done: currentDone, streams: map[string]*logicalStream{"current": stream}, OnCarrierLost: func(error) { calls++ }}
	mux.carrierLost(old, oldDone, errors.New("retired socket EOF"))
	mux.carrierLost(current, oldDone, errors.New("retired generation"))
	if mux.conn != current || mux.done != currentDone || mux.streams["current"] != stream || mux.readErr != nil || calls != 0 {
		t.Fatal("old socket/generation mutated replacement authority")
	}
	failure := errors.New("current EOF")
	mux.carrierLost(current, currentDone, failure)
	mux.carrierLost(current, currentDone, failure)
	if mux.conn != nil || mux.readErr != failure || len(mux.streams) != 0 || calls != 1 {
		t.Fatal("current failure did not retire and notify exactly once")
	}
}

func TestCancelledGenerationWatcherJoinsWithoutReadyRegistration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		mux, owner, attempt := testReadinessOwner(context.Background(), DefaultRecoveryConfig())
		attempt.cancel(context.Canceled)
		if mux.BindGeneration(attempt.ctx, make(chan struct{})) {
			t.Fatal("cancelled registration admitted an events pump")
		}
		go mux.watchReadiness(owner, attempt, DefaultRecoveryConfig(), func(error) { t.Error("retired generation warned") })
		synctest.Wait()
		select {
		case <-attempt.watchDone:
		default:
			t.Fatal("cancelled pre-Ready watcher could not join")
		}
	})
}

func TestManualReconnectJoinsOldGenerationBeforeRedial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var carriers atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		carriers.Add(1)
		for {
			if _, _, err = conn.Read(r.Context()); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	ready := make(chan context.Context, 2)
	retire := make(chan struct{})
	done := make(chan error, 1)
	var release sync.Once
	defer func() { release.Do(func() { close(retire) }); cancel() }()
	var n atomic.Int32
	go func() {
		done <- MaintainMuxWithRecovery(ctx, mux, DefaultRecoveryConfig(), func(g context.Context) error {
			if n.Add(1) == 1 {
				if !mux.BindGeneration(g, retire) {
					return errors.New("bind failed")
				}
			}
			ready <- g
			return nil
		}, nil, nil)
	}()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("first Dial timed out")
	}
	if !mux.RequestReconnect() {
		t.Fatal("manual reconnect rejected")
	}
	// Cancellation starts retirement, but a delivery owner still owns cleanup.
	deadline := time.NewTimer(30 * time.Millisecond)
	select {
	case <-ready:
		t.Fatal("redial preceded delivery retirement")
	case <-deadline.C:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if carriers.Load() != 1 {
		t.Fatal("new carrier opened before join")
	}
	release.Do(func() { close(retire) })
	select {
	case g := <-ready:
		if !mux.MarkReady(g) {
			t.Fatal("new generation Ready rejected")
		}
	case <-ctx.Done():
		t.Fatal("redial did not resume after join")
	}
	cancel()
	<-done
}

type recoveryTestTransport func(*http.Request) (*http.Response, error)

func (f recoveryTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestMaintainMuxOfflineAllowsOnlyOneManualDial(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var attempts atomic.Int32
		mux := &Mux{Endpoint: Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: recoveryTestTransport(func(*http.Request) (*http.Response, error) {
			attempts.Add(1)
			return nil, errors.New("synthetic Dial failure")
		})}}
		mux.SetNetworkAvailability(true, false)
		done := make(chan error, 1)
		go func() { done <- MaintainMuxWithRecovery(ctx, mux, DefaultRecoveryConfig(), nil, nil, nil) }()
		synctest.Wait()
		time.Sleep(time.Minute)
		synctest.Wait()
		if attempts.Load() != 0 {
			t.Fatal("automatic Dial occurred offline")
		}
		if !mux.RequestReconnect() {
			t.Fatal("offline Retry owner missing")
		}
		synctest.Wait()
		if attempts.Load() != 1 {
			t.Fatal("manual request did not make exactly one attempt", attempts.Load())
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		if attempts.Load() != 1 {
			t.Fatal("manual failure started automatic backoff while offline", attempts.Load())
		}
		mux.SetNetworkAvailability(true, true)
		synctest.Wait()
		time.Sleep(500 * time.Millisecond)
		synctest.Wait()
		if attempts.Load() < 2 {
			t.Fatal("online event did not resume automatic recovery")
		}
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	})
}

func TestRetiredStreamCannotWriteReplacementCarrier(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	frames := make(chan ClientFrame, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer conn.CloseNow()
		for {
			var frame ClientFrame
			if wsjson.Read(r.Context(), conn, &frame) != nil {
				return
			}
			frames <- frame
		}
	}))
	defer srv.Close()
	mux := &Mux{Endpoint: Endpoint{Origin: srv.URL}}
	defer mux.Close()
	if err := mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	old, err := mux.Open(ctx, "old", nil)
	if err != nil {
		t.Fatal(err)
	}
	<-frames
	if err = mux.closeCarrier(true); err != nil {
		t.Fatal(err)
	}
	mux.Reset()
	if err = mux.Dial(ctx); err != nil {
		t.Fatal(err)
	}
	current, err := mux.Open(ctx, "current", nil)
	if err != nil {
		t.Fatal(err)
	}
	<-frames
	if err = old.SendItem(ctx, "stale"); !errors.Is(err, ErrCarrierLost) {
		t.Fatal(err)
	}
	if err = old.End(ctx); !errors.Is(err, ErrCarrierLost) {
		t.Fatal(err)
	}
	old.Cancel()
	if err = current.SendItem(ctx, "current"); err != nil {
		t.Fatal(err)
	}
	select {
	case f := <-frames:
		if f.StreamID != current.id || f.Type != FrameItem {
			t.Fatalf("old stream reached new carrier: %+v", f)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err = old.Recv(ctx); !errors.Is(err, ErrCarrierLost) {
		t.Fatal(err)
	}
}
