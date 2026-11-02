package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/recovery"
)

const resumeCaptureInterval = 500 * time.Millisecond
const resumeCheckpointDelay = 600 * time.Millisecond
const resumeCheckpointMaxWait = 5 * time.Second

type resumeCheckpointTimer interface{ Stop() bool }

type resumeCheckpointRecord struct {
	revision uint64
	queuedAt time.Time
	state    recovery.State // detached on the native UI owner before enqueue
}

// One resume file owns one serialized disk worker. No worker reads a composer,
// file tab, pane tree, mainState or frame Context. Question drafts use the same
// recovery.Store mutex but retain their independent field ownership.
type resumeCheckpoint struct {
	mu            sync.Mutex
	owner         *recovery.Store
	invalidate    func()
	captureAt     time.Time
	changedAt     time.Time
	captureTimer  resumeCheckpointTimer
	writeTimer    resumeCheckpointTimer
	after         func(time.Duration, func()) resumeCheckpointTimer
	now           func() time.Time
	writeEpoch    uint64
	running       chan struct{}
	pending       *resumeCheckpointRecord
	revision      uint64
	digest        [32]byte
	hasDigest     bool
	forceCapture  bool
	paused        int
	closed        bool
	err           error
	errorSerial   uint64
	noticedSerial uint64
}

func (q *resumeCheckpoint) nowTime() time.Time {
	if q.now != nil {
		return q.now()
	}
	return time.Now()
}

func (q *resumeCheckpoint) afterFunc(delay time.Duration, fn func()) resumeCheckpointTimer {
	if q.after != nil {
		return q.after(delay, fn)
	}
	return time.AfterFunc(delay, fn)
}

func (q *resumeCheckpoint) due(now time.Time, invalidate func()) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.invalidate = invalidate
	if q.closed {
		return false
	}
	if !q.forceCapture && !q.captureAt.IsZero() && now.Sub(q.captureAt) < resumeCaptureInterval {
		if q.captureTimer == nil {
			q.captureTimer = q.afterFunc(resumeCaptureInterval-now.Sub(q.captureAt), func() {
				q.mu.Lock()
				q.captureTimer = nil
				notify, active := q.invalidate, !q.closed
				q.mu.Unlock()
				if active && notify != nil {
					notify()
				}
			})
		}
		return false
	}
	q.forceCapture, q.captureAt = false, now
	if q.captureTimer != nil {
		q.captureTimer.Stop()
		q.captureTimer = nil
	}
	return true
}

