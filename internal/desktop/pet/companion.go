package pet

import (
	"embed"
	"fmt"
	"net/http"
	"sync"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

//go:embed assets/pet-head.png assets/pet.html
var assets embed.FS

// Companion is owned by the UI thread. Use Window.Update to call it from a
// harness callback. Only ServeHTTP runs concurrently, using an immutable sheet.
type Companion struct {
	Store         *config.Store
	Enabled       bool
	SkinID        string
	Error         string
	Skins         []Skin
	ready         bool
	win           *mygo.Window
	assetMu       sync.RWMutex
	sheet         []byte
	task          TaskStatus
	OnOpenSession func(string)
}

func New(store *config.Store) *Companion {
	cfg := store.DesktopPet()
	p := &Companion{Store: store, Enabled: cfg.Enabled, SkinID: cfg.SkinID, ready: true}
	if err := mygo.Protocol.Handle("whale-pet", p); err != nil {
		p.Error = err.Error()
	}
	return p
}
func (p *Companion) Refresh() {
	skins, err := Discover(DefaultRoot())
	if err != nil {
		p.Error = err.Error()
		return
	}
	p.Skins = skins
	p.Error = ""
}
func (p *Companion) Save() {
	if p.SkinID != "" {
		found := false
		for _, s := range p.Skins {
			if s.ID == p.SkinID {
				found = true
			}
		}
		// Keep an already-saved missing skin, but don't allow arbitrary new IDs.
		if !found && p.SkinID != p.Store.DesktopPet().SkinID {
			p.Error = "Selected skin is unavailable / 皮肤不可用"
			return
		}
	}
	if err := p.Store.SaveDesktopPet(config.DesktopPet{Enabled: p.Enabled, SkinID: p.SkinID}); err != nil {
		p.Error = err.Error()
		return
	}
	p.Error = ""
	p.close()
	p.sync()
}
func (p *Companion) SetReady(ready bool) { p.ready = ready; p.sync() }
func (p *Companion) Close()              { p.ready = false; p.close() }
func (p *Companion) close() {
	if p.win != nil {
		w := p.win
		p.win = nil
		w.Destroy()
	}
}
func (p *Companion) sync() {
	cfg := p.Store.DesktopPet()
	if !cfg.Enabled || !p.ready {
		p.close()
		return
	}
	if p.win != nil && !p.win.IsDestroyed() {
		return
	}
	p.Refresh()
	var sheet []byte
	for _, s := range p.Skins {
		if s.ID == cfg.SkinID {
			sheet = s.Sheet
			break
		}
	}
	p.assetMu.Lock()
	p.sheet = sheet
	p.assetMu.Unlock()
	w := mygo.NewWindow(mygo.WindowOptions{Title: "Whale Isle companion", URL: "whale-pet://pet/index.html", Width: 220, Height: 198, Hidden: true, Frameless: true, Transparent: true, AlwaysOnTop: true, SkipTaskbar: true, DisableResize: true, DisableMaximize: true, DisableMinimize: true, DisableFullScreen: true, DisableShadow: true, StateKey: "mygo-desktop-pet", Page: mygo.PageOptions{WindowControlsOnly: true, DevTools: mygo.DevToolsDisabled}})
	if w == nil || w.IsDestroyed() {
		p.Error = "Companion window could not open / 桌宠窗口创建失败"
		return
	}
	p.win = w
	w.Page().OnWillNavigate(func(e *mygo.NavigateEvent) {
		if e.URL == "whale-pet://pet/index.html" {
			return
		}
		e.PreventDefault()
		p.assetMu.RLock()
		task := p.task
		p.assetMu.RUnlock()
		switch taskAction(e.URL, task) {
		case "hide":
			p.close()
		case "open":
			if p.OnOpenSession != nil {
				p.OnOpenSession(task.SessionID)
			}
		}
	})
	w.Page().SetWindowOpenHandler(func(mygo.WindowOpenRequest) *mygo.WindowOptions { return nil })
	w.Page().OnDidFailLoad(func(err *mygo.LoadError) { p.Error = err.Error(); p.close() })
	w.OnReadyToShow(func() {
		if p.win == w && p.ready {
			w.ShowInactive()
		}
	})
	// An OS close hides only this instance; the persisted opt-in is unchanged.
	// Explicit disable belongs to the main settings page, including on quit.
	w.OnClosed(func() {
		if p.win == w {
			p.win = nil
		}
	})
}
func (p *Companion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Host != "pet" || (r.Method != "GET" && r.Method != "HEAD") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'none'; frame-ancestors 'none'")
	var body []byte
	switch r.URL.Path {
	case "/index.html":
		body, _ = assets.ReadFile("assets/pet.html")
		p.assetMu.RLock()
		body = renderTaskPage(body, p.task)
		p.assetMu.RUnlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case "/sprite.webp":
		p.assetMu.RLock()
		body = p.sheet
		p.assetMu.RUnlock()
		if len(body) == 0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/webp")
	case "/fallback.png":
		body, _ = assets.ReadFile("assets/pet-head.png")
		w.Header().Set("Content-Type", "image/png")
	default:
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	if r.Method != "HEAD" {
		_, _ = w.Write(body)
	}
}

// UpdateTask runs on the main UI thread. Hidden/closed pets remain hidden;
// saving settings is the explicit way to show them again.
func (p *Companion) UpdateTask(task TaskStatus) {
	p.assetMu.Lock()
	changed := p.task != task
	p.task = task
	p.assetMu.Unlock()
	if changed && p.win != nil && !p.win.IsDestroyed() {
		if err := p.win.Page().LoadURL("whale-pet://pet/index.html"); err != nil {
			p.Error = err.Error()
		}
	}
}
