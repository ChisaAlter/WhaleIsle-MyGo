package files

import (
	"context"
	"sync"
)

// The UI supplies a fresh tab identity, not merely a reusable path. No UI
// pointer is dereferenced by the worker, including on cancellation/completion.
type textReadOwner struct {
	binding, root, session, path string
	tab                          uint64
}

type textReadResult struct {
	text, stat, fullPath string
	err                  error
	truncated, remote    bool
}

type textReadJob struct {
	owner      textReadOwner
	generation uint64
	ctx        context.Context
	read       func(context.Context) textReadResult
}

// One worker plus one replaceable latest job bounds even reads that ignore
// context while blocked in the OS. Cancellation retires authority immediately;
// it cannot promise to interrupt a synchronous filesystem syscall.
type textReadQueue struct {
	mu         sync.Mutex
	generation uint64
	owner      textReadOwner
	worker     bool
	closed     bool
	pending    *textReadJob
	cancel     context.CancelFunc
	result     *textReadResult
	notify     func()
}

func (q *textReadQueue) setNotify(notify func()) {
	q.mu.Lock()
	q.notify = notify
	q.mu.Unlock()
}

func (q *textReadQueue) request(owner textReadOwner, read func(context.Context) textReadResult) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.retireLocked()
	q.owner = owner
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel
	q.pending = &textReadJob{owner: owner, generation: q.generation, ctx: ctx, read: read}
	if !q.worker {
		q.worker = true
		go q.run()
	}
}

func (q *textReadQueue) run() {
	for {
		q.mu.Lock()
		job := q.pending
		q.pending = nil
		if job == nil {
			q.worker = false
			q.mu.Unlock()
			return
		}
		q.mu.Unlock()
		var result textReadResult
		if job.ctx.Err() == nil {
			result = job.read(job.ctx)
		}
		q.mu.Lock()
		var notify func()
		if !q.closed && q.generation == job.generation && q.owner == job.owner && job.ctx.Err() == nil {
			q.result = &result
			notify = q.notify
		}
		q.mu.Unlock()
		if notify != nil {
			notify() // invalidation only; the UI takes and applies its own result
		}
	}
}

func (q *textReadQueue) take(owner textReadOwner) (textReadResult, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || q.owner != owner || q.result == nil {
		return textReadResult{}, false
	}
	result := *q.result
	q.result = nil
	return result, true
}

func (q *textReadQueue) retireLocked() {
	if q.cancel != nil {
		q.cancel()
		q.cancel = nil
	}
	q.generation++
	q.pending = nil
	q.result = nil
}

func (q *textReadQueue) stop(closeQueue bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.retireLocked()
	q.closed = q.closed || closeQueue
}
