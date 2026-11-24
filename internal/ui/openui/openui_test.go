package openui

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"whaleisle/internal/desktop/openui"
	"whaleisle/internal/openui/card"
	"whaleisle/internal/ui/design"
)

// fakeHost records host effects.
type fakeHost struct {
	mu        sync.Mutex
	submitted []string
	opened    []string
	copied    []string
	panels    []openui.PanelOperation
	submitErr error
	done      chan struct{}
}

func newHost() *fakeHost { return &fakeHost{done: make(chan struct{}, 16)} }

func (h *fakeHost) Submit(ctx context.Context, text string) error {
	h.mu.Lock()
	h.submitted = append(h.submitted, text)
	err := h.submitErr
	h.mu.Unlock()
	h.done <- struct{}{}
	return err
}
func (h *fakeHost) OpenURL(u string) error {
	h.mu.Lock()
	h.opened = append(h.opened, u)
	h.mu.Unlock()
	h.done <- struct{}{}
	return nil
}
func (h *fakeHost) Copy(t string) error {
	h.mu.Lock()
	h.copied = append(h.copied, t)
	h.mu.Unlock()
	h.done <- struct{}{}
	return nil
}
func (h *fakeHost) Panel(op openui.PanelOperation) error {
	h.mu.Lock()
	h.panels = append(h.panels, op)
	h.mu.Unlock()
	h.done <- struct{}{}
	return nil
}

func (h *fakeHost) wait(t *testing.T) {
	t.Helper()
	select {
	case <-h.done:
	case <-time.After(5 * time.Second):
		t.Fatal("host effect did not run")
	}
}

// harness renders one card in a headless tester.
type harness struct {
	t    *testing.T
	card *card.Card
	host *fakeHost
	tt   *ui.Tester
	dark bool
	opts Options
	// compat counts compatibility-workbench opens.
	compat int
	links  []string
}

func newHarness(t *testing.T, program string, cfg card.Config) *harness {
	t.Helper()
	h := &harness{t: t, host: newHost()}
	if cfg.Host == nil {
		cfg.Host = h.host
	}
	h.card = card.New(cfg)
	t.Cleanup(h.card.Close)
	h.card.SetSource("whale-ui", program, false)
	h.tt = ui.NewTester(func(c *ui.Context) {
		th := *c.Theme()
		th.Dark = h.dark
		c.SetTheme(&th)
		ui.Column(c).Fill().Background(design.For(c).Background).Padding(16).Children(func() {
			o := h.opts
			o.Runtime = h.card
			o.Width = 560
			o.OpenCompat = func() { h.compat++ }
			o.OpenURL = func(u string) { h.links = append(h.links, u) }
			View(c, o)
		})
	}, 600, 900)
	return h
}

func (h *harness) frame() { h.tt.Frame() }

func (h *harness) mustText(s string) {
	h.t.Helper()
	h.frame()
	if !h.tt.HasText(s) {
		h.t.Fatalf("text %q not shown; texts: %q", s, h.tt.Texts())
	}
}

func (h *harness) noText(s string) {
	h.t.Helper()
	for _, x := range h.tt.Texts() {
		if strings.Contains(x, s) {
			h.t.Fatalf("text %q shown: %q", s, h.tt.Texts())
		}
	}
}

func (h *harness) click(s string) {
	h.t.Helper()
	h.frame()
	if err := h.tt.Click(s); err != nil {
		h.t.Fatalf("click %q: %v; texts: %q", s, err, h.tt.Texts())
	}
	h.frame()
}

var errBoom = errors.New("connection refused")
