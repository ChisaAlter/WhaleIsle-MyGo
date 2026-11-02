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
	"whaleisle/internal/store/storelocale"
)

//go:embed assets/pet-head.png assets/pet.html assets/tha4.js assets/anime4k.js assets/anime4k.LICENSE assets/anime4k-provenance.json assets/pet-super-resolution.js assets/pet-runtime.js assets/pet-body.js assets/pet-sleep.js assets/pet-layout.js assets/pet-interaction.js assets/pet-behavior.js assets/pet-dialogue.json assets/render-manifest.json
var assets embed.FS

// Companion is owned by the UI thread. Use Window.Update to call it from a
// harness callback. Only ServeHTTP runs concurrently, using an immutable sheet.
type Companion struct {
	Store                     *config.Store
	Enabled                   bool
	SkinID                    string
	Neural                    bool
	Behavior                  config.PetBehavior
	PositionDraft             *config.PetPosition
	OriginalImportReport      []string
	Error                     string
	Skins                     []Skin
	ready                     bool
	win                       *mygo.Window
	placingOverlay            bool
	stage                     mygo.Rectangle
	layout                    petLayout
	assetMu                   sync.RWMutex
	sheet                     []byte
	renderAssets              map[string][]byte
	neuralActive              bool
	renderNotice              string
	loadRetried               bool
	loaded                    bool
	notifications             notificationState
	OnNotificationsClosed     func()
	OnNotificationsReady      func()
	OnOpenNotificationSession func(string, string)
	task                      TaskStatus
	chat                      chatEntry
	look                      lookEntry
	growth                    chatEntry
	growthClock               restClock
	growthAutomatic           bool
	motion                    motionState
	interaction               interactionState
	careAction                careAnimation
	rest                      restState
	restClock                 restClock
	OnRefreshGrowth           func()
	OnAutoRefreshGrowth       func()
	OnFeedTokens              func()
	OnPersonality             func(string)
	feeding                   bool
	feedMessage               string
	OnLook                    func()
	OnOpenSettings            func()
	OnOpenChat                func()
	OnQuickChat               func()
	QuickChat                 *QuickChat
	quickChatOpen             bool
	OnOpenSession             func(string)
	OnEnabledChanged          func()
}

