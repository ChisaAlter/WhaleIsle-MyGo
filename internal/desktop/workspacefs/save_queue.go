package workspacefs

import (
	"context"
	"errors"
	"sync"
	"time"
)

const AutoSaveDelay = 500 * time.Millisecond

// SaveSnapshot contains only immutable values. UI-owned draft pointers are
// never written from a timer or completion goroutine.
type SaveSnapshot struct {
	Confirmed, Error          string
	Revision                  uint64
	Pending, Running, Blocked bool
}

type SaveQueue struct {
	mu                                     sync.Mutex
	ctx                                    context.Context
	cancel                                 context.CancelFunc
	delay                                  time.Duration
	persist                                func(context.Context, string, string) error
	notify                                 func()
	timer                                  *time.Timer
	timerVersion                           uint64
	draft, confirmed, problem              string
	revision                               uint64
	pending, due, running, blocked, closed bool
}

func NewSaveQueue(baseline string, persist func(context.Context, string, string) error, notify func()) *SaveQueue {
	return newSaveQueue(baseline, AutoSaveDelay, persist, notify)
}
func newSaveQueue(baseline string, delay time.Duration, persist func(context.Context, string, string) error, notify func()) *SaveQueue {
	ctx, cancel := context.WithCancel(context.Background())
	return &SaveQueue{ctx: ctx, cancel: cancel, delay: delay, persist: persist, notify: notify, draft: baseline, confirmed: baseline}
}

// Observe schedules only actual edits. Initial/recovered drafts can be observed
// with automatic=false and remain preserved until edited or explicitly saved.
func (q *SaveQueue) Observe(text string, automatic bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	changed := q.draft != text
	q.draft = text
	if !automatic {
		q.stopPendingLocked()
		return
	}
	if !changed || q.blocked {
		return
	}
	q.stopPendingLocked()
	if text == q.confirmed && !q.running {
		return
	}
	q.pending = true
	version := q.timerVersion
	q.timer = time.AfterFunc(q.delay, func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		if q.closed || q.timerVersion != version {
			return
		}
		q.timer = nil
		q.due = true
		q.startLocked()
	})
}

// Flush bypasses debounce but shares the same serial queue as automatic saves.
// An in-flight write always records its exact snapshot, never newer keystrokes.
func (q *SaveQueue) Flush(text string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.stopPendingLocked()
	q.draft = text
	q.blocked = false
	q.problem = ""
	if text == q.confirmed && !q.running {
		return
	}
	q.pending = true
	q.due = true
	q.startLocked()
}
func (q *SaveQueue) stopPendingLocked() {
	if q.timer != nil {
		q.timer.Stop()
		q.timer = nil
	}
	q.timerVersion++
	q.pending = false
	q.due = false
}
func (q *SaveQueue) startLocked() {
	if q.closed || q.running || q.blocked || !q.pending || !q.due {
		return
	}
	if q.draft == q.confirmed {
		q.pending = false
		q.due = false
		return
	}
	base, text := q.confirmed, q.draft
	q.pending = false
	q.due = false
	q.running = true
	go func() {
		err := q.persist(q.ctx, base, text)
		q.mu.Lock()
		if q.closed {
			q.running = false
			q.mu.Unlock()
			return
		}
		q.running = false
		if err != nil {
			q.problem = err.Error()
			q.blocked = true
			q.stopPendingLocked()
		} else {
			q.confirmed = text
			q.revision++
			q.problem = ""
			q.startLocked()
		}
		notify := q.notify
		q.mu.Unlock()
		if notify != nil {
			notify()
		}
	}()
}
func (q *SaveQueue) Snapshot() SaveSnapshot {
	q.mu.Lock()
	defer q.mu.Unlock()
	return SaveSnapshot{Confirmed: q.confirmed, Error: q.problem, Revision: q.revision, Pending: q.pending, Running: q.running, Blocked: q.blocked}
}

// Dispose cancels timers and pre-write work. It never flushes on close, discard,
// file/session/Host switches, and suppresses all late completion publication.
func (q *SaveQueue) Dispose() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	q.stopPendingLocked()
	q.cancel()
}

// SaveTextChecked preserves the exact read baseline and refuses unreadable,
// binary, truncated, externally changed or out-of-workspace targets. A remote
// preview is never authority to create or overwrite a local mirror file.
// One gate also serializes a newly bound queue with an older write that had
// already entered the OS before cancellation. Waiting remains cancellable.
var checkedSaveGate = make(chan struct{}, 1)

func SaveTextChecked(ctx context.Context, root, rel, baseline, text string, write func(string, string) error) error {
	select {
	case checkedSaveGate <- struct{}{}:
		defer func() { <-checkedSaveGate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	full, err := JoinRoot(root, rel)
	if err != nil {
		return err
	}
	current, err := ReadFileText(full, TextPreviewLimit)
	if err != nil {
		return err
	}
	if current.Text == text {
		return nil
	}
	if current.Text != baseline {
		return errors.New("File changed on disk. Draft kept. Copy your draft before reopening to reconcile.")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Revalidate immediately before the write, not just at preview/selection time.
	if _, err = JoinRoot(root, rel); err != nil {
		return err
	}
	if write == nil {
		return errors.New("no file writer")
	}
	return write(rel, text)
}

// Pause drops only queued/debounced work. A write already in progress may
// finish against its captured file; its result stays with that file's queue.
func (q *SaveQueue) Pause() { q.mu.Lock(); defer q.mu.Unlock(); q.stopPendingLocked() }
