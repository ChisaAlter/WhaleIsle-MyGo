// Package pad connects native Pad controls and isolated card bodies to the real
// shipped dsh-pad service. It does not open the full Harness browser UI.
package pad

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/webview"
	store "whaleisle/internal/store/pad"
)

type Options struct {
	Parent              *mygo.Window
	UserData, VendorDir string
	Changed             func()
	// OnAsk targets the current native conversation. insert preserves its draft;
	// neither operation sends automatically. Rejected handoffs remain visible.
	OnAsk     func(text string, insert bool) error
	OnOpenURL func(string) error
	OnFailure func(webview.PageFailure)
}
type Appearance struct {
	CSS  string `json:"css"`
	Mode string `json:"mode"`
	Lang string `json:"lang"`
}
type Desktop struct {
	mu             sync.Mutex
	opts           Options
	Store          *store.Store
	gateway        *pluginslots.Gateway
	origin, cookie string
	generation     uint64
	runtime        string
	cards          map[string]*card
	wanted         map[string]bool
	appearance     Appearance
	autoHeights    map[string]float32
	masked, closed bool
	pendingAsk     string
	off            []func()
}

func NewDesktop(opts Options) (*Desktop, error) {
	if opts.Parent == nil {
		return nil, errors.New("Pad: parent required")
	}
	d := &Desktop{opts: opts, cards: map[string]*card{}, wanted: map[string]bool{}, appearance: Appearance{CSS: ":root{--pad-fg:#151515;--pad-bg:#fff}", Mode: "light", Lang: "zh-CN"}}
	s, err := store.Open(opts.UserData, d.changed)
	if err != nil {
		return nil, err
	}
	d.Store = s
	// Reuse the reviewed release's complete dsh runtime (charts, formatting,
	// state, run and message semantics), not a second hand-written panel SDK.
	raw, err := os.ReadFile(filepath.Join(opts.VendorDir, "dsh-pad", "lib", "client.js"))
	if err != nil {
		return nil, err
	}
	d.runtime, err = frameRuntime(string(raw))
	if err != nil {
		return nil, err
	}
	d.off = []func(){opts.Parent.OnClosed(d.Close), opts.Parent.OnHide(d.Hide), opts.Parent.OnMinimize(d.Hide)}
	return d, nil
}
func frameRuntime(bundle string) (string, error) {
	start := "var FRAME_RUNTIME = String.raw`"
	i := strings.Index(bundle, start)
	if i < 0 {
		return "", errors.New("Pad: unsupported bundled frame runtime")
	}
	text := bundle[i+len(start):]
	j := strings.Index(text, "\n`;")
	if j < 0 {
		return "", errors.New("Pad: incomplete bundled frame runtime")
	}
	return text[:j], nil
}
func (d *Desktop) changed() {
	d.mu.Lock()
	cards := make([]*card, 0, len(d.cards))
	for _, c := range d.cards {
		cards = append(cards, c)
	}
	d.mu.Unlock()
	for _, c := range cards {
		c.update()
	}
	if d.opts.Changed != nil {
		d.opts.Changed()
	}
}
func (d *Desktop) Sync(origin, cookie string, generation uint64) {
	d.mu.Lock()
	if d.closed || d.origin == origin && d.cookie == cookie && d.generation == generation {
		d.mu.Unlock()
		return
	}
	old, cards := d.gateway, d.cards
	d.gateway = nil
	d.cards = map[string]*card{}
	d.origin, d.cookie, d.generation = origin, cookie, generation
	d.mu.Unlock()
	for _, c := range cards {
		c.close()
	}
	if old != nil {
		old.Close()
	}
	if origin == "" {
		d.Store.Detach()
		return
	}
	client, err := store.NewClient(origin, cookie)
	if err != nil {
		d.Store.Detach()
		d.Store.SetError(err)
		return
	}
	gateway, err := pluginslots.NewGatewayWithHandler(origin, cookie, d.serve)
	if err != nil {
		d.Store.Detach()
		d.Store.SetError(err)
		return
	}
	d.mu.Lock()
	if d.closed || d.generation != generation || d.origin != origin || d.cookie != cookie {
		d.mu.Unlock()
		gateway.Close()
		return
	}
	d.gateway = gateway
	d.mu.Unlock()
	d.Store.Attach(client, generation)
}
func (d *Desktop) Ask(text string, insert bool) {
	d.mu.Lock()
	d.pendingAsk = text
	d.mu.Unlock()
	if d.opts.OnAsk == nil {
		mygo.Clipboard.WriteText(text)
		d.Store.SetNotice(errors.New("Pad: no conversation is selected"))
		return
	}
	err := d.opts.OnAsk(text, insert)
	if err == nil {
		d.mu.Lock()
		d.pendingAsk = ""
		d.mu.Unlock()
	} else {
		mygo.Clipboard.WriteText(text)
	}
	d.Store.SetNotice(err)
}

// PendingAsk retains the rejected handoff without changing the user's draft.
func (d *Desktop) PendingAsk() string { d.mu.Lock(); defer d.mu.Unlock(); return d.pendingAsk }
func (d *Desktop) DismissAsk()        { d.mu.Lock(); d.pendingAsk = ""; d.mu.Unlock(); d.Store.SetNotice(nil) }

