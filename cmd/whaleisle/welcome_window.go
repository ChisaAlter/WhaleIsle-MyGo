package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"whaleisle/internal/accountlogin"
	"whaleisle/internal/config"
	"whaleisle/internal/credentials"
	desktopwebview "whaleisle/internal/desktop/webview"
	"whaleisle/internal/dsh/api"
	"whaleisle/internal/paths"
	settingsstore "whaleisle/internal/store/settings"
	usagestore "whaleisle/internal/store/usage"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
	"whaleisle/internal/ui/welcome"
)

// welcomeMarker remembers that the first-run welcome window was already shown.
// The Electron gate reads the harness account/credential state over RPC, which
// needs the local service running; before boot the Go shell can only see
// credentials.json, so a completed or skipped welcome is persisted as a
// once-per-user-data-directory marker instead of re-prompting every launch.
const welcomeMarker = "welcome-shown"

// welcomeWindow owns the first-run welcome window (Electron welcome-window.js +
// renderer/welcome.html + welcome-backend.js): API key entry, account sign-in
// and skip. It is UI-thread owned; worker goroutines return through
// Window.Update, as the pet quick-chat card does.
type welcomeWindow struct {
	win    *mygo.Window
	parent *mygo.Window
	cfg    config.Config

	apiKey  string
	page    string
	note    string
	noteErr bool
	busy    bool

	signingIn bool
	sawActive bool

	done        bool
	bootStarted bool
	polling     bool
	stopPoll    chan struct{}
}

// welcomeRequired gates the window: a profile without any stored credential
// sees it once. Isolated/QA desktops keep their own startup path.
func (s *mainState) welcomeRequired() bool {
	if isolatedDesktopRun() {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.credsMissing && !s.welcomeDone
}

// openWelcomeWindow shows the welcome window and holds the main window until
// the user completes or skips it, then runs the ordinary launcher/boot flow.
func (s *mainState) openWelcomeWindow(parent *mygo.Window, cfg config.Config) {
	s.mu.Lock()
	if s.welcomeWin != nil {
		s.mu.Unlock()
		return
	}
	w := &welcomeWindow{parent: parent, cfg: cfg, page: welcome.PageSignIn}
	s.welcomeWin = w
	s.mu.Unlock()
	if parent != nil && !parent.IsDestroyed() {
		parent.Hide()
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:                    i18n.T(i18n.KeyAppName),
		Width:                    600,
		Height:                   700,
		UseContentSize:           true,
		TitleBarStyle:            mygo.TitleBarHidden,
		TitleBarHeight:           36,
		WindowsCaptionButtons:    &mygo.WindowsCaptionButtonOptions{Width: 32, Height: 32, Top: 8, Right: 8, CornerRadius: 8, GlyphSize: 12},
		WindowsFrameCornerRadius: 20,
		Transparent:              true,
		BackgroundColor:          theme.Resolve(cfg.Theme).BackgroundColor(),
		DisableResize:            true,
		DisableMaximize:          true,
		DisableMinimize:          true,
		// The native welcome window is frame:false (welcome-window.js:36), so no
		// OS menu bar is visible there either.
		AutoHideMenuBar: true,
		Content:         ui.View(w.view),
	})
	w.win = win
	if win == nil || win.IsDestroyed() {
		// Without the window the ordinary boot must still happen exactly once.
		s.mu.Lock()
		s.welcomeWin = nil
		s.mu.Unlock()
		w.showParent()
		w.beginBoot()
		return
	}
	// Native window.js:185 gives the Windows window a 48 DIP icon.
	applyWindowIcon(win)
	win.OnClosed(func() { w.finish() })
	w.startPoll()
	win.Focus()
}

func (w *welcomeWindow) view(c *ui.Context) {
	welcome.View(c, w.model())
}

