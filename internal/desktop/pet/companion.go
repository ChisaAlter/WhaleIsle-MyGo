package pet

import (
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"sync"

	"github.com/egoist/mygo"
	"whaleisle/internal/config"
)

//go:embed assets/pet-head.png assets/pet.html assets/tha4.js assets/pet-runtime.js assets/render-manifest.json
var assets embed.FS

// Companion is owned by the UI thread. Use Window.Update to call it from a
// harness callback. Only ServeHTTP runs concurrently, using an immutable sheet.
type Companion struct {
	Store           *config.Store
	Enabled         bool
	SkinID          string
	Neural          bool
	Error           string
	Skins           []Skin
	ready           bool
	win             *mygo.Window
	assetMu         sync.RWMutex
	sheet           []byte
	renderAssets    map[string][]byte
	neuralActive    bool
	renderNotice    string
	task            TaskStatus
	chat            chatEntry
	growth          chatEntry
	OnRefreshGrowth func()
	feeding         bool
	feedMessage     string
	OnOpenChat      func()
	OnOpenSession   func(string)
}

func New(store *config.Store) *Companion {
	cfg := store.DesktopPet()
	p := &Companion{Store: store, Enabled: cfg.Enabled, SkinID: cfg.SkinID, Neural: cfg.Neural, ready: true}
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
	if err := p.Store.SaveDesktopPet(config.DesktopPet{Enabled: p.Enabled, SkinID: p.SkinID, Neural: p.Neural}); err != nil {
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
	p.assetMu.Lock()
	p.chat.close()
	p.growth.close()
	p.feeding = false
	p.feedMessage = ""
	p.renderAssets = nil
	p.neuralActive = false
	p.assetMu.Unlock()
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
	var neuralAssets map[string][]byte
	notice := ""
	if cfg.Neural {
		var err error
		neuralAssets, err = loadRenderAssets(defaultRenderRoot())
		if err != nil {
			p.Error = err.Error()
			notice = "THA4 资源不可用，已回退 / Missing or changed THA4 assets; fallback"
		}
	}
	p.assetMu.Lock()
	p.renderAssets = neuralAssets
	p.neuralActive = cfg.Neural && neuralAssets != nil
	p.renderNotice = notice
	p.sheet = sheet
	p.assetMu.Unlock()
	w := mygo.NewWindow(mygo.WindowOptions{Title: "Whale Isle companion", URL: "whale-pet://localhost/index.html", Width: 220, Height: 438, MinHeight: 438, Hidden: true, Frameless: true, Transparent: true, AlwaysOnTop: true, SkipTaskbar: true, DisableResize: true, DisableMaximize: true, DisableMinimize: true, DisableFullScreen: true, DisableShadow: true, StateKey: "mygo-desktop-pet", Page: mygo.PageOptions{WindowControlsOnly: true, DevTools: mygo.DevToolsDisabled}})
	if w == nil || w.IsDestroyed() {
		p.Error = "Companion window could not open / 桌宠窗口创建失败"
		p.close()
		return
	}
	p.win = w
	// StateKey restores older window sizes as well as position. Keep all action
	// links reachable when an existing installation restores the shorter card.
	w.SetSize(220, 438)
	w.Page().OnWillNavigate(func(e *mygo.NavigateEvent) {
		if e.URL == "whale-pet://localhost/index.html" {
			return
		}
		e.PreventDefault()
		if p.win != w {
			return
		}
		p.assetMu.RLock()
		task := p.task
		p.assetMu.RUnlock()
		switch taskAction(e.URL, task) {
		case "hide":
			p.close()
		case "refresh-growth":
			if p.OnRefreshGrowth != nil {
				p.OnRefreshGrowth()
			}
		case "feed-tokens":
			p.feedTokens()
		case "feed-file":
			p.feedFile()
		case "chat":
			if p.OnOpenChat != nil {
				p.OnOpenChat()
			}
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
			p.assetMu.Lock()
			p.chat.close()
			p.growth.close()
			p.feeding = false
			p.feedMessage = ""
			p.renderAssets = nil
			p.neuralActive = false
			p.assetMu.Unlock()
		}
	})
}
func (p *Companion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Host != "localhost" || r.URL.User != nil || r.URL.RawQuery != "" || r.URL.Fragment != "" || (r.Method != "GET" && r.Method != "HEAD") {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; script-src 'self' 'wasm-unsafe-eval'; connect-src 'self'; worker-src 'none'; frame-ancestors 'none'")
	var body []byte
	switch r.URL.Path {
	case "/index.html":
		body, _ = assets.ReadFile("assets/pet.html")
		p.assetMu.RLock()
		body = renderTaskPage(body, p.task)
		body = []byte(strings.NewReplacer("{{NEURAL}}", fmt.Sprint(p.neuralActive), "{{RENDER_NOTICE}}", html.EscapeString(p.renderNotice), "{{CHAT_STATUS}}", html.EscapeString(p.chat.status.Message)).Replace(string(body)))
		p.assetMu.RUnlock()
		body = renderCarePage(body, p.careStatus())
		body = renderGrowthPage(body, p.growthStatus())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case "/task.json":
		care := p.careStatus()
		growth := p.growthStatus()
		p.assetMu.RLock()
		body, _ = json.Marshal(struct {
			TaskStatus
			Chat   ChatStatus
			Care   CareStatus
			Growth GrowthStatus
		}{p.task, p.chat.status, care, growth})
		p.assetMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
	case "/tha4.js", "/pet-runtime.js":
		body, _ = assets.ReadFile("assets" + r.URL.Path)
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
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
		p.assetMu.RLock()
		body = p.renderAssets[r.URL.Path]
		p.assetMu.RUnlock()
		if len(body) == 0 {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", renderContentType(r.URL.Path))
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
	p.task = task
	p.assetMu.Unlock()
	// The fixed local status endpoint updates text without restarting inference.
}