// ReservedWidth matches the original right-edge dock's content reservation.
// The Shell applies this to its main content, not to the Pad overlay viewport.
func (d *Desktop) ReservedWidth(viewport float32) float32 {
	snap := d.Store.Snapshot()
	if viewport < 1280 || !snap.Attached || snap.Disabled || !snap.Layout.Expanded {
		return 0
	}
	width := min(snap.Layout.Width, max(float32(0), viewport-16))
	if snap.Layout.Dock == nil || snap.Layout.Dock.X+width >= viewport-14 {
		return width + 12
	}
	return 0
}
func (d *Desktop) BeginFrame(appearance Appearance) {
	d.mu.Lock()
	changed := d.appearance != appearance
	d.appearance = appearance
	d.wanted = map[string]bool{}
	d.mu.Unlock()
	if changed {
		d.changed()
	}
}
func (d *Desktop) EndFrame() {
	d.mu.Lock()
	var hidden []*card
	for id, c := range d.cards {
		if !d.wanted[id] || d.masked {
			hidden = append(hidden, c)
		}
	}
	d.mu.Unlock()
	for _, c := range hidden {
		c.close()
	}
}
func (d *Desktop) SetMasked(masked bool) {
	d.mu.Lock()
	d.masked = masked
	d.mu.Unlock()
	if masked {
		d.Hide()
	}
}
func (d *Desktop) Hide() {
	d.mu.Lock()
	cards := make([]*card, 0, len(d.cards))
	for _, c := range d.cards {
		cards = append(cards, c)
	}
	d.mu.Unlock()
	for _, c := range cards {
		c.close()
	}
}
func (d *Desktop) Close() {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return
	}
	d.closed = true
	cards, gateway, off := d.cards, d.gateway, d.off
	d.cards = nil
	d.gateway = nil
	d.off = nil
	d.mu.Unlock()
	d.Store.Detach()
	for _, c := range cards {
		c.close()
	}
	if gateway != nil {
		gateway.Close()
	}
	for _, f := range off {
		f()
	}
}

// Body is only a native-clipped HTML/script card body. Headers, menus,
// permissions, layouts, model choice and the floating dock are native controls.
func (d *Desktop) Body(c *ui.Context, p store.Panel, height float32, focused bool) {
	key := p.ID
	if focused {
		key += "/focus"
	}
	d.mu.Lock()
	d.wanted[key] = true
	d.mu.Unlock()
	ui.Box(c).Key("pad-body-" + key).FillWidth().Height(height).DrawOver(func(painter *ui.Painter, r ui.Rect) {
		clip := painter.NativeSurfaceClip()
		rect := func(v ui.Rect) mygo.Rectangle {
			return mygo.Rectangle{X: int(v.X), Y: int(v.Y), Width: max(0, int(v.W)), Height: max(0, int(v.H))}
		}
		d.present(key, p, rect(r), rect(clip))
	})
}
func (d *Desktop) present(key string, p store.Panel, bounds, clip mygo.Rectangle) {
	d.mu.Lock()
	if d.closed || d.masked || d.gateway == nil || !d.wanted[key] {
		d.mu.Unlock()
		return
	}
	current := d.cards[key]
	if current != nil && (current.version != p.Version || current.lang != d.appearance.Lang) {
		delete(d.cards, key)
	} else if current != nil {
		d.mu.Unlock()
		current.present(bounds, clip)
		return
	}
	gateway, origin, generation := d.gateway, d.origin, d.generation
	d.mu.Unlock()
	if current != nil {
		current.close()
	}
	next, err := d.newCard(key, p, gateway, origin, generation)
	if err != nil {
		d.Store.SetPanelError(p.ID, err.Error())
		return
	}
	d.mu.Lock()
	if d.gateway != gateway || d.closed {
		d.mu.Unlock()
		next.close()
		return
	}
	d.cards[key] = next
	d.mu.Unlock()
	next.present(bounds, clip)
}
func (d *Desktop) serve(w http.ResponseWriter, r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/__whale_plugin_ui/pad/") {
		return false
	}
	key := r.URL.Query().Get("card")
	d.mu.Lock()
	c := d.cards[key]
	d.mu.Unlock()
	if c == nil {
		http.NotFound(w, r)
		return true
	}
	switch r.URL.Path {
	case "/__whale_plugin_ui/pad/card":
		if r.Method != "GET" {
			w.WriteHeader(405)
			break
		}
		html, err := c.document()
		if err != nil {
			http.Error(w, err.Error(), 500)
			break
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		_, _ = w.Write([]byte(html))
	case "/__whale_plugin_ui/pad/events":
		c.serveEvents(w, r)
	default:
		http.NotFound(w, r)
	}
	return true
}
func (d *Desktop) active(c *card) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return !d.closed && !d.masked && d.cards[c.key] == c && d.generation == c.generation
}
func (d *Desktop) run(ctx context.Context, c *card, prompt, system string) store.RunResult {
	if !d.active(c) {
		return store.RunResult{Error: "Pad: Host changed"}
	}
	result, err := d.Store.RunForGeneration(ctx, c.generation, c.id, prompt, system)
	if err != nil {
		return store.RunResult{Error: err.Error()}
	}
	return result
}
func jsonString(v any) string { raw, _ := json.Marshal(v); return string(raw) }
