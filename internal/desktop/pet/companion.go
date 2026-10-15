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

//go:embed assets/pet-head.png assets/pet.html assets/tha4.js assets/pet-runtime.js assets/pet-behavior.js assets/pet-dialogue.json assets/render-manifest.json
var assets embed.FS

// Companion is owned by the UI thread. Use Window.Update to call it from a
// harness callback. Only ServeHTTP runs concurrently, using an immutable sheet.
type Companion struct {
	Store           *config.Store
	Enabled         bool
	SkinID          string
	Neural          bool
	Behavior        config.PetBehavior
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
	look            lookEntry
	growth          chatEntry
	motion          motionState
	rest            restState
	OnRefreshGrowth func()
	OnPersonality   func(string)
	feeding         bool
	feedMessage     string
	OnLook          func()
	OnOpenSettings  func()
	OnOpenChat      func()
	OnQuickChat     func()
	QuickChat       *QuickChat
	quickChatOpen   bool
	OnOpenSession   func(string)
}

func New(store *config.Store) *Companion {
	cfg := store.DesktopPet()
	p := &Companion{Store: store, Enabled: cfg.Enabled, SkinID: cfg.SkinID, Neural: cfg.Neural, Behavior: cfg.Behavior, ready: true}
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
	previous := p.Store.DesktopPet()
	if p.SkinID != "" {
		found := false
		for _, s := range p.Skins {
			if s.ID == p.SkinID {
				found = true
			}
		}
		// Keep an already-saved missing skin, but don't allow arbitrary new IDs.
		if !found && p.SkinID != previous.SkinID {
			p.Error = "Selected skin is unavailable / 皮肤不可用"
			return
		}
	}
	p.Behavior = config.NormalizePetBehavior(p.Behavior)
	if err := p.Store.SaveDesktopPet(config.DesktopPet{Enabled: p.Enabled, SkinID: p.SkinID, Neural: p.Neural, Behavior: p.Behavior, PersonalityConfigured: true, Position: previous.Position}); err != nil {
		p.Error = err.Error()
		return
	}
	p.Error = ""
	if previous.Enabled != p.Enabled || previous.SkinID != p.SkinID || previous.Neural != p.Neural {
		p.close()
	}
	p.sync()
	p.applyBehavior()
	p.CheckQuickChat()
	if p.OnPersonality != nil {
		p.OnPersonality(p.Behavior.Personality)
	}
}
func (p *Companion) SetReady(ready bool) { p.ready = ready; p.sync() }
func (p *Companion) Close()              { p.ready = false; p.close() }
func (p *Companion) close() {
	p.CloseQuickChat()
	p.stopMotion()
	p.assetMu.Lock()
	p.chat.close()
	p.look.close()
	p.growth.close()
	p.rest = restState{}
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
	w := mygo.NewWindow(mygo.WindowOptions{Title: "Whale Isle companion", URL: "whale-pet://localhost/index.html", Width: 220, Height: petWindowHeight(cfg.Behavior), MinHeight: 438, Hidden: true, Frameless: true, Transparent: true, AlwaysOnTop: true, SkipTaskbar: true, DisableResize: true, DisableMaximize: true, DisableMinimize: true, DisableFullScreen: true, DisableShadow: true, Page: mygo.PageOptions{WindowControlsOnly: true, DevTools: mygo.DevToolsDisabled}})
	if w == nil || w.IsDestroyed() {
		p.Error = "Companion window could not open / 桌宠窗口创建失败"
		p.close()
		return
	}
	p.win = w
	// Apply the footprint before restoring the deliberate drag anchor. Roaming
	// is transient and must not be saved by MyGo automatic window state.
	w.SetSize(220, petWindowHeight(cfg.Behavior))
	p.startMotion(w)
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
		action := taskAction(e.URL, task)
		if action != "" {
			p.motionContact()
		}
		switch action {
		case "motion-contact":
			// Contact only: no page-provided coordinates or native move API.
		case "hide":
			p.hide()
		case "look-screen":
			if p.OnLook != nil {
				p.OnLook()
			}
		case "settings":
			if p.OnOpenSettings != nil {
				p.OnOpenSettings()
			}
		case "refresh-growth":
			if p.OnRefreshGrowth != nil {
				p.OnRefreshGrowth()
			}
		case "feed-tokens":
			p.feedTokens()
		case "sleep-pet", "wake-pet", "wake-contact":
			p.restPet(taskAction(e.URL, task))
		case "pat-pet":
			p.patPet()
		case "feed-file":
			p.feedFile()
		case "quick-chat":
			if p.ChatAllowed() && p.OnQuickChat != nil {
				p.OnQuickChat()
			}
		case "chat":
			if p.ChatAllowed() && p.OnOpenChat != nil {
				p.OnOpenChat()
			}
		case "open":
			if p.OnOpenSession != nil {
				p.OnOpenSession(task.SessionID)
			}
		}
	})
	w.Page().SetWindowOpenHandler(func(mygo.WindowOpenRequest) *mygo.WindowOptions { return nil })
	// Original sleep lives in the page: a reload/recreation starts awake too.
	w.Page().OnDidFinishLoad(func() {
		if p.win == w {
			p.assetMu.Lock()
			p.rest = restState{}
			p.assetMu.Unlock()
		}
	})
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
			p.CloseQuickChat()
			p.stopMotion()
			p.win = nil
			p.assetMu.Lock()
			p.chat.close()
			p.look.close()
			p.growth.close()
			p.rest = restState{}
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
		body = renderLookPage(body, p.look.status)
		body = []byte(strings.NewReplacer("{{NEURAL}}", fmt.Sprint(p.neuralActive), "{{RENDER_NOTICE}}", html.EscapeString(p.renderNotice), "{{CHAT_STATUS}}", html.EscapeString(p.chat.status.Message)).Replace(string(body)))
		p.assetMu.RUnlock()
		body = renderBehaviorPage(body, p.savedBehavior())
		body = renderCarePage(body, p.careStatus())
		body = renderGrowthPage(body, p.growthStatus())
		body = renderRestPage(body, p.restStatus())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case "/task.json":
		care := p.careStatus()
		growth := p.growthStatus()
		rest := p.restStatus()
		behavior := p.savedBehavior()
		p.assetMu.RLock()
		body, _ = json.Marshal(struct {
			TaskStatus
			QuickChatOpen bool
			Behavior      config.PetBehavior
			Chat          ChatStatus
			Look          LookStatus
			Care          CareStatus
			Growth        GrowthStatus
			Rest          RestStatus
		}{p.task, p.quickChatOpen, behavior, p.chat.status, p.look.status, care, growth, rest})
		p.assetMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
	case "/pet-dialogue.json":
		body, _ = assets.ReadFile("assets/pet-dialogue.json")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case "/tha4.js", "/pet-runtime.js", "/pet-behavior.js":
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
	p.CheckQuickChat()
	// The fixed local status endpoint updates text without restarting inference.
}
