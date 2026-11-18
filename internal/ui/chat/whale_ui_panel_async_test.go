package chat

import (
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/paths"
	"whaleisle/internal/store/session"
	"whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

func resetWhalePanelsForTest() {
	whalePanels.Lock()
	whalePanels.values = map[string]*whalePanelState{}
	whalePanels.Unlock()
}

func TestWhalePanelRenderAndButtonsNeverBlockOnSlowDiskIO(t *testing.T) {
	resetWhalePanelsForTest()
	t.Cleanup(resetWhalePanelsForTest)

	root := t.TempDir()
	t.Setenv(paths.EnvUserData, root)
	oldLocale := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(oldLocale) })

	m := Model{
		ThemeID:     theme.IDDeepseekLight,
		Snapshot:    session.ViewState{SelectedID: "async-ui-panel", PlanHistoryBinding: "binding"},
		Preferences: settings.ChatPreferences{TranscriptView: "standard"},
	}

	// Seed persisted state on disk first using a separate instance, then clear in-memory map.
	seed := whalePanelFor(m)
	first := json.RawMessage(`{"append":false,"spec":{"title":"Async","items":[{"type":"text","content":"Persisted item"}]}}`)
	if err := seed.Publish("seed", [3]float64{1, 0, 0}, first); err != nil {
		t.Fatal(err)
	}
	if err := seed.Flush(); err != nil {
		t.Fatal(err)
	}
	resetWhalePanelsForTest()

	readStarted := make(chan struct{})
	readRelease := make(chan struct{})
	oldRead := whalePanelReadFileFn
	whalePanelReadFileFn = func(path string) ([]byte, error) {
		if path == seed.Path {
			close(readStarted)
			<-readRelease
		}
		return oldRead(path)
	}
	t.Cleanup(func() { whalePanelReadFileFn = oldRead })

	// First UI render must return immediately without blocking on initial os.ReadFile.
	rendered := make(chan *ui.Tester, 1)
	go func() {
		tt := ui.NewTester(func(c *ui.Context) {
			whaleUIPanelView(c, m)
		}, 640, 320)
		rendered <- tt
	}()

	var tt *ui.Tester
	select {
	case tt = <-rendered:
	case <-time.After(2 * time.Second):
		t.Fatal("whaleUIPanelView blocked UI render frame waiting for initial os.ReadFile")
	}

	select {
	case <-readStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("background initial panel read did not start")
	}

	// Unblock disk read and await UI invalidation/repaint.
	close(readRelease)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if tt.HasText("Conversation panel") {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !tt.HasText("Conversation panel") {
		t.Fatalf("panel header did not appear after async initial load; texts=%v", tt.Texts())
	}

	p := whalePanelFor(m)
	p.mu.Lock()
	p.debounce = time.Millisecond
	p.mu.Unlock()

	// Block disk writes and inject a write failure to verify non-blocking collapse
	// toggle + asynchronous rollback with Invalidate().
	writeStarted := make(chan struct{}, 4)
	writeRelease := make(chan struct{})
	oldWrite := whalePanelWriteFileFn
	whalePanelWriteFileFn = func(path string, raw []byte) error {
		select {
		case writeStarted <- struct{}{}:
		default:
		}
		<-writeRelease
		return errors.New("simulated disk failure")
	}
	t.Cleanup(func() { whalePanelWriteFileFn = oldWrite })

	clickDone := make(chan error, 1)
	go func() {
		clickDone <- tt.Click("Collapse / expand")
	}()
	select {
	case err := <-clickDone:
		if err != nil {
			t.Fatalf("collapse click failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("collapse toggle click blocked on disk write")
	}

	p.mu.Lock()
	collapsedOptimistic := p.Collapsed
	p.mu.Unlock()
	if !collapsedOptimistic {
		t.Fatal("expected optimistic collapsed=true immediately after UI click")
	}

	select {
	case <-writeStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("async persist worker did not start after collapse toggle")
	}

	// Release failing write and wait for async rollback + error surface.
	close(writeRelease)
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		p.mu.Lock()
		errText := p.Err
		collapsed := p.Collapsed
		p.mu.Unlock()
		if errText != "" && !collapsed {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	p.mu.Lock()
	gotCollapsed, gotErr := p.Collapsed, p.Err
	p.mu.Unlock()
	if gotCollapsed || gotErr == "" {
		t.Fatalf("expected rollback on async write failure, got collapsed=%v err=%q", gotCollapsed, gotErr)
	}
}

func TestWhalePanelDebounceDedupSerialAndGenerationOrdering(t *testing.T) {
	resetWhalePanelsForTest()
	t.Cleanup(resetWhalePanelsForTest)
	t.Setenv(paths.EnvUserData, t.TempDir())

	m := Model{Snapshot: session.ViewState{SelectedID: "debounce-panel", PlanHistoryBinding: "binding"}}
	p := whalePanelFor(m)
	p.WaitLoaded()

	var writes atomic.Int32
	var active atomic.Int32
	var maxActive atomic.Int32

	p.mu.Lock()
	p.debounce = 30 * time.Millisecond
	p.writeFile = func(path string, raw []byte) error {
		cur := active.Add(1)
		for {
			prev := maxActive.Load()
			if cur <= prev || maxActive.CompareAndSwap(prev, cur) {
				break
			}
		}
		defer active.Add(-1)
		writes.Add(1)
		return writeWhalePanelFile(path, raw)
	}
	p.mu.Unlock()

	// Burst 5 updates inside debounce window: must coalesce into 1 disk write.
	for i := 1; i <= 5; i++ {
		op := json.RawMessage(`{"append":false,"spec":{"title":"Burst","items":[{"type":"text","content":"step"}]}}`)
		if i > 1 {
			op = json.RawMessage(`{"append":true,"spec":{"items":[{"type":"text","content":"delta"}]}}`)
		}
		if err := p.Publish("src", [3]float64{float64(i), 0, 0}, op); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		p.mu.Lock()
		done := p.persistedSeq == p.persistSeq && !p.persistWorkerStarted
		p.mu.Unlock()
		if done {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if n := writes.Load(); n != 1 {
		t.Fatalf("expected burst within debounce to coalesce into 1 write, got %d", n)
	}
	if maxActive.Load() != 1 {
		t.Fatalf("expected strictly serial persistence, max concurrent writes=%d", maxActive.Load())
	}

	// Double collapse toggle within debounce returns to identical persisted JSON:
	// dedup must avoid an unnecessary disk write.
	beforeDedup := writes.Load()
	p.toggleCollapsed()
	p.toggleCollapsed()
	if err := p.Flush(); err != nil {
		t.Fatal(err)
	}
	if writes.Load() != beforeDedup {
		t.Fatalf("expected dedup to skip no-op roundtrip write, writes %d -> %d", beforeDedup, writes.Load())
	}

	// If an older generation write fails while a newer generation is submitted in flight,
	// the stale failure must NOT roll back the newer in-memory generation.
	firstWriteStarted := make(chan struct{})
	firstWriteRelease := make(chan struct{})
	var seqWrites atomic.Int32
	p.mu.Lock()
	p.debounce = time.Millisecond
	p.writeFile = func(path string, raw []byte) error {
		n := seqWrites.Add(1)
		if n == 1 {
			close(firstWriteStarted)
			<-firstWriteRelease
			return errors.New("stale generation transient error")
		}
		return writeWhalePanelFile(path, raw)
	}
	p.mu.Unlock()

	staleOp := json.RawMessage(`{"append":false,"spec":{"title":"GenOld","items":[{"type":"text","content":"old"}]}}`)
	if err := p.Publish("gen", [3]float64{10, 0, 0}, staleOp); err != nil {
		t.Fatal(err)
	}
	select {
	case <-firstWriteStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first generation write did not start")
	}

	// While gen 1 write is blocked in flight, publish newer gen 2.
	newerOp := json.RawMessage(`{"append":false,"spec":{"title":"GenNew","items":[{"type":"text","content":"fresh"}]}}`)
	if err := p.Publish("gen", [3]float64{11, 0, 0}, newerOp); err != nil {
		t.Fatal(err)
	}
	close(firstWriteRelease)

	if err := p.Flush(); err != nil {
		t.Fatalf("expected newer generation persist to succeed after stale failure, got %v", err)
	}
	p.mu.Lock()
	snap := string(p.Snapshot)
	errText := p.Err
	p.mu.Unlock()
	if errText != "" || snap == "" || !json.Valid([]byte(snap)) {
		t.Fatalf("unexpected state after newer gen flush: snap=%q err=%q", snap, errText)
	}
	if !strings.Contains(snap, "fresh") {
		t.Fatalf("stale write failure clobbered newer generation snapshot: %s", snap)
	}
}
