package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/recovery"
	"whaleisle/internal/ui/composer"
	"whaleisle/internal/ui/files"
	"whaleisle/internal/ui/frame"
)

type checkpointTestClock struct {
	mu     sync.Mutex
	time   time.Time
	timers []*checkpointTestTimer
}

type checkpointTestTimer struct {
	clock          *checkpointTestClock
	at             time.Time
	fn             func()
	stopped, fired bool
}

func (c *checkpointTestClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.time
}

func (c *checkpointTestClock) after(delay time.Duration, fn func()) resumeCheckpointTimer {
	c.mu.Lock()
	defer c.mu.Unlock()
	timer := &checkpointTestTimer{clock: c, at: c.time.Add(delay), fn: fn}
	c.timers = append(c.timers, timer)
	return timer
}

func (t *checkpointTestTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	active := !t.stopped && !t.fired
	t.stopped = true
	return active
}

func (c *checkpointTestClock) advance(delay time.Duration) {
	c.mu.Lock()
	until := c.time.Add(delay)
	for {
		var next *checkpointTestTimer
		for _, timer := range c.timers {
			if !timer.stopped && !timer.fired && !timer.at.After(until) && (next == nil || timer.at.Before(next.at)) {
				next = timer
			}
		}
		if next == nil {
			c.time = until
			c.mu.Unlock()
			return
		}
		c.time, next.fired = next.at, true
		c.mu.Unlock()
		next.fn()
		c.mu.Lock()
	}
}

func checkpointFixture(t *testing.T) (*mainState, *resumeCheckpoint, *files.FileTab, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := recovery.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	tab := &files.FileTab{Path: "owned.txt", Preview: "disk baseline", Draft: "first editor draft", Editing: true}
	s := &mainState{resumeStore: store, layout: frame.DefaultLayout(), terminalSavedBottom: 280,
		composerDrafts: map[string]*composer.DraftState{"owned-session": composer.NewDraftState("first composer draft")},
		fileTabs:       files.Tabs{Workspaces: map[string]*files.WorkspaceTabs{"owned-workspace": {Open: []*files.FileTab{tab}, Active: tab.Path}}}}
	clock := &checkpointTestClock{time: time.Unix(0, 0)}
	q := &resumeCheckpoint{owner: store, now: clock.now, after: clock.after}
	s.resumeCheckpoint = q
	t.Cleanup(q.close)
	return s, q, tab, dir
}

func captureCheckpoint(t *testing.T, s *mainState) recovery.State {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	next, err := s.captureResumeOwnedLocked()
	if err != nil {
		t.Fatal(err)
	}
	return next
}

// Exercise the production disk worker deterministically after suppressing only
// its delay. Real Store.Update, atomic file bytes and CAS checks remain active.
func writeCheckpointNow(q *resumeCheckpoint) {
	q.mu.Lock()
	q.stopWriteLocked()
	epoch := q.writeEpoch
	q.mu.Unlock()
	q.write(epoch)
}

func savedCheckpointDrafts(t *testing.T, state recovery.State) (string, string) {
	t.Helper()
	draft, err := composer.RestoreDraftChecked(state.Drafts["owned-session"])
	if err != nil {
		t.Fatal(err)
	}
	var tabs files.Tabs
	if err := json.Unmarshal(state.Files, &tabs); err != nil {
		t.Fatal(err)
	}
	workspace := tabs.Workspaces["owned-workspace"]
	if workspace == nil || len(workspace.Open) != 1 || workspace.Open[0].Preview != "disk baseline" {
		t.Fatal("checkpoint lost the editor baseline or workspace owner")
	}
	return draft.Text, workspace.Open[0].Draft
}

