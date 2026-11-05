package pad

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/egoist/mygo"
	"whaleisle/internal/desktop/pluginslots"
	"whaleisle/internal/desktop/webview"
	store "whaleisle/internal/store/pad"
)

type card struct {
	mu             sync.Mutex
	desktop        *Desktop
	key, id, token string
	lang           string
	version        int
	generation     uint64
	host           *webview.Host
	gateway        *pluginslots.Gateway
	socket         *websocket.Conn
	queue          chan any
	cancel         context.CancelFunc
	closed         bool
	lastUpdates    map[string]string
}

func (d *Desktop) newCard(key string, p store.Panel, gateway *pluginslots.Gateway, origin string, generation uint64) (*card, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	d.mu.Lock()
	lang := d.appearance.Lang
	d.mu.Unlock()
	c := &card{desktop: d, key: key, id: p.ID, lang: lang, version: p.Version, generation: generation, token: hex.EncodeToString(b), gateway: gateway, queue: make(chan any, 8)}
	entry, err := gateway.EntryURL(origin + "/__whale_plugin_ui/pad/card?card=" + url.QueryEscape(key))
	if err != nil {
		return nil, err
	}
	// Register before creating the async page, so its first request sees the card.
	d.mu.Lock()
	if d.gateway != gateway || d.closed {
		d.mu.Unlock()
		return nil, errors.New("Pad: Host changed")
	}
	d.cards[key] = c
	d.mu.Unlock()
	h, err := webview.NewPluginSurface(webview.HostOptions{Parent: d.opts.Parent, URL: entry, Title: "Pad · " + p.Definition.Title, Untrusted: true, OnChange: d.opts.Changed, OnClosed: c.close, OnFailure: d.opts.OnFailure, PermissionPolicy: func(mygo.PermissionRequest) bool { return false }}, gateway)
	if err != nil {
		d.mu.Lock()
		if d.cards[key] == c {
			delete(d.cards, key)
		}
		d.mu.Unlock()
		return nil, err
	}
	c.mu.Lock()
	c.host = h
	c.mu.Unlock()
	return c, nil
}
func (c *card) present(bounds, clip mygo.Rectangle) {
	c.mu.Lock()
	h, closed := c.host, c.closed
	c.mu.Unlock()
	if h == nil || closed {
		return
	}
	if _, err := h.PresentPlugin(bounds, clip, nil, c.gateway); err != nil {
		c.desktop.Store.SetPanelError(c.id, err.Error())
	}
}
func (c *card) hide() {
	c.mu.Lock()
	h := c.host
	c.mu.Unlock()
	if h != nil {
		h.SetVisible(false)
	}
}
func (c *card) close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	h, socket, cancel := c.host, c.socket, c.cancel
	c.host = nil
	c.socket = nil
	c.cancel = nil
	c.mu.Unlock()
	c.desktop.mu.Lock()
	if c.desktop.cards[c.key] == c {
		delete(c.desktop.cards, c.key)
	}
	c.desktop.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if socket != nil {
		_ = socket.CloseNow()
	}
	if h != nil {
		_ = h.Close()
	}
}
func (c *card) send(v any) {
	c.mu.Lock()
	closed, connected := c.closed, c.socket != nil
	c.mu.Unlock()
	if closed || !connected {
		return
	}
	select {
	case c.queue <- v:
	default:
		c.desktop.Store.SetPanelError(c.id, "Pad: card message queue is full")
	}
}
func (c *card) update() {
	p, ok := c.desktop.Store.Panel(c.id)
	if !ok {
		c.close()
		return
	}
	c.desktop.mu.Lock()
	appearance := c.desktop.appearance
	c.desktop.mu.Unlock()
	c.sendUpdate("data", map[string]any{"kind": "data", "data": p.Data, "histories": p.History})
	c.sendUpdate("theme", map[string]any{"kind": "theme", "css": appearance.CSS, "mode": appearance.Mode})
	c.sendUpdate("height", map[string]any{"kind": "height", "auto": string(p.Definition.Height) == `"auto"` && c.desktop.Store.Snapshot().Layout.Heights[c.id] == 0})
}

// Native chrome/notice updates must not rerun the panel's onData handlers.
func (c *card) sendUpdate(kind string, value any) {
	encoded := jsonString(value)
	c.mu.Lock()
	if c.closed || c.socket == nil {
		c.mu.Unlock()
		return
	}
	if c.lastUpdates == nil {
		c.lastUpdates = map[string]string{}
	}
	if c.lastUpdates[kind] == encoded {
		c.mu.Unlock()
		return
	}
	c.lastUpdates[kind] = encoded
	c.mu.Unlock()
	c.send(value)
}

type cardMessage struct {
	Kind    string          `json:"kind"`
	ID      string          `json:"id"`
	Text    string          `json:"text"`
	URL     string          `json:"url"`
	Prompt  string          `json:"prompt"`
	System  string          `json:"system"`
	Key     string          `json:"key"`
	Value   json.RawMessage `json:"value"`
	Message string          `json:"message"`
	Height  float32         `json:"height"`
}

