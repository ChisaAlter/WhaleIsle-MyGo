package main

import (
	"log"
	"sync"
	"time"

	"whaleisle/internal/config"
)

const asyncConfigDebounce = 120 * time.Millisecond

type asyncConfigOp struct {
	merge bool
	fn    func(*config.Config)
}

// asyncConfigWriter serializes high-frequency preference writes onto a single
// debounced background writer so UI clicks never block on NTFS fsync. Pending
// edits remain immediately visible to callers via Effective(store), and Flush
// drains all queued edits synchronously before exit or Host restart.
type asyncConfigWriter struct {
	mu       sync.Mutex
	cond     *sync.Cond
	store    *config.Store
	debounce time.Duration
	onError  func(error)

	pending  []asyncConfigOp
	inFlight []asyncConfigOp
	timer    *time.Timer
	writing  bool
	closed   bool
	lastErr  error
}

func newAsyncConfigWriter(store *config.Store, debounce time.Duration, onError func(error)) *asyncConfigWriter {
	if debounce <= 0 {
		debounce = asyncConfigDebounce
	}
	w := &asyncConfigWriter{
		store:    store,
		debounce: debounce,
		onError:  onError,
	}
	w.cond = sync.NewCond(&w.mu)
	return w
}

// EnqueueUpdate schedules fn to run inside store.Update off the UI thread.
func (w *asyncConfigWriter) EnqueueUpdate(fn func(*config.Config)) {
	w.enqueue(asyncConfigOp{merge: false, fn: fn})
}

// EnqueueMerge schedules fn to run inside store.Merge off the UI thread.
func (w *asyncConfigWriter) EnqueueMerge(fn func(*config.Config)) {
	w.enqueue(asyncConfigOp{merge: true, fn: fn})
}

func (w *asyncConfigWriter) enqueue(op asyncConfigOp) {
	_ = w.tryEnqueue(op)
}

func (w *asyncConfigWriter) tryEnqueue(op asyncConfigOp) bool {
	if w == nil || op.fn == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed || w.store == nil {
		return false
	}
	w.pending = append(w.pending, op)
	if w.timer != nil {
		w.timer.Reset(w.debounce)
		return true
	}
	w.timer = time.AfterFunc(w.debounce, func() {
		go w.drain()
	})
	return true
}

// Effective returns store.Get() overlaid with any in-flight and queued
// optimistic mutations so UI frames observe the latest preference immediately.
func (w *asyncConfigWriter) Effective(store *config.Store) config.Config {
	if store == nil {
		return config.Defaults()
	}
	cfg := store.Get()
	if w == nil {
		return cfg
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.store != store {
		return cfg
	}
	for _, op := range w.inFlight {
		op.fn(&cfg)
	}
	for _, op := range w.pending {
		op.fn(&cfg)
	}
	return cfg
}

func (w *asyncConfigWriter) drain() {
	w.mu.Lock()
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	if w.writing {
		w.mu.Unlock()
		return
	}
	w.writing = true
	w.drainLocked()
	w.writing = false
	w.cond.Broadcast()
	w.mu.Unlock()
}

// drainLocked writes queued batches sequentially while releasing w.mu across
// disk I/O so new UI clicks never stall waiting for an fsync in progress.
// REQUIRES: w.mu held and w.writing == true.
func (w *asyncConfigWriter) drainLocked() {
	for len(w.pending) > 0 {
		if w.timer != nil {
			w.timer.Stop()
			w.timer = nil
		}
		batch := w.pending
		w.pending = nil
		w.inFlight = batch
		store := w.store
		onErr := w.onError
		w.mu.Unlock()

		var batchErr error
		if store != nil {
			// Partition contiguous Update vs Merge ops while preserving order.
			i := 0
			for i < len(batch) {
				merge := batch[i].merge
				j := i + 1
				for j < len(batch) && batch[j].merge == merge {
					j++
				}
				slice := batch[i:j]
				apply := func(c *config.Config) {
					for _, op := range slice {
						op.fn(c)
					}
				}
				var err error
				if merge {
					err = store.Merge(apply)
				} else {
					err = store.Update(apply)
				}
				if err != nil {
					batchErr = err
					if onErr != nil {
						onErr(err)
					} else {
						log.Printf("async config save: %v", err)
					}
				}
				i = j
			}
		}

		w.mu.Lock()
		w.inFlight = nil
		w.lastErr = batchErr
	}
}

// Flush synchronously persists all pending mutations and waits for any active
// background write to finish. Safe to call before quit, relaunch, or restart.
func (w *asyncConfigWriter) Flush() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flushLocked()
}