func (w *welcomeWindow) model() welcome.Model {
	s := &state
	s.mu.Lock()
	page, note, noteErr, busy := w.page, w.note, w.noteErr, w.busy
	themeID := s.themeID
	usageStore := s.usage
	s.mu.Unlock()
	usage := usagestore.ViewState{}
	if usageStore != nil {
		usage = usageStore.Snapshot()
	}
	login := s.accountLogin.Snapshot()
	active := login.Busy || accountlogin.Active(usage.AttemptPhase)
	authURL := strings.TrimSpace(login.URL)
	if authURL == "" {
		authURL = strings.TrimSpace(usage.AuthorizeURL)
	}
	m := welcome.Model{
		ThemeID:      themeID,
		SystemDark:   mygo.Theme.IsDark(),
		Page:         page,
		APIKey:       &w.apiKey,
		Busy:         busy,
		SignInActive: active,
		AuthURL:      authURL,
		Note:         note,
		NoteError:    noteErr,
		OnSignIn:     w.signIn,
		OnShowKeyForm: func() {
			w.page = welcome.PageKey
			w.note, w.noteErr = "", false
			w.refresh()
		},
		OnBack: func() {
			w.page = welcome.PageSignIn
			w.note, w.noteErr = "", false
			w.refresh()
		},
		OnSaveAPIKey:   w.saveAPIKey,
		OnCopyLink:     w.copyLink,
		OnCancelSignIn: w.cancelSignIn,
		OnSkip:         func() { w.finish() },
	}
	// Live account state overrides any local note, mirroring the Electron
	// welcome auth page (waiting → exchanging → done).
	switch {
	case usage.AccountStatus == "credential-stored":
		m.Note, m.NoteError = i18n.T(i18n.KeyWelcomeSignInDone), false
	case active:
		switch usage.AttemptPhase {
		case "initializing", "exchanging", "committing":
			m.Note = i18n.T(i18n.KeyWelcomeSignInExchanging)
		default:
			m.Note = i18n.T(i18n.KeyWelcomeSignInWaiting)
			m.Hint = i18n.T(i18n.KeyWelcomeSignInWaitingHint)
		}
	}
	return m
}

func (w *welcomeWindow) refresh() {
	if w.win == nil || w.win.IsDestroyed() {
		return
	}
	w.win.Update(func() {})
}

func (w *welcomeWindow) showParent() {
	if w.parent == nil || w.parent.IsDestroyed() {
		return
	}
	// When the launcher owns its own window, the welcome flow hands over to it
	// (launcher_window.go); the main window stays hidden until the start.
	if state.launcherStageHasWindow() {
		return
	}
	w.parent.Show()
	w.parent.Restore()
	w.parent.Focus()
}

// beginBoot runs the launcher/boot flow at most once, whether the welcome
// window completed, was skipped, or never opened.
func (w *welcomeWindow) beginBoot() {
	if w.bootStarted {
		return
	}
	w.bootStarted = true
	state.beginLauncherBoot(w.cfg, w.parent)
}

// finish closes the window and continues the ordinary startup. Closing the
// window with its caption button counts as skip.
func (w *welcomeWindow) finish() {
	if w.done {
		return
	}
	w.done = true
	w.stopPolling()
	s := &state
	s.mu.Lock()
	if s.welcomeWin == w {
		s.welcomeWin = nil
	}
	quitting := s.quitting
	s.mu.Unlock()
	writeWelcomeMarker(paths.UserDataDir())
	if !quitting {
		w.showParent()
		w.beginBoot()
	}
	if w.win != nil && !w.win.IsDestroyed() {
		w.win.Destroy()
	}
}

