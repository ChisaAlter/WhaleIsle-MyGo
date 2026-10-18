package installctrl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	// EnvURL is passed to the harness child (loopback install base URL).
	EnvURL = "DSH_DESKTOP_INSTALL_URL"
	// EnvToken is the Bearer token paired with EnvURL.
	EnvToken = "DSH_DESKTOP_INSTALL_TOKEN"
	// DefaultRestartDelay matches the Electron desktop install control grace period.
	DefaultRestartDelay = 500 * time.Millisecond
	maxBodyBytes        = 64 * 1024
)

// InstallOpts are passed to InstallFunc for one install request.
type InstallOpts struct {
	AllowBuilds []string
}

// Result is the JSON body of POST /install.
type Result struct {
	OK               bool     `json:"ok"`
	NeedsAllowBuilds bool     `json:"needsAllowBuilds"`
	AllowBuilds      []string `json:"allowBuilds"`
	Spec             string   `json:"spec"`
	Error            string   `json:"error"`
	Log              string   `json:"log"`
	RolledBack       bool     `json:"rolledBack,omitempty"`
	RollbackError    string   `json:"rollbackError,omitempty"`
}

// InstallFunc performs the plugin install (may be stubbed in tests).
type InstallFunc func(spec string, opts InstallOpts) Result

// Options configure the loopback server.
type Options struct {
	HTMLPreview  http.Handler
	Project      func(context.Context, map[string]any) map[string]any
	Install      InstallFunc
	OnRestart    func()
	RestartDelay time.Duration
	// Desktop exposes the authenticated /desktop/* control plane used by the
	// whale assistant. A nil value keeps the legacy routes available and
	// reports unconfigured desktop operations explicitly.
	Desktop DesktopOps
}

// Server is a loopback HTTP install control plane.
type Server struct {
	opts Options

	mu           sync.Mutex
	http         *http.Server
	listener     net.Listener
	url          string
	token        string
	restartTimer *time.Timer
}

// New returns a stopped server.
func New(opts Options) *Server {
	if opts.RestartDelay <= 0 {
		opts.RestartDelay = DefaultRestartDelay
	}
	return &Server{opts: opts}
}

// Start binds 127.0.0.1:0 and serves until Stop. Idempotent while already running.
func (s *Server) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if s.listener != nil {
		s.mu.Unlock()
		return nil
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		s.mu.Unlock()
		return err
	}
	token := hex.EncodeToString(tokenBytes)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		s.mu.Unlock()
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	srv := &http.Server{
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	s.listener = ln
	s.token = token
	s.url = "http://" + ln.Addr().String()
	s.http = srv
	s.mu.Unlock()

	go func() { _ = srv.Serve(ln) }()
	return nil
}

// Stop closes the listener and clears active state.
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.restartTimer != nil {
		s.restartTimer.Stop()
		s.restartTimer = nil
	}
	if s.http != nil {
		_ = s.http.Close()
		s.http = nil
	}
	s.listener = nil
	s.url = ""
	s.token = ""
}

// Env returns DSH_DESKTOP_INSTALL_* entries for a harness child process.
func (s *Server) Env() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.url == "" || s.token == "" {
		return nil
	}
	return []string{EnvURL + "=" + s.url, EnvToken + "=" + s.token}
}

// URL and Token expose the active credentials (for tests).
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

func (s *Server) Token() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.token
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/install" && strings.HasPrefix(r.URL.Path, "/desktop/") == false {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
		return
	}
	s.mu.Lock()
	token := s.token
	s.mu.Unlock()
	if token == "" || r.Header.Get("Authorization") != "Bearer "+token {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "unauthorized"})
		return
	}
	// Readiness is authenticated with the same token as capture and never runs
	// a probe or enables settings as a side effect.
	if r.URL.Path == "/desktop/html-preview/status" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		if s.opts.HTMLPreview == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"available": false, "reason": "isolated HTML preview unavailable"})
			return
		}
		s.opts.HTMLPreview.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/desktop/html-preview" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		if s.opts.HTMLPreview == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "available": false, "error": "isolated HTML preview unavailable"})
			return
		}
		s.opts.HTMLPreview.ServeHTTP(w, r)
		return
	}
	if r.URL.Path == "/desktop/project" {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
			return
		}
		s.handleProject(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/desktop/") {
		s.handleDesktop(w, r)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": "not found"})
		return
	}
	s.handleInstall(w, r)
}

type installRequest struct {
	Spec        string   `json:"spec"`
	AllowBuilds []string `json:"allowBuilds"`
}

func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	var req installRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid json"})
			return
		}
	}
	spec := strings.TrimSpace(req.Spec)
	allowBuilds := NormalizeAllowBuilds(req.AllowBuilds)
	if !IsValidGithubSpec(spec) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "仅支持 github:owner/repo[#ref] 安装规格"})
		return
	}
	if allowBuilds == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "allowBuilds 包含非法包名"})
		return
	}
	install := s.opts.Install
	if install == nil {
		writeJSON(w, http.StatusInternalServerError, normalizeResult(Result{
			OK: false, Spec: spec, Error: "install handler not configured",
		}, spec))
		return
	}
	result := normalizeResult(install(spec, InstallOpts{AllowBuilds: allowBuilds}), spec)
	writeJSON(w, http.StatusOK, result)
	if result.OK && !result.NeedsAllowBuilds {
		s.scheduleRestart()
	}
}

func (s *Server) scheduleRestart() {
	onRestart := s.opts.OnRestart
	if onRestart == nil {
		return
	}
	delay := s.opts.RestartDelay
	s.mu.Lock()
	if s.restartTimer != nil {
		s.restartTimer.Stop()
	}
	s.restartTimer = time.AfterFunc(delay, func() {
		s.mu.Lock()
		s.restartTimer = nil
		s.mu.Unlock()
		onRestart()
	})
	s.mu.Unlock()
}

func normalizeResult(result Result, fallbackSpec string) Result {
	out := Result{
		OK:               result.OK,
		NeedsAllowBuilds: result.NeedsAllowBuilds,
		Spec:             result.Spec,
		Error:            result.Error,
		Log:              result.Log,
	}
	if out.Spec == "" {
		out.Spec = fallbackSpec
	}
	if result.AllowBuilds != nil {
		out.AllowBuilds = append([]string(nil), result.AllowBuilds...)
	} else {
		out.AllowBuilds = []string{}
	}
	return out
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	limited := io.LimitReader(r.Body, maxBodyBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) > maxBodyBytes {
		return nil, errors.New("request too large")
	}
	return data, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// StubOKInstall returns an InstallFunc that always succeeds (tests).
func StubOKInstall() InstallFunc {
	return func(spec string, _ InstallOpts) Result {
		return Result{OK: true, Spec: spec}
	}
}

// Project shares the existing loopback bearer boundary; no separate credential.
func (s *Server) handleProject(w http.ResponseWriter, r *http.Request) {
	body, err := readBody(r)
	var payload map[string]any
	if err != nil || json.Unmarshal(body, &payload) != nil || payload == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "invalid Project request"})
		return
	}
	if s.opts.Project == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "available": false, "error": "Project desktop bridge unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, s.opts.Project(r.Context(), payload))
}
