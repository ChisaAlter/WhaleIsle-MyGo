package card

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/openui/lang"
)

type host struct {
	mu     sync.Mutex
	texts  []string
	panels []openui.PanelOperation
	ch     chan struct{}
}

func newHost() *host { return &host{ch: make(chan struct{}, 8)} }

func (h *host) Submit(_ context.Context, s string) error {
	h.mu.Lock()
	h.texts = append(h.texts, s)
	h.mu.Unlock()
	h.ch <- struct{}{}
	return nil
}
func (h *host) OpenURL(string) error { return nil }
func (h *host) Copy(string) error    { return nil }
func (h *host) Panel(op openui.PanelOperation) error {
	h.mu.Lock()
	h.panels = append(h.panels, op)
	h.mu.Unlock()
	h.ch <- struct{}{}
	return nil
}

func (h *host) wait(t *testing.T) {
	t.Helper()
	select {
	case <-h.ch:
	case <-time.After(5 * time.Second):
		t.Fatal("no host call")
	}
}

func TestStreamingCoalescesAndFinalAppliesAtOnce(t *testing.T) {
	now := time.Unix(1000, 0)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	inval := make(chan struct{}, 16)
	c := New(Config{Now: clock, Coalesce: 40 * time.Millisecond, Invalidate: func() { inval <- struct{}{} }})
	defer c.Close()
	c.SetSource("whale-ui", `root = Col([Text("a")`, true)
	r1 := c.Revision()
	c.SetSource("whale-ui", `root = Col([Text("a"), Text("b")`, true)
	if c.Revision() != r1 {
		t.Fatal("second streaming update applied inside the coalescing window")
	}
	select {
	case <-inval:
	case <-time.After(2 * time.Second):
		t.Fatal("pending update never flushed")
	}
	if c.Revision() != r1+1 {
		t.Fatalf("revision %d, want %d", c.Revision(), r1+1)
	}
	c.SetSource("whale-ui", `root = Col([Text("a"), Text("b")])`, false)
	if c.Revision() != r1+2 || c.Streaming() {
		t.Fatal("final source did not apply at once")
	}
	evals := c.Evaluations()
	for i := 0; i < 10; i++ {
		c.SetSource("whale-ui", `root = Col([Text("a"), Text("b")])`, false)
	}
	if c.Evaluations() != evals {
		t.Fatal("unchanged source evaluated again")
	}
}

func TestStaleClickIsIgnored(t *testing.T) {
	h := newHost()
	c := New(Config{Host: h})
	defer c.Close()
	c.SetSource("whale-ui", `root = Col([Button("Send", @ToAssistant("one"))])`, false)
	old := c.Frame()
	btn := old.Root.Children()[0]
	c.SetSource("whale-ui", `root = Col([Button("Send", @ToAssistant("two"))])`, false)
	c.Dispatch(old.Revision, btn.Identity(), "Send", "", btn.Prop("action"))
	time.Sleep(50 * time.Millisecond)
	if len(h.texts) != 0 {
		t.Fatalf("stale click submitted %q", h.texts)
	}
	f := c.Frame()
	nb := f.Root.Children()[0]
	c.Dispatch(f.Revision, nb.Identity(), "Send", "", nb.Prop("action"))
	h.wait(t)
	if h.texts[0] != "two" {
		t.Fatalf("submitted %q", h.texts)
	}
}

func TestPanelSpecPublishesOnce(t *testing.T) {
	h := newHost()
	c := New(Config{Host: h})
	defer c.Close()
	spec := `{"panel":true,"append":true,"items":[{"type":"text","content":"hi"}]}`
	c.SetSource("dsh-ui", spec[:30], true)
	c.SetSource("dsh-ui", spec, false)
	h.wait(t)
	c.SetSource("dsh-ui", spec, false)
	time.Sleep(30 * time.Millisecond)
	if len(h.panels) != 1 || !h.panels[0].Append || strings.Contains(string(h.panels[0].Spec), "panel") {
		t.Fatalf("panels = %+v", h.panels)
	}
	if _, err := openui.ValidatePanelOperation([]byte(`{"append":true,"spec":` + string(h.panels[0].Spec) + `}`)); err != nil {
		t.Fatal(err)
	}
	if !c.Panel() {
		t.Fatal("card does not report its panel spec")
	}
}

func TestComponentStateAndFieldsPersist(t *testing.T) {
	dir := t.TempDir()
	id := strings.Repeat("cd", 32)
	c := New(Config{StateDir: dir, StateID: id})
	c.SetSource("whale-ui", `root = Col([Input("x", "Plain")])`, false)
	c.SetComponentState("a:Flashcard:", "flipped", true)
	c.SetField("", "Input", "Plain", "typed")
	c.Close()
	raw, err := openui.ReadState(dir, id)
	if err != nil {
		t.Fatal(err)
	}
	vals, err := lang.UnmarshalState(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"format":"whale-ui-state"`) || vals.Len() != 2 {
		t.Fatalf("state = %s", raw)
	}
	c2 := New(Config{StateDir: dir, StateID: id})
	defer c2.Close()
	for !c2.Loaded() {
		time.Sleep(time.Millisecond)
	}
	c2.SetSource("whale-ui", `root = Col([Input("x", "Plain")])`, false)
	if c2.ComponentState("a:Flashcard:", "flipped") != true || c2.FieldValue("", "Plain") != "typed" {
		t.Fatal("state did not round-trip")
	}
}

func TestCacheEvictsAndCloses(t *testing.T) {
	cache := NewCache(2)
	a := cache.Get("a", func() *Card { return New(Config{}) })
	cache.Get("b", func() *Card { return New(Config{}) })
	if cache.Get("a", func() *Card { t.Fatal("recreated"); return nil }) != a {
		t.Fatal("cache miss")
	}
	cache.Get("c", func() *Card { return New(Config{}) })
	if cache.Len() != 2 {
		t.Fatalf("len %d", cache.Len())
	}
}