func TestResumeCheckpointDetachedLatestCaptureAndQuestionOwnership(t *testing.T) {
	s, q, tab, dir := checkpointFixture(t)
	next := captureCheckpoint(t, s)
	// The injected clock leaves all delayed callbacks dormant; explicitly drive
	// the real writer after changing the native owner's mutable state.
	now := time.Now().Add(time.Hour)
	q.stage(next, nil, now)
	s.composerDrafts["owned-session"].Text, tab.Draft = "second composer draft", "second editor draft"
	question := json.RawMessage(`{"owner":"isolated-profile","draft":"newer answer"}`)
	if err := s.resumeStore.Update(func(state *recovery.State) { state.QuestionDrafts = question }); err != nil {
		t.Fatal(err)
	}
	writeCheckpointNow(q)
	first := s.resumeStore.Snapshot()
	if composerText, fileText := savedCheckpointDrafts(t, first); composerText != "first composer draft" || fileText != "first editor draft" {
		t.Fatal("disk worker read mutable UI state instead of its detached capture")
	}
	if !bytes.Equal(first.QuestionDrafts, question) {
		t.Fatal("a combined checkpoint replaced the independent question writer")
	}
	q.stage(captureCheckpoint(t, s), nil, now.Add(time.Second))
	s.composerDrafts["owned-session"].Text, tab.Draft = "latest composer draft", "latest editor draft"
	latest := captureCheckpoint(t, s)
	q.stage(latest, nil, now.Add(2*time.Second))
	writeCheckpointNow(q)
	if composerText, fileText := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "latest composer draft" || fileText != "latest editor draft" {
		t.Fatal("a coalesced checkpoint wrote an older queued revision")
	}
	before, err := os.ReadFile(filepath.Join(dir, recovery.FileName))
	if err != nil {
		t.Fatal(err)
	}
	q.stage(latest, nil, now.Add(3*time.Second))
	q.mu.Lock()
	unchanged := q.pending == nil
	q.mu.Unlock()
	if !unchanged {
		t.Fatal("an unchanged repaint queued another disk transaction")
	}
	writeCheckpointNow(q)
	after, _ := os.ReadFile(filepath.Join(dir, recovery.FileName))
	if !bytes.Equal(before, after) {
		t.Fatal("an unchanged checkpoint rewrote saved data")
	}
}

func TestResumeCheckpointConflictRetainsLatestAndExplicitRetryUsesFreshUI(t *testing.T) {
	s, q, tab, dir := checkpointFixture(t)
	q.stage(captureCheckpoint(t, s), nil, time.Now().Add(time.Hour))
	writeCheckpointNow(q)
	filename := filepath.Join(dir, recovery.FileName)
	baseline, _ := os.ReadFile(filename)
	other, err := recovery.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := other.Update(func(state *recovery.State) { state.SelectedSession = "other-instance" }); err != nil {
		t.Fatal(err)
	}
	external, _ := os.ReadFile(filename)
	s.composerDrafts["owned-session"].Text, tab.Draft = "blocked draft", "blocked editor"
	q.stage(captureCheckpoint(t, s), nil, time.Now().Add(time.Hour))
	writeCheckpointNow(q)
	q.mu.Lock()
	conflict, pending := q.err, q.pending
	q.mu.Unlock()
	if !errors.Is(conflict, recovery.ErrConflict) || pending == nil {
		t.Fatalf("conflict lost pending recovery: error=%v pending=%v", conflict, pending)
	}
	if after, _ := os.ReadFile(filename); !bytes.Equal(after, external) {
		t.Fatal("checkpoint overwrote another instance's newer file")
	}
	s.composerDrafts["owned-session"].Text, tab.Draft = "retry current draft", "retry current editor"
	q.stage(captureCheckpoint(t, s), nil, time.Now().Add(time.Hour))
	q.mu.Lock()
	blocked := q.writeTimer == nil
	q.mu.Unlock()
	if !blocked {
		t.Fatal("UI changes automatically retried a failed CAS transaction")
	}
	// Simulate the explicit owner resolving the conflict by restoring its own
	// reviewed baseline in this private fixture. The CAS guard stays enabled.
	if err := os.WriteFile(filename, baseline, 0600); err != nil {
		t.Fatal(err)
	}
	q.mu.Lock()
	q.captureAt = time.Now()
	q.mu.Unlock()
	q.retry()
	if !q.due(time.Now(), nil) {
		t.Fatal("Retry did not request a fresh UI-owned capture")
	}
	q.stage(captureCheckpoint(t, s), nil, time.Now().Add(time.Hour))
	writeCheckpointNow(q)
	if composerText, fileText := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "retry current draft" || fileText != "retry current editor" {
		t.Fatal("Retry wrote the failed older snapshot instead of current UI state")
	}
}