// saveAPIKey stores the draft in credentials.json, preserving the other
// secrets, and mirrors it into the harness credential store when the local
// service is already reachable (settings credentials page semantics).
func (w *welcomeWindow) saveAPIKey() {
	key := strings.TrimSpace(w.apiKey)
	if key == "" {
		w.note, w.noteErr = i18n.T(i18n.KeyWelcomeKeyBlank), true
		w.refresh()
		return
	}
	// Electron gates the key twice: welcome-backend.save takes printable ASCII
	// only and the renderer additionally refuses `KEY=value` environment
	// assignments and quote/backtick-wrapped values.
	if !welcomeKeyShape(key) {
		w.note, w.noteErr = i18n.T(i18n.KeyWelcomeKeyInvalid), true
		w.refresh()
		return
	}
	if w.busy {
		return
	}
	w.busy = true
	w.note, w.noteErr = "", false
	w.refresh()
	win := w.win
	go func() {
		dir := paths.UserDataDir()
		existing, err := credentials.Load(dir)
		if err != nil {
			err = fmt.Errorf("%s: %w", resumeText("无法读取现有凭据", "Could not read the existing credentials"), err)
		}
		if err == nil {
			existing.APIKey = key
			err = credentials.Save(dir, existing)
		}
		if win == nil || win.IsDestroyed() {
			return
		}
		win.Update(func() {
			w.busy = false
			if err != nil {
				w.note, w.noteErr = i18n.T(i18n.KeyWelcomeKeyFailed)+": "+err.Error(), true
				return
			}
			s := &state
			s.mu.Lock()
			s.credsMissing = false
			s.mu.Unlock()
			s.dshMu.Lock()
			client := s.dshClient
			s.dshMu.Unlock()
			if client != nil {
				go func() {
					if syncErr := settingsstore.SyncAPIKeyToHarness(context.Background(), client, key); syncErr != nil {
						log.Printf("welcome: credentials/set: %v", syncErr)
					}
				}()
			}
			w.finish()
		})
	}()
}

// signIn reuses the desktop account login RPC. The Electron welcome backend
// posts to the harness origin, so without a reachable local service the boot
// starts first and the user presses sign-in again once it is ready.
func (w *welcomeWindow) signIn() {
	s := &state
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	if client == nil {
		if strings.TrimSpace(w.cfg.Workspace) == "" {
			w.note, w.noteErr = i18n.T(i18n.KeyWelcomeNoWorkspace), false
			w.refresh()
			return
		}
		w.beginBoot()
		w.note, w.noteErr = i18n.T(i18n.KeyWelcomeServiceStarting), false
		w.refresh()
		return
	}
	if !s.accountLogin.Begin() {
		return
	}
	w.signingIn, w.sawActive = true, false
	w.note, w.noteErr = i18n.T(i18n.KeyWelcomeSignInStarting), false
	w.refresh()
	go func() {
		_, zoneOff := time.Now().Zone()
		clientMeta := api.AccountClientMetadata{
			Version:               "whaleisle",
			Locale:                "zh-CN",
			TimezoneOffsetSeconds: float64(zoneOff),
		}
		var view api.AccountView
		err := client.Call(context.Background(), "account/startSignIn", map[string]any{
			"client":         clientMeta,
			"callbackOrigin": "whaleisle://auth",
			"loginSource":    "desktop",
		}, &view)
		s.accountLogin.Started(view, err, desktopwebview.OpenExternal)
		s.mu.Lock()
		usage, sessions := s.usage, s.sessions
		s.mu.Unlock()
		if usage != nil {
			sid := ""
			if sessions != nil {
				sid = sessions.Snapshot().SelectedID
			}
			usage.Refresh(context.Background(), sid)
		}
		w.refresh()
	}()
}

func (w *welcomeWindow) copyLink() {
	url := w.authorizeURL()
	if url == "" {
		return
	}
	mygo.Clipboard.WriteText(url)
	w.note, w.noteErr = i18n.T(i18n.KeyWelcomeCopied), false
	w.refresh()
}

func (w *welcomeWindow) cancelSignIn() {
	s := &state
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	usage, sessions := s.usage, s.sessions
	s.mu.Unlock()
	if client != nil && usage != nil {
		if attemptID := strings.TrimSpace(usage.Snapshot().AttemptID); attemptID != "" {
			s.cancelAccountSignIn(client, attemptID, usage, sessions)
		}
	}
	w.signingIn, w.sawActive = false, false
	w.note, w.noteErr = "", false
	w.refresh()
}

func (w *welcomeWindow) authorizeURL() string {
	s := &state
	if url := strings.TrimSpace(s.accountLogin.Snapshot().URL); url != "" {
		return url
	}
	s.mu.Lock()
	usage := s.usage
	s.mu.Unlock()
	if usage != nil {
		return strings.TrimSpace(usage.Snapshot().AuthorizeURL)
	}
	return ""
}