func (q *resumeCheckpoint) stage(next recovery.State, captureErr error, now time.Time) {
	var data []byte
	if captureErr == nil {
		data, captureErr = json.Marshal(next)
		if len(data) > recovery.MaxBytes {
			captureErr = fmt.Errorf(i18n.T(i18n.KeyCmdResumeSnapshotLimit), recovery.MaxBytes>>20)
		}
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	if captureErr != nil {
		// An older valid payload is not the current UI state. Preserve the UI
		// and original file, and require an explicit fresh capture on Retry.
		q.pending = nil
		q.hasDigest = false
		q.stopWriteLocked()
		q.setErrorLocked(captureErr)
		return
	}
	digest := sha256.Sum256(data)
	if !q.hasDigest || digest != q.digest {
		queuedAt := now
		if q.pending != nil {
			queuedAt = q.pending.queuedAt
		}
		q.revision++
		q.digest, q.hasDigest, q.changedAt = digest, true, now
		q.pending = &resumeCheckpointRecord{revision: q.revision, queuedAt: queuedAt, state: next}
		q.stopWriteLocked()
	}
	q.armLocked(now)
}

func (q *resumeCheckpoint) stopWriteLocked() {
	q.writeEpoch++
	if q.writeTimer != nil {
		q.writeTimer.Stop()
		q.writeTimer = nil
	}
}

func (q *resumeCheckpoint) setErrorLocked(err error) {
	if q.err == nil || q.err.Error() != err.Error() {
		q.errorSerial++
	}
	q.err = err
}

func (q *resumeCheckpoint) armLocked(now time.Time) {
	if q.closed || q.paused > 0 || q.err != nil || q.running != nil || q.pending == nil || q.writeTimer != nil {
		return
	}
	delay := max(time.Duration(0), resumeCheckpointDelay-now.Sub(q.changedAt))
	delay = min(delay, max(time.Duration(0), resumeCheckpointMaxWait-now.Sub(q.pending.queuedAt)))
	epoch := q.writeEpoch
	q.writeTimer = q.afterFunc(delay, func() { q.write(epoch) })
}

func (q *resumeCheckpoint) write(epoch uint64) {
	q.mu.Lock()
	if epoch != q.writeEpoch {
		q.mu.Unlock()
		return // A timer already dispatched before its newer capture canceled it.
	}
	q.writeTimer = nil
	if q.closed || q.paused > 0 || q.err != nil || q.running != nil || q.pending == nil {
		q.mu.Unlock()
		return
	}
	record := q.pending
	q.pending = nil
	done := make(chan struct{})
	q.running = done
	q.mu.Unlock()
	err := q.owner.Update(func(state *recovery.State) { applyResumeOwned(state, record.state) })
	q.mu.Lock()
	q.running = nil
	if err != nil {
		if q.pending == nil {
			q.pending = record
		}
		q.setErrorLocked(err)
	}
	q.armLocked(q.nowTime())
	notify, active := q.invalidate, !q.closed
	close(done)
	q.mu.Unlock()
	if active && notify != nil {
		notify()
	}
}

// Pause is a quit barrier, not a UI capture. Wait with no mainState lock held;
// the final save captures the latest UI and cannot race an older disk writer.
func (q *resumeCheckpoint) pause() {
	q.mu.Lock()
	q.paused++
	q.stopWriteLocked()
	done := q.running
	q.mu.Unlock()
	if done != nil {
		<-done
	}
}

func (q *resumeCheckpoint) resume() {
	q.mu.Lock()
	q.paused--
	q.armLocked(q.nowTime())
	q.mu.Unlock()
}

func (q *resumeCheckpoint) acceptFinal(next recovery.State) {
	data, err := json.Marshal(next)
	if err != nil {
		return // The successfully written candidate was already JSON-checked.
	}
	q.mu.Lock()
	q.revision++
	q.digest, q.hasDigest = sha256.Sum256(data), true
	q.pending, q.err = nil, nil
	q.stopWriteLocked()
	q.mu.Unlock()
}

func (q *resumeCheckpoint) retry() {
	q.mu.Lock()
	q.err, q.forceCapture = nil, true
	notify, active := q.invalidate, !q.closed
	q.mu.Unlock()
	// Do not write an old pending revision here. The next UI frame captures
	// current state first, then rearms the same CAS-protected Store.Update.
	if active && notify != nil {
		notify()
	}
}

func (q *resumeCheckpoint) close() {
	q.mu.Lock()
	q.closed = true
	q.pending = nil
	q.stopWriteLocked()
	if q.captureTimer != nil {
		q.captureTimer.Stop()
		q.captureTimer = nil
	}
	q.mu.Unlock()
}

func (s *mainState) closeResumeCheckpoint() {
	s.mu.Lock()
	checkpoint := s.resumeCheckpoint
	s.mu.Unlock()
	if checkpoint != nil {
		checkpoint.close()
	}
}

// mainView defers this until all visible editors and pane projections finish.
// Throttle capture, debounce writes and fingerprint unchanged snapshots: native
// paint/animation frames never cause a full disk write per rendered frame.
func (s *mainState) resumeCheckpointForOwnerLocked() *resumeCheckpoint {
	if s.resumeStore == nil || s.resumeError != "" || s.quitting {
		return nil
	}
	checkpoint := s.resumeCheckpoint
	if checkpoint == nil || checkpoint.owner != s.resumeStore {
		if checkpoint != nil {
			checkpoint.close()
		}
		checkpoint = &resumeCheckpoint{owner: s.resumeStore}
		s.resumeCheckpoint = checkpoint
	}
	return checkpoint
}

// Native lifecycle callbacks already run on the UI owner. A hidden/minimized
// surface need not paint again, so capture the last edit here rather than
// relying on the throttled Invalidate callback to build another mainView.
func (s *mainState) checkpointResumeOnVisibilityChange() {
	s.mu.Lock()
	checkpoint := s.resumeCheckpointForOwnerLocked()
	if checkpoint == nil {
		s.mu.Unlock()
		return
	}
	next, err := s.captureResumeOwnedLocked()
	s.mu.Unlock()
	now := checkpoint.nowTime()
	checkpoint.mu.Lock()
	checkpoint.captureAt, checkpoint.forceCapture = now, false
	if checkpoint.captureTimer != nil {
		checkpoint.captureTimer.Stop()
		checkpoint.captureTimer = nil
	}
	checkpoint.mu.Unlock()
	checkpoint.stage(next, err, now)
}

func (s *mainState) syncResumeCheckpoint(c *ui.Context) {
	s.mu.Lock()
	checkpoint := s.resumeCheckpointForOwnerLocked()
	s.mu.Unlock()
	if checkpoint == nil {
		return
	}
	if checkpoint.due(time.Now(), c.Services().Invalidate) {
		s.mu.Lock()
		next, err := s.captureResumeOwnedLocked()
		s.mu.Unlock()
		checkpoint.stage(next, err, time.Now())
	}
	checkpoint.mu.Lock()
	err := checkpoint.err
	show := err != nil && checkpoint.errorSerial != checkpoint.noticedSerial
	if show {
		checkpoint.noticedSerial = checkpoint.errorSerial
	}
	checkpoint.mu.Unlock()
	if show {
		c.AddToast(ui.Toast{ID: "resume-checkpoint-error", Type: "error", Timeout: -1,
			Title:       i18n.T(i18n.KeyCmdResumeStateWasNotSaved),
			Description: err.Error(), Action: i18n.T(i18n.KeyCmdRetry), OnAction: checkpoint.retry})
	} else if err == nil {
		c.CloseToast("resume-checkpoint-error")
	}
}