func TestResumeCheckpointSamePayloadAfterCaptureFailureIsNotLost(t *testing.T) {
	s, q, _, dir := checkpointFixture(t)
	a := captureCheckpoint(t, s)
	q.stage(a, nil, time.Now().Add(time.Hour))
	// Both serialization and size errors take the capture-error path. Use real
	// invalid JSON here instead of allocating a 32-MiB editor on every test run.
	bad := a
	bad.Layout.WorkspacePanes = json.RawMessage(`{invalid pane snapshot`)
	q.stage(bad, nil, time.Now().Add(time.Hour))
	if _, err := os.Stat(filepath.Join(dir, recovery.FileName)); !os.IsNotExist(err) {
		t.Fatal("the unsaved A snapshot unexpectedly reached disk before retry")
	}
	q.retry()
	q.stage(a, nil, time.Now().Add(time.Hour))
	writeCheckpointNow(q)
	if composerText, fileText := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "first composer draft" || fileText != "first editor draft" {
		t.Fatal("A -> capture failure -> A was mistaken for a persisted unchanged snapshot")
	}
}

func TestResumeCheckpointCanceledTimerCannotPublishNewerCaptureEarly(t *testing.T) {
	s, q, tab, dir := checkpointFixture(t)
	now := time.Now().Add(time.Hour)
	q.stage(captureCheckpoint(t, s), nil, now)
	q.mu.Lock()
	oldTimerEpoch := q.writeEpoch
	q.mu.Unlock()
	s.composerDrafts["owned-session"].Text, tab.Draft = "new capture", "new editor"
	q.stage(captureCheckpoint(t, s), nil, now.Add(time.Second))
	q.write(oldTimerEpoch) // A callback already dispatched when Stop ran.
	if _, err := os.Stat(filepath.Join(dir, recovery.FileName)); !os.IsNotExist(err) {
		t.Fatal("a canceled older timer bypassed the current capture's quiet period")
	}
	writeCheckpointNow(q)
	if composerText, fileText := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "new capture" || fileText != "new editor" {
		t.Fatal("canceling an obsolete timer discarded the newest immutable capture")
	}
}

func TestResumeQuitBarrierPreventsOlderWriteAfterFinalSnapshot(t *testing.T) {
	s, q, tab, _ := checkpointFixture(t)
	q.stage(captureCheckpoint(t, s), nil, time.Now().Add(time.Hour))
	held, release, questionDone := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		questionDone <- s.resumeStore.Update(func(state *recovery.State) {
			close(held)
			<-release
			state.QuestionDrafts = json.RawMessage(`{"draft":"question persisted before final save"}`)
		})
	}()
	<-held
	go writeCheckpointNow(q)
	deadline := time.Now().Add(3 * time.Second)
	for {
		q.mu.Lock()
		running := q.running != nil
		q.mu.Unlock()
		if running {
			break
		}
		if time.Now().After(deadline) {
			close(release)
			t.Fatal("checkpoint worker did not enter the real Store transaction")
		}
		time.Sleep(time.Millisecond)
	}
	s.composerDrafts["owned-session"].Text, tab.Draft = "final composer draft", "final editor draft"
	finished := make(chan error, 1)
	go func() { finished <- s.saveResume() }()
	select {
	case err := <-finished:
		close(release)
		t.Fatalf("quit did not wait for the active earlier writer: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	close(release)
	if err := <-questionDone; err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("quit barrier deadlocked with the independent question writer")
	}
	saved := s.resumeStore.Snapshot()
	if composerText, fileText := savedCheckpointDrafts(t, saved); composerText != "final composer draft" || fileText != "final editor draft" || !bytes.Contains(saved.QuestionDrafts, []byte("question persisted")) {
		t.Fatal("the old queued checkpoint overwrote the final UI/question snapshot")
	}
	q.mu.Lock()
	quiet := q.pending == nil && q.running == nil && q.writeTimer == nil && q.paused == 0
	q.mu.Unlock()
	if !quiet {
		t.Fatal("final save left an older checkpoint able to write afterwards")
	}
}

