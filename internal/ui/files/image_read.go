package files

import (
	"context"
	"errors"
	"sync"
)

// Owner pointers are identity tokens only. Workers never dereference UI state.
type imageReadOwner struct {
	tab                          *FileTab
	binding, session, root, path string
	epoch                        uint64
}

type imageReadJob struct {
	owner      imageReadOwner
	generation uint64
	ctx        context.Context
	read       func(context.Context) *imagePreview
}

// One worker plus one replaceable latest job bounds even decodes that ignore
// context while blocked in the OS. Cancellation retires authority immediately;
// it cannot promise to interrupt a synchronous filesystem syscall or bounded native decoder.
type imageReadQueue struct {
	mu         sync.Mutex
	generation uint64
	owner      imageReadOwner
	worker     bool
	closed     bool
	pending    *imageReadJob
	cancel     context.CancelFunc
	result     *imagePreview
	notify     func()
}

func (q *imageReadQueue) setNotify(notify func()) {
	q.mu.Lock()
	q.notify = notify
	q.mu.Unlock()
}

func (q *imageReadQueue) request(owner imageReadOwner, read func(context.Context) *imagePreview) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.retireLocked()
	q.owner = owner
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel
	q.pending = &imageReadJob{owner: owner, generation: q.generation, ctx: ctx, read: read}
	if !q.worker {
		q.worker = true
		go q.run()
	}
}

func (q *imageReadQueue) run() {
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
		var result *imagePreview
		if job.ctx.Err() == nil {
			result = job.read(job.ctx)
		}
		q.mu.Lock()
		var notify func()
		if !q.closed && q.generation == job.generation && q.owner == job.owner && job.ctx.Err() == nil {
			q.result = result
			notify = q.notify
		}
		q.mu.Unlock()
		if notify != nil {
			notify() // invalidation only; the UI takes and applies its own result
		}
	}
}

func (q *imageReadQueue) take(owner imageReadOwner) (*imagePreview, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || q.owner != owner || q.result == nil {
		return nil, false
	}
	result := q.result
	q.result = nil
	return result, true
}

func (q *imageReadQueue) retireLocked() {
	if q.cancel != nil {
		q.cancel()
		q.cancel = nil
	}
	q.generation++
	q.pending = nil
	q.result = nil
}

func (q *imageReadQueue) stop(closeQueue bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.retireLocked()
	q.closed = q.closed || closeQueue
}

// stopImageRead is called on the UI thread. Only unfinished placeholders are
// removed; completed pixels and independent per-tab view/draft state survive.
func (t *Tabs) stopImageRead(closeQueue bool) {
	if t == nil {
		return
	}
	if t.imageReadTab != nil && t.imageReadTab.image != nil && t.imageReadTab.image.loading {
		t.imageReadTab.image = nil
	}
	t.imageReadTab = nil
	if t.imageReader != nil {
		t.imageReader.stop(closeQueue)
	}
}

func imageOwner(m Model, tab *FileTab) imageReadOwner {
	return imageReadOwner{tab: tab, binding: m.BindingID, session: m.SessionID, root: m.WorkspaceRoot, path: tab.Path, epoch: m.ImageGeneration}
}

// Painting only starts/takes work. All file reads, validation, WASM compilation
// and decoding run on a single worker with one replaceable pending request.
func readImagePreview(m Model, tab *FileTab) *imagePreview {
	if m.Tabs == nil || tab == nil {
		return &imagePreview{err: errors.New("Image preview requires a resource tab")}
	}
	t := m.Tabs
	owner := imageOwner(m, tab)
	if tab.image != nil && tab.image.owner != owner {
		t.stopImageRead(false)
		tab.image = nil
	}
	if tab.image == nil {
		t.stopImageRead(false)
		retainImagePreview(m, tab, &imagePreview{loading: true, owner: owner})
		if t.imageReader == nil {
			t.imageReader = &imageReadQueue{}
		}
		t.imageReader.setNotify(m.notify)
		t.imageReadTab = tab
		root, path := owner.root, owner.path
		t.imageReader.request(owner, func(ctx context.Context) *imagePreview {
			return loadImagePreviewContext(ctx, root, path)
		})
	}
	if tab.image.loading && t.imageReader != nil {
		if result, ok := t.imageReader.take(owner); ok {
			result.owner = owner
			tab.image = result
			t.imageReadTab = nil
		}
	}
	return tab.image
}