// poll follows the account state once a second, as the Electron welcome
// renderer followed account state events: a stored credential enters the
// workspace, and an attempt that ends without one reports expiry or failure.
func (w *welcomeWindow) poll() {
	if w.done {
		return
	}
	s := &state
	s.dshMu.Lock()
	client := s.dshClient
	s.dshMu.Unlock()
	s.mu.Lock()
	usage := s.usage
	s.mu.Unlock()
	if client != nil && !w.signingIn && w.note == i18n.T(i18n.KeyWelcomeServiceStarting) {
		// The local service came up while the window waited; sign-in works now.
		w.note, w.noteErr = "", false
	}
	if usage == nil {
		return
	}
	snap := usage.Snapshot()
	if snap.AccountStatus == "credential-stored" {
		w.finish()
		return
	}
	if !w.signingIn {
		return
	}
	login := s.accountLogin.Snapshot()
	active := login.Busy || accountlogin.Active(snap.AttemptPhase)
	switch {
	case login.Error != "":
		w.signingIn, w.sawActive = false, false
		w.note, w.noteErr = login.Error, true
	case w.sawActive && !active:
		w.signingIn, w.sawActive = false, false
		w.note, w.noteErr = i18n.T(i18n.KeyWelcomeSignInExpired), true
	case active:
		w.sawActive = true
	}
}

func (w *welcomeWindow) startPoll() {
	if w.polling || w.win == nil {
		return
	}
	w.polling = true
	stop := make(chan struct{})
	w.stopPoll = stop
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				win := w.win
				if win == nil || win.IsDestroyed() {
					return
				}
				win.Update(func() { w.poll() })
			}
		}
	}()
}

func (w *welcomeWindow) stopPolling() {
	if w.stopPoll != nil {
		close(w.stopPoll)
		w.stopPoll = nil
	}
	w.polling = false
}

// welcomeKeyShape mirrors the Electron welcome key gate. The main-process
// backend only checks /^[\x21-\x7e]+$/ (reference/electron/src/main/
// welcome-backend.js:104: printable ASCII, no spaces), but the renderer that
// actually submits the form rejects that same shape plus two paste artifacts
// (reference/electron/src/renderer/welcome/welcome.js:40): an environment
// assignment (`KEY=value`) and a value still wrapped in quotes or backticks
// (`"sk-..."`, `'sk-...'`, "`sk-...`"). A key is accepted only when all three
// renderer conditions hold.
func welcomeKeyShape(key string) bool {
	if key == "" {
		return false
	}
	for _, r := range key {
		if r < 0x21 || r > 0x7e {
			return false
		}
	}
	if welcomeKeyAssignment(key) {
		return false
	}
	// The renderer only strips the wrapping when the closing character equals
	// the opening one, so `"sk-abc' stays a (weird but legal) printable key.
	if first, last := key[0], key[len(key)-1]; (first == '"' || first == '\'' || first == '`') && last == first {
		return false
	}
	return true
}

// welcomeKeyAssignment reports the renderer's `/^[A-Z][A-Z0-9_]*=[^=]/` shape:
// an upper-case variable name, an `=` and a non-empty value that does not start
// with another `=`.
func welcomeKeyAssignment(key string) bool {
	if first := key[0]; first < 'A' || first > 'Z' {
		return false
	}
	for i := 1; i < len(key); i++ {
		switch c := key[i]; {
		case c == '=':
			return i+1 < len(key) && key[i+1] != '='
		case c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
		default:
			return false
		}
	}
	return false
}

// writeWelcomeMarker records that the welcome window was already shown, so a
// profile that skipped or completed it is never gated again. Failures are
// logged only: the marker is a convenience, never a startup condition.
func writeWelcomeMarker(dir string) {
	path := filepath.Join(dir, welcomeMarker)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		log.Println("welcome marker:", err)
		return
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if !errors.Is(err, os.ErrExist) {
			log.Println("welcome marker:", err)
		}
		return
	}
	_ = f.Close()
}