func (c *card) serveEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" || !c.desktop.active(c) {
		http.Error(w, "Pad: expired card", 409)
		return
	}
	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{"whaleisle.pad.v1"}})
	if err != nil {
		return
	}
	socket.SetReadLimit(512 << 10)
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer socket.CloseNow()
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	if c.socket != nil {
		_ = c.socket.CloseNow()
	}
	c.socket = socket
	c.cancel = cancel
	c.lastUpdates = map[string]string{}
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		if c.socket == socket {
			c.socket = nil
			c.cancel = nil
		}
		c.mu.Unlock()
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case value := <-c.queue:
				write, cancelWrite := context.WithTimeout(ctx, 3*time.Second)
				err := wsjson.Write(write, socket, value)
				cancelWrite()
				if err != nil {
					cancel()
					return
				}
			}
		}
	}()
	c.update()
	for {
		var m cardMessage
		if err = wsjson.Read(ctx, socket, &m); err != nil {
			return
		}
		if !c.desktop.active(c) {
			return
		}
		switch m.Kind {
		case "refresh":
			c.desktop.Store.Do(store.Action{Op: "refresh", PanelID: c.id, Generation: c.generation})
		case "ask":
			if len(m.Text) > 0 && len(m.Text) <= 16000 {
				c.desktop.Ask(m.Text, false)
			}
		case "open":
			u, e := url.Parse(m.URL)
			if e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil && c.desktop.opts.OnOpenURL != nil {
				if e = c.desktop.opts.OnOpenURL(u.String()); e != nil {
					c.desktop.Store.SetPanelError(c.id, e.Error())
				}
			}
		case "state":
			if err = c.desktop.Store.SetPanelStateForGeneration(c.generation, c.id, m.Key, m.Value); err != nil {
				c.desktop.Store.SetPanelError(c.id, err.Error())
			}
		case "run":
			if len(m.Prompt) == 0 || len(m.Prompt) > 96000 || len(m.System) > 32000 {
				continue
			}
			id, prompt, system := m.ID, m.Prompt, m.System
			go func() {
				result := c.desktop.run(ctx, c, prompt, system)
				c.send(map[string]any{"kind": "run-result", "id": id, "ok": result.OK, "text": result.Text, "error": result.Error})
			}()
		case "error":
			c.desktop.Store.SetPanelError(c.id, m.Message)
		case "ok":
			c.desktop.Store.SetPanelError(c.id, "")
		case "resize":
			if m.Height > 0 && m.Height <= 2000 {
				c.desktop.autoHeight(c.id, m.Height)
			}
		}
	}
}
func (d *Desktop) autoHeight(id string, height float32) {
	height = max(32, min(640, height))
	d.mu.Lock()
	if d.autoHeights == nil {
		d.autoHeights = map[string]float32{}
	}
	previous := d.autoHeights[id]
	d.autoHeights[id] = height
	d.mu.Unlock()
	if previous != height && d.opts.Changed != nil {
		d.opts.Changed()
	}
}
func (d *Desktop) AutoHeight(id string) float32 {
	d.mu.Lock()
	defer d.mu.Unlock()
	if n := d.autoHeights[id]; n > 0 {
		return n
	}
	return 96
}

var frameEndTag = regexp.MustCompile(`(?i)</(style|script)`)

func escapeFrameEndTag(source string) string {
	return frameEndTag.ReplaceAllString(source, `<\/$1`)
}

func (c *card) document() (string, error) {
	p, ok := c.desktop.Store.Panel(c.id)
	if !ok {
		return "", errors.New("Pad: panel removed")
	}
	c.desktop.mu.Lock()
	a, runtime := c.desktop.appearance, c.desktop.runtime
	c.desktop.mu.Unlock()
	init := map[string]any{"data": p.Data, "histories": p.History, "state": c.desktop.Store.PanelState(c.id), "lang": a.Lang}
	mode := "fixed"
	if string(p.Definition.Height) == `"auto"` && c.desktop.Store.Snapshot().Layout.Heights[c.id] == 0 {
		mode = "auto"
	}
	frame := `<!doctype html><html data-token="` + c.token + `" data-mode="` + a.Mode + `" data-height="` + mode + `"><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:; font-src data:; media-src data:"><style id="pad-theme">` + a.CSS + `</style><style>*,*::before,*::after{box-sizing:border-box}html,body{margin:0;padding:0;background:transparent;color:var(--pad-fg);font:13px/1.45 var(--pad-font)}body{padding:2px 0 4px}html[data-height=auto]{overflow:hidden}</style><style>` + escapeFrameEndTag(p.Definition.CSS) + `</style><script type="application/json" id="pad-init">` + jsonString(init) + `</script></head><body>` + p.Definition.HTML + `<script>` + runtime + `</script><script>` + escapeFrameEndTag(p.Definition.Script) + `</script></body></html>`
	return `<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; frame-src about:; script-src 'unsafe-inline'; style-src 'unsafe-inline'; connect-src 'self'"><style>html,body{margin:0;width:100%;height:100%;overflow:hidden;background:transparent}iframe{border:0;width:100%;height:100%;display:block}</style></head><body><iframe id="card" sandbox="allow-scripts" referrerpolicy="no-referrer"></iframe><script>(()=>{'use strict';const frame=document.getElementById('card'),token=` + jsonString(c.token) + `;let ready=false,queued=[];const socket=new WebSocket(location.href.replace(/^http/,'ws').replace('/card?','/events?'),'whaleisle.pad.v1');const post=m=>frame.contentWindow.postMessage({...m,token},'*');socket.onmessage=e=>{const m=JSON.parse(e.data);if(ready)post(m);else queued.push(m)};window.addEventListener('message',e=>{if(e.source!==frame.contentWindow||!e.data||e.data.token!==token)return;const m=e.data;if(m.kind==='ready'){ready=true;for(const v of queued)post(v);queued=[];return}if(m.kind==='run'){m.prompt=String(m.prompt||'').slice(0,24000);if(typeof m.system==='string')m.system=m.system.slice(0,8000)}if(m.kind==='ask')m.text=String(m.text||'').slice(0,4000);if(socket.readyState===1)socket.send(JSON.stringify(m))});frame.srcdoc=` + jsonString(frame) + `;})();</script></body></html>`, nil
}