func New(store *config.Store) *Companion {
	cfg := store.DesktopPet()
	p := &Companion{Store: store, Enabled: cfg.Enabled, SkinID: cfg.SkinID, Neural: cfg.Neural, Behavior: cfg.Behavior, ready: true}
	if err := RegisterProtocol(); err != nil {
		p.Error = err.Error()
	}
	// The scheme is registered at package init (protocol.go); New only
	// publishes the companion that the stable dispatcher forwards to.
	setCurrent(p)
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
	positionImported := p.PositionDraft != nil
	if p.SkinID != "" {
		found := false
		for _, s := range p.Skins {
			if s.ID == p.SkinID {
				found = true
			}
		}
		// Keep an already-saved missing skin, but don't allow arbitrary new IDs.
		if !found && p.SkinID != previous.SkinID {
			p.Error = storelocale.T(storelocale.KeyDesktopPetSkinUnavailable)
			return
		}
	}
	p.Behavior = config.NormalizePetBehavior(p.Behavior)
	if err := p.Store.SaveDesktopPet(config.DesktopPet{Enabled: p.Enabled, SkinID: p.SkinID, Neural: p.Neural, Behavior: p.Behavior, PersonalityConfigured: true, Position: p.PositionForSave(previous.Position)}); err != nil {
		p.Error = err.Error()
		return
	}
	p.Error = ""
	p.PositionDraft = nil
	p.OriginalImportReport = nil
	p.cancelChangedLook(previous.Behavior, p.Behavior)
	if !p.Behavior.ChatEnabled {
		p.assetMu.Lock()
		p.chat.close()
		p.assetMu.Unlock()
	}
	if previous.Enabled != p.Enabled && p.OnEnabledChanged != nil {
		p.OnEnabledChanged()
	}
	if previous.Enabled != p.Enabled || previous.SkinID != p.SkinID || previous.Neural != p.Neural || positionImported {
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
	p.stopGrowthClock()
	p.stopRestClock()
	p.clearNotifications()
	p.CloseQuickChat()
	p.stopMotion()
	p.stopInteraction()
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
		petDebugStack("close: destroy window")
		w := p.win
		p.win = nil
		w.Destroy()
	}
}
func (p *Companion) sync() {
	cfg := p.Store.DesktopPet()
	petDebug("sync enabled=%v ready=%v win=%v", cfg.Enabled, p.ready, p.win != nil)
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
		neuralAssets, err = loadRenderAssets(petRenderRoot())
		if err != nil {
			p.Error = err.Error()
			notice = storelocale.T(storelocale.KeyDesktopPetMissingOrChangedTha4AssetsFallback)
		}
	}
	p.assetMu.Lock()
	p.renderAssets = neuralAssets
	p.neuralActive = cfg.Neural && neuralAssets != nil
	p.renderNotice = notice
	p.sheet = sheet
	p.assetMu.Unlock()
	p.preparePetStage(cfg.Position)
	w := openCompanionWindow(companionWindowOptions(petDisplays(), p.petBounds()))
	petDebug("newWindow nil=%v destroyed=%v", w == nil, w == nil || w.IsDestroyed())
	if w == nil || w.IsDestroyed() {
		p.Error = storelocale.T(storelocale.KeyDesktopPetWindowCreateFailed)
		p.close()
		return
	}
	p.win = w
	p.resetHitRegions()
	// The overlay stays fixed; only the character's deliberate anchor is saved.
	// Native default placement is the bottom-right of the work area with
	// EDGE_GAP (desktop-live2d.js:82-88); a remembered drag anchor wins, clamped
	// onto the work area so a monitor change cannot lose the pet.
	p.placePetWindow(w, cfg.Position)
	// A hidden WebView2 controller aborts a custom-scheme navigation
	// (COREWEBVIEW2_WEB_ERROR_STATUS 14), so the window must be shown before the
	// page starts loading; it paints transparently until the page renders.
	petDebug("showing before load")
	w.ShowInactive()
	p.startMotion(w)
	p.watchRecovery(w)
	w.Page().OnWillNavigate(func(e *mygo.NavigateEvent) {
		petDebug("willNavigate %s", e.URL)
		if e.URL == "whale-pet://localhost/index.html" {
			p.stopGrowthClock()
			return
		}
		e.PreventDefault()
		if p.win != w {
			return
		}
		p.assetMu.RLock()
		task := p.task
		p.assetMu.RUnlock()
		if p.fileDropAction(e.URL) {
			return
		}
		if p.interactionAction(e.URL) {
			return
		}
		if p.notificationAction(e.URL) {
			return
		}
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
			if p.OnFeedTokens != nil {
				p.OnFeedTokens()
			}
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
			p.startRestClock()
			p.startGrowthClock()
			p.startInteraction()
			// The window was already shown before loading (a hidden WebView2
			// controller aborts the custom-scheme navigation); this keeps it
			// visible after the page is up and marks the load as successful.
			petDebug("didFinishLoad: showing the companion")
			p.loaded = true
			w.ShowInactive()
		}
	})
	w.Page().OnDidFailLoad(func(err *mygo.LoadError) {
		if p.win != w {
			return
		}
		petDebug("didFailLoad url=%s code=%d desc=%s", err.URL, err.Code, err.Description)
		if p.loaded {
			// The page already rendered: a later aborted navigation (a reload
			// racing window activation, say) must not kill the pet.
			petDebug("didFailLoad: ignored, the page already loaded")
			return
		}
		if !p.loadRetried {
			// Diagnostic fallback: mygo serves custom schemes from
			// http://<scheme>.localhost/ on Windows. If the custom form aborted,
			// the mapped form distinguishes a missing request filter from a real
			// page failure before the window is closed.
			p.loadRetried = true
			petDebug("didFailLoad: loading %s", mappedPetURL)
			_ = w.Page().LoadURL(mappedPetURL)
			return
		}
		p.Error = err.Error()
		p.close()
	})
	w.OnHide(func() {
		if p.win == w {
			p.stopInteraction()
			p.stopGrowthClock()
			p.stopRestClock()
		}
	})
	w.OnShow(func() {
		if p.win == w {
			p.startInteraction()
			p.startRestClock()
			p.startGrowthClock()
		}
	})
	w.OnReadyToShow(func() {
		if p.win == w && p.ready {
			w.ShowInactive()
			if p.OnNotificationsReady != nil {
				p.OnNotificationsReady()
			}
		}
	})
	// An OS close hides only this instance; the persisted opt-in is unchanged.
	// Explicit disable belongs to the main settings page, including on quit.
	w.OnClosed(func() {
		if p.win == w {
			p.stopGrowthClock()
			p.stopRestClock()
			p.clearNotifications()
			p.CloseQuickChat()
			p.stopMotion()
			p.stopInteraction()
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
	if serveCareAnimation(w, r) {
		return
	}
	if serveFileDrop(w, r) {
		return
	}
	if serveStatusCard(w, r) {
		return
	}
	if serveRigResource(w, r) {
		return
	}
	if p.serveInteraction(w, r) {
		return
	}
	if p.serveNotifications(w, r) {
		return
	}
	var body []byte
	switch r.URL.Path {
	case "/index.html":
		body, _ = assets.ReadFile("assets/pet.html")
		body = renderNotificationPage(body)
		p.assetMu.RLock()
		body = renderLayoutPage(body, p.layout)
		body = renderTaskPage(body, p.activityTaskLocked())
		body = renderLookPage(body, p.look.status)
		body = []byte(strings.NewReplacer("{{RIG}}", fmt.Sprint(len(p.sheet) == 0), "{{NEURAL}}", fmt.Sprint(p.neuralActive), "{{RENDER_NOTICE}}", html.EscapeString(p.renderNotice), "{{CHAT_STATUS}}", html.EscapeString(p.chat.status.Message)).Replace(string(body)))
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
			HitEpoch      uint64
			Layout        petLayout
		}{p.activityTaskLocked(), p.quickChatOpen, behavior, p.chat.status, p.look.status, care, growth, rest, p.interaction.view.Epoch, p.layout})
		p.assetMu.RUnlock()
		w.Header().Set("Content-Type", "application/json")
	case "/pet-dialogue.json":
		body, _ = assets.ReadFile("assets/pet-dialogue.json")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	case "/tha4.js", "/anime4k.js", "/pet-super-resolution.js", "/pet-runtime.js", "/pet-body.js", "/pet-sleep.js", "/pet-layout.js", "/pet-interaction.js", "/pet-behavior.js":
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

// mappedPetURL is the Windows WebView2 form of the pet page: mygo serves custom
// schemes from http://<scheme>.localhost/ (internal/windows/webview.go:26-33).
const mappedPetURL = "http://whale-pet.localhost/index.html"
