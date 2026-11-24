package chat

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/store/session"
)

type fakeOpenUISession struct {
	mu         sync.Mutex
	done       chan struct{}
	events     chan json.RawMessage
	closeOnce  sync.Once
	updates    []openui.Render
	themeCalls []string
	closed     atomic.Bool
}

func newFakeOpenUISession() *fakeOpenUISession {
	return &fakeOpenUISession{
		done:   make(chan struct{}),
		events: make(chan json.RawMessage),
	}
}

func (f *fakeOpenUISession) UpdateOpenUI(r openui.Render) error {
	f.mu.Lock()
	f.updates = append(f.updates, r)
	f.mu.Unlock()
	return nil
}

func (f *fakeOpenUISession) SetTheme(appearance string, variables map[string]string) error {
	f.mu.Lock()
	f.themeCalls = append(f.themeCalls, appearance)
	f.mu.Unlock()
	return nil
}

func (f *fakeOpenUISession) Events() <-chan json.RawMessage { return f.events }

func (f *fakeOpenUISession) ReplyOpenUIAPI(response openui.APIResponse) error { return nil }

func (f *fakeOpenUISession) Done() <-chan struct{} { return f.done }

func (f *fakeOpenUISession) Close() error {
	f.closeOnce.Do(func() {
		f.closed.Store(true)
		close(f.done)
	})
	return nil
}

func waitChatCond(t *testing.T, timeout time.Duration, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestOpenUILoopIdleZeroHzAndCoalescedEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := newFakeOpenUISession()
	b := &openUIBlock{
		program:    "initial",
		appearance: "light",
		variables:  map[string]string{"--bg": "#ffffff"},
		streaming:  false,
		wake:       make(chan struct{}, 1),
		handle:     sess,
		revision:   1,
	}

	var validCalls atomic.Int32
	var invalidations atomic.Int32
	loopDone := make(chan struct{})
	finish := func(err error) {
		b.mu.Lock()
		b.loading = false
		b.handle = nil
		b.mu.Unlock()
	}

	go func() {
		defer close(loopDone)
		runOpenUILoop(
			ctx, b, sess, "inst-1", "", t.TempDir(), "store-1",
			"initial", "light", map[string]string{"--bg": "#ffffff"},
			false, false,
			func() bool { validCalls.Add(1); return true },
			func() { invalidations.Add(1) },
			finish,
		)
	}()

	// Let startup sync complete, then observe complete 0 Hz silence for >300ms.
	waitChatCond(t, time.Second, func() bool { return validCalls.Load() >= 1 }, "loop did not complete startup sync")
	callsAtIdleStart := validCalls.Load()
	time.Sleep(320 * time.Millisecond)

	if got := validCalls.Load(); got != callsAtIdleStart {
		t.Fatalf("idle OpenUI loop woke %d times while visible and settled", got-callsAtIdleStart)
	}
	sess.mu.Lock()
	if len(sess.themeCalls) != 0 || len(sess.updates) != 0 {
		sess.mu.Unlock()
		t.Fatalf("idle loop performed redundant theme/update IPC: themes=%v updates=%d", sess.themeCalls, len(sess.updates))
	}
	sess.mu.Unlock()

	// Streaming pushes coalesce rapid updates and flush final program cleanly.
	b.mu.Lock()
	b.program = "stream-1"
	b.streaming = true
	b.revision++
	b.mu.Unlock()
	b.notifyWake()

	waitChatCond(t, time.Second, func() bool {
		sess.mu.Lock()
		defer sess.mu.Unlock()
		return len(sess.updates) == 1 && sess.updates[0].Program == "stream-1"
	}, "first streaming push did not happen promptly")

	b.mu.Lock()
	b.program = "stream-2"
	b.revision++
	b.mu.Unlock()
	b.notifyWake()
	time.Sleep(5 * time.Millisecond)
	b.mu.Lock()
	b.program = "stream-3"
	b.streaming = false
	b.revision++
	b.mu.Unlock()
	b.notifyWake()

	waitChatCond(t, time.Second, func() bool {
		sess.mu.Lock()
		defer sess.mu.Unlock()
		return len(sess.updates) >= 2 && sess.updates[len(sess.updates)-1].Program == "stream-3"
	}, "final non-streaming chunk did not flush")

	// Theme update pushes exactly once when changed.
	b.mu.Lock()
	b.appearance = "dark"
	b.mu.Unlock()
	b.notifyWake()
	waitChatCond(t, time.Second, func() bool {
		sess.mu.Lock()
		defer sess.mu.Unlock()
		return len(sess.themeCalls) == 1 && sess.themeCalls[0] == "dark"
	}, "theme change did not wake SetTheme")

	cancel()
	select {
	case <-loopDone:
	case <-time.After(time.Second):
		t.Fatal("runOpenUILoop did not exit on context cancel")
	}
}

func TestOpenUIBlockViewDoesNotScheduleContinuousRebuilds(t *testing.T) {
	store := session.New()
	var builds atomic.Int32
	msg := session.Message{ID: "m1", Role: "assistant"}

	tt := ui.NewTester(func(c *ui.Context) {
		builds.Add(1)
		_ = openUIBlockView(c, Model{
			FeedbackStore: store,
			Snapshot:      session.ViewState{SelectedID: "s", PlanHistoryBinding: "b"},
		}, msg, "openui", `main=Text("ok")`, 0, "m1", false, nil)
	}, 360, 240)
	_ = tt.Image()

	initial := builds.Load()
	time.Sleep(220 * time.Millisecond)
	if builds.Load() != initial {
		t.Fatalf("openUIBlockView rebuilt %d extra times while idle", builds.Load()-initial)
	}
}