func TestResumeDrawerRestoresPresentationWithoutOpeningTerminal(t *testing.T) {
	dir := t.TempDir()
	store, err := recovery.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(state *recovery.State) { state.Layout = recovery.Layout{Bottom: 310, TerminalOpen: true} }); err != nil {
		t.Fatal(err)
	}
	s := &mainState{layout: frame.DefaultLayout(), terminalSavedBottom: 280}
	s.layout.Bottom = 0
	s.openResume(dir)
	if s.resumeError != "" || !s.terminalVisible || s.layout.Bottom != 310 || !s.terminalResumePending {
		t.Fatalf("drawer presentation not restored: visible=%v height=%v pending=%v err=%s", s.terminalVisible, s.layout.Bottom, s.terminalResumePending, s.resumeError)
	}
	// Even if the gate regresses, this guaranteed-missing owned path cannot
	// start a PTY; the assertion still detects an attempted terminal creation.
	s.ensureTerminal(filepath.Join(t.TempDir(), "not-created"), false)
	if s.terminalPanel != nil || s.terminalSessions.RunningCount() != 0 {
		t.Fatal("restoring drawer geometry attempted to open a shell")
	}
	if next := captureCheckpoint(t, s); !next.Layout.TerminalOpen || next.Layout.Bottom != 310 {
		t.Fatal("restored open drawer lost its persistence fields before explicit Start")
	}
}

func TestResumeCheckpointQuietAndContinuousEditsHaveBoundedWriteLatency(t *testing.T) {
	for _, mode := range []string{"quiet", "continuous"} {
		t.Run(mode, func(t *testing.T) {
			s, q, tab, dir := checkpointFixture(t)
			clock := &checkpointTestClock{time: time.Unix(0, 0)}
			q.now, q.after = clock.now, clock.after
			filename := filepath.Join(dir, recovery.FileName)
			capture := func(index int) {
				s.composerDrafts["owned-session"].Text = fmt.Sprintf("edit-%02d", index)
				tab.Draft = fmt.Sprintf("editor-%02d", index)
				q.stage(captureCheckpoint(t, s), nil, clock.now())
			}
			if mode == "quiet" {
				capture(0)
				clock.advance(599 * time.Millisecond)
				if _, err := os.Stat(filename); !os.IsNotExist(err) {
					t.Fatal("quiet capture wrote before the 600ms delay")
				}
				clock.advance(time.Millisecond)
				if composerText, _ := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "edit-00" {
					t.Fatal("quiet capture did not persist after the trailing delay")
				}
				return
			}
			// A new edit arrives every 100ms, so no 600ms quiet period exists.
			for i := 0; i < 49; i++ {
				capture(i)
				clock.advance(100 * time.Millisecond)
			}
			if _, err := os.Stat(filename); !os.IsNotExist(err) {
				t.Fatal("continuous capture wrote before its five-second deadline")
			}
			capture(49)
			clock.advance(100 * time.Millisecond)
			if composerText, fileText := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "edit-49" || fileText != "editor-49" {
				t.Fatal("continuous edits postponed recovery indefinitely or wrote an obsolete capture")
			}
			for i := 50; i < 99; i++ {
				capture(i)
				clock.advance(100 * time.Millisecond)
			}
			if composerText, _ := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "edit-49" {
				t.Fatal("a new pending batch inherited the previous batch's max-wait clock")
			}
			capture(99)
			clock.advance(100 * time.Millisecond)
			if composerText, fileText := savedCheckpointDrafts(t, s.resumeStore.Snapshot()); composerText != "edit-99" || fileText != "editor-99" {
				t.Fatal("the next continuous batch missed its independent five-second deadline")
			}
		})
	}
}