// flushLocked drains all pending mutations while w.mu is held.
func (w *asyncConfigWriter) flushLocked() error {
	if w.timer != nil {
		w.timer.Stop()
		w.timer = nil
	}
	for w.writing {
		w.cond.Wait()
	}
	if len(w.pending) > 0 {
		w.writing = true
		w.drainLocked()
		w.writing = false
		w.cond.Broadcast()
	}
	return w.lastErr
}

// Close rejects further enqueues, flushes all accepted pending writes, and
// prevents further background timers.
func (w *asyncConfigWriter) Close() error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return w.flushLocked()
}

func (s *mainState) configWriterFor(store *config.Store) *asyncConfigWriter {
	if store == nil {
		return nil
	}
	s.configWriterMu.Lock()
	var replaced *asyncConfigWriter
	if s.configWriter == nil || s.configWriter.store != store {
		replaced = s.configWriter
		s.configWriter = newAsyncConfigWriter(store, asyncConfigDebounce, func(err error) {
			log.Printf("config: async save failed: %v", err)
		})
	}
	w := s.configWriter
	s.configWriterMu.Unlock()
	if replaced != nil {
		if err := replaced.Close(); err != nil {
			log.Printf("config: flush replaced writer failed: %v", err)
		}
	}
	return w
}

func (s *mainState) effectiveConfig() config.Config {
	s.mu.Lock()
	store := s.cfgStore
	s.mu.Unlock()
	if store == nil {
		return config.Defaults()
	}
	return s.effectiveConfigFor(store)
}

func (s *mainState) effectiveConfigFor(store *config.Store) config.Config {
	if store == nil {
		return config.Defaults()
	}
	s.configWriterMu.Lock()
	w := s.configWriter
	s.configWriterMu.Unlock()
	if w != nil && w.store == store {
		return w.Effective(store)
	}
	return store.Get()
}

func (s *mainState) enqueueConfigUpdate(store *config.Store, fn func(*config.Config)) {
	if store == nil || fn == nil {
		return
	}
	s.configWriterFor(store).EnqueueUpdate(fn)
}

func (s *mainState) enqueueConfigMerge(store *config.Store, fn func(*config.Config)) {
	if store == nil || fn == nil {
		return
	}
	s.configWriterFor(store).EnqueueMerge(fn)
}

// flushExistingConfigWriter flushes pending debounced edits for store only if
// an async writer already exists for it, preserving CAS conflict behavior when
// no async writes were queued.
func (s *mainState) flushExistingConfigWriter(store *config.Store) error {
	if store == nil {
		return nil
	}
	s.configWriterMu.Lock()
	w := s.configWriter
	s.configWriterMu.Unlock()
	if w != nil && w.store == store {
		return w.Flush()
	}
	return nil
}

// flushPendingConfigWrites drains all debounced async persistence layers
// (session workspace preferences, workspace view preferences, general async
// config writer, and config.Store async writes) in dependency order before
// quit, relaunch, data-location switch, or Host restart.
func (s *mainState) flushPendingConfigWrites() error {
	s.configWriterMu.Lock()
	w := s.configWriter
	s.configWriterMu.Unlock()
	s.mu.Lock()
	sessions := s.sessions
	vp := s.viewPreferences
	cfgStore := s.cfgStore
	s.mu.Unlock()
	var firstErr error
	recordErr := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if sessions != nil {
		recordErr(sessions.FlushWorkspacePreferences())
	}
	if vp != nil {
		recordErr(vp.Flush())
	}
	if w != nil {
		recordErr(w.Flush())
	}
	if cfgStore != nil {
		recordErr(cfgStore.Flush())
	}
	return firstErr
}
