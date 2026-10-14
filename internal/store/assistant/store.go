// Package assistant persists native whale/dshbot quick-prompt history and mode
// under userData/assistant/ so the M10 panel works beyond session SendPrompt.
package assistant

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	historyFile = "history.json"
	modeFile    = "mode.json"
	// MaxHistory is the number of turns kept on disk and shown in the panel.
	MaxHistory = 24
)

// Mode selects native routing and the default WebView desktopPlugin slot.
type Mode string

const (
	ModeWhale  Mode = "whale"
	ModeDshbot Mode = "dshbot"
)

// Turn is one native prompt/response pair recorded by the assistant panel.
type Turn struct {
	ID        string `json:"id"`
	Mode      Mode   `json:"mode"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Err       string `json:"err,omitempty"`
	Source    string `json:"source,omitempty"` // rpc | session | local
	At        string `json:"at"`               // RFC3339
}

// ViewState is a UI-safe snapshot.
type ViewState struct {
	Mode       Mode
	Busy       bool
	LastAction string
	LastErr    string
	SessionID  string // last known assistant / bot session from ensure/RPC
	History    []Turn
}

type modeDisk struct {
	Mode Mode `json:"mode"`
}

// Store holds mode + local history under root (userData/assistant).
type Store struct {
	mu         sync.Mutex
	root       string
	notify     func()
	mode       Mode
	busy       bool
	lastAction string
	lastErr    string
	sessionID  string
	history    []Turn
}

// Dir returns userData/assistant.
func Dir(userData string) string {
	return filepath.Join(userData, "assistant")
}

// New returns an in-memory store with no persistence root.
func New() *Store {
	return &Store{mode: ModeWhale}
}

// Open loads (or seeds) persisted assistant state under userData/assistant.
func Open(userData string) (*Store, error) {
	root := Dir(userData)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root, mode: ModeWhale}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Root returns the persistence directory (may be empty for New()).
func (s *Store) Root() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.root
}

// Attach sets an optional UI refresh hook.
func (s *Store) Attach(notify func()) {
	s.mu.Lock()
	s.notify = notify
	s.mu.Unlock()
}

// Snapshot copies observable state for one frame.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		Mode:       s.mode,
		Busy:       s.busy,
		LastAction: s.lastAction,
		LastErr:    s.lastErr,
		SessionID:  s.sessionID,
	}
	st.History = append([]Turn(nil), s.history...)
	return st
}

// SetMode updates the active native/WebView routing mode and persists it.
func (s *Store) SetMode(mode Mode) {
	mode = NormalizeMode(mode)
	s.mu.Lock()
	if s.mode == mode {
		s.mu.Unlock()
		return
	}
	s.mode = mode
	_ = s.saveModeLocked()
	s.mu.Unlock()
	s.fire()
}

// SetBusy marks an in-flight native turn (streamed into LastAction by callers).
func (s *Store) SetBusy(busy bool) {
	s.mu.Lock()
	s.busy = busy
	s.mu.Unlock()
	s.fire()
}

// SetLastAction records a status line (and clears LastErr when err is empty).
func (s *Store) SetLastAction(msg, errMsg string) {
	s.mu.Lock()
	s.lastAction = strings.TrimSpace(msg)
	s.lastErr = strings.TrimSpace(errMsg)
	s.mu.Unlock()
	s.fire()
}

// SetSessionID remembers the last ensure / open session id for 「打开会话」.
func (s *Store) SetSessionID(id string) {
	s.mu.Lock()
	s.sessionID = strings.TrimSpace(id)
	s.mu.Unlock()
	s.fire()
}

// AppendTurn prepends a completed native turn and trims to MaxHistory.
func (s *Store) AppendTurn(t Turn) {
	s.mu.Lock()
	if strings.TrimSpace(t.ID) == "" {
		t.ID = newID()
	}
	if strings.TrimSpace(t.At) == "" {
		t.At = time.Now().UTC().Format(time.RFC3339)
	}
	t.Mode = NormalizeMode(t.Mode)
	t.Prompt = strings.TrimSpace(t.Prompt)
	t.Response = strings.TrimSpace(t.Response)
	t.Err = strings.TrimSpace(t.Err)
	t.SessionID = strings.TrimSpace(t.SessionID)
	if t.SessionID != "" {
		s.sessionID = t.SessionID
	}
	s.history = append([]Turn{t}, s.history...)
	if len(s.history) > MaxHistory {
		s.history = s.history[:MaxHistory]
	}
	_ = s.saveHistoryLocked()
	s.mu.Unlock()
	s.fire()
}

// PrefixForMode returns the prompt prefix used for session-routed native sends.
func PrefixForMode(mode Mode) string {
	switch NormalizeMode(mode) {
	case ModeDshbot:
		return "[dshbot] "
	default:
		return "[whale] "
	}
}

// NormalizeMode maps unknown values to ModeWhale.
func NormalizeMode(mode Mode) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case ModeDshbot:
		return ModeDshbot
	default:
		return ModeWhale
	}
}

// HistoryFromSessionMessages builds native-style turns from session chat lines
// (user/assistant text pairs), newest first, capped at limit.
func HistoryFromSessionMessages(messages []struct{ Role, Body string }, limit int) []Turn {
	if limit <= 0 {
		limit = MaxHistory
	}
	var pairs []Turn
	var pendingUser string
	for _, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		body := strings.TrimSpace(m.Body)
		if body == "" {
			continue
		}
		switch role {
		case "user":
			pendingUser = body
		case "assistant":
			if pendingUser == "" {
				continue
			}
			pairs = append(pairs, Turn{
				Prompt:   pendingUser,
				Response: body,
				Source:   "session",
			})
			pendingUser = ""
		}
	}
	// Newest last in scan → reverse for newest-first UI.
	for i, j := 0, len(pairs)-1; i < j; i, j = i+1, j-1 {
		pairs[i], pairs[j] = pairs[j], pairs[i]
	}
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	return pairs
}

func (s *Store) load() error {
	if raw, err := os.ReadFile(filepath.Join(s.root, modeFile)); err == nil {
		var d modeDisk
		if json.Unmarshal(raw, &d) == nil {
			s.mode = NormalizeMode(d.Mode)
		}
	}
	if raw, err := os.ReadFile(filepath.Join(s.root, historyFile)); err == nil {
		var turns []Turn
		if json.Unmarshal(raw, &turns) == nil {
			s.history = turns
			if len(s.history) > MaxHistory {
				s.history = s.history[:MaxHistory]
			}
			for _, t := range s.history {
				if sid := strings.TrimSpace(t.SessionID); sid != "" {
					s.sessionID = sid
					break
				}
			}
		}
	}
	return nil
}

func (s *Store) saveModeLocked() error {
	if s.root == "" {
		return nil
	}
	raw, err := json.MarshalIndent(modeDisk{Mode: s.mode}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.root, modeFile), raw, 0o644)
}

func (s *Store) saveHistoryLocked() error {
	if s.root == "" {
		return nil
	}
	raw, err := json.MarshalIndent(s.history, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.root, historyFile), raw, 0o644)
}

func (s *Store) fire() {
	s.mu.Lock()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}

func newID() string {
	return time.Now().UTC().Format("20060102T150405.000000000")
}
