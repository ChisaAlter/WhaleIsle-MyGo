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

	"whaleisle/internal/state"
	"whaleisle/internal/store/internal/storefs"
	"whaleisle/internal/store/storelocale"
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

// ViewState is a UI-safe snapshot. A published ViewState is immutable; the
// History slice belongs to the store and is replaced, never edited, on write.
type ViewState struct {
	Mode         Mode
	Busy         bool
	LastAction   string
	LastErr      *state.Fault
	SessionID    string // last known assistant / bot session from ensure/RPC
	History      []Turn
	DshbotID     string
	DshbotOrigin string
}

type modeDisk struct {
	Mode Mode `json:"mode"`
}

// Store holds mode + local history under root (userData/assistant). The view
// lives in a state.Published version; disk writes run outside mu under
// persistMu so the last queued write wins.
type Store struct {
	mu             sync.Mutex
	persistMu      sync.Mutex
	root           string
	notify         func()
	view           ViewState
	pub            state.Published[ViewState]
	acceptedPrompt string
}

// Dir returns userData/assistant.
func Dir(userData string) string {
	return filepath.Join(userData, "assistant")
}

// New returns an in-memory store with no persistence root.
func New() *Store {
	s := &Store{view: ViewState{Mode: ModeWhale}}
	s.pub.Publish(s.view)
	return s
}

// Open loads (or seeds) persisted assistant state under userData/assistant.
func Open(userData string) (*Store, error) {
	root := Dir(userData)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s := &Store{root: root, view: ViewState{Mode: ModeWhale}}
	if err := s.load(); err != nil {
		return nil, err
	}
	s.pub.Publish(s.view)
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

// Snapshot returns the published view; unchanged reads share one version.
func (s *Store) Snapshot() *state.Version[ViewState] {
	return s.pub.Load()
}

// SetMode updates the active native/WebView routing mode and persists it.
func (s *Store) SetMode(mode Mode) {
	mode = NormalizeMode(mode)
	s.mu.Lock()
	if s.view.Mode == mode {
		s.mu.Unlock()
		return
	}
	s.view.Mode = mode
	s.pub.Publish(s.view)
	s.mu.Unlock()
	s.fire()
	s.persistMode(mode)
}

// SetBusy marks an in-flight native turn (streamed into LastAction by callers).
func (s *Store) SetBusy(busy bool) {
	s.mu.Lock()
	s.view.Busy = busy
	s.pub.Publish(s.view)
	s.mu.Unlock()
	s.fire()
}

// SetLastAction records a status line and the classified fault behind it.
func (s *Store) SetLastAction(msg string, f *state.Fault) {
	s.mu.Lock()
	s.view.LastAction = strings.TrimSpace(msg)
	s.view.LastErr = f
	s.pub.Publish(s.view)
	s.mu.Unlock()
	s.fire()
}

// SetSessionID remembers the last ensure / open session id for 「打开会话」.
func (s *Store) SetSessionID(id string) {
	s.mu.Lock()
	s.view.SessionID = strings.TrimSpace(id)
	s.pub.Publish(s.view)
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
		s.view.SessionID = t.SessionID
	}
	s.view.History = append([]Turn{t}, s.view.History...)
	if len(s.view.History) > MaxHistory {
		s.view.History = s.view.History[:MaxHistory]
	}
	history := append([]Turn(nil), s.view.History...)
	s.pub.Publish(s.view)
	s.mu.Unlock()
	s.fire()
	s.persistHistory(history)
}

// notePersistFailure records a failed write in the field the panel renders,
// and clears that note once a later write of the same kind succeeds.
func (s *Store) notePersistFailure(err error, key string) {
	s.mu.Lock()
	if err != nil {
		s.view.LastErr = state.FaultText(storelocale.T(key))
	} else if s.view.LastErr != nil && s.view.LastErr.Text == storelocale.T(key) {
		s.view.LastErr = nil
	}
	s.pub.Publish(s.view)
	s.mu.Unlock()
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
	var d modeDisk
	if ok, err := storefs.ReadJSON("assistant", filepath.Join(s.root, modeFile), &d, false); err != nil {
		return err
	} else if ok {
		s.view.Mode = NormalizeMode(d.Mode)
	}
	var turns []Turn
	if ok, err := storefs.ReadJSON("assistant", filepath.Join(s.root, historyFile), &turns, false); err != nil {
		return err
	} else if ok {
		s.view.History = turns
		if len(s.view.History) > MaxHistory {
			s.view.History = s.view.History[:MaxHistory]
		}
		for _, t := range s.view.History {
			if sid := strings.TrimSpace(t.SessionID); sid != "" {
				s.view.SessionID = sid
				break
			}
		}
	}
	return nil
}

// persistMode serializes mode writes so a slower earlier write cannot
// overwrite a newer one; failures are reported through the view.
func (s *Store) persistMode(mode Mode) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	if s.root == "" {
		return
	}
	raw, err := json.MarshalIndent(modeDisk{Mode: mode}, "", "  ")
	if err == nil {
		err = storefs.WriteFileAtomic(filepath.Join(s.root, modeFile), raw, 0o644)
	}
	s.notePersistFailure(err, storelocale.KeyStoreAssistantModeSaveFailed)
}

// persistHistory serializes history writes; the slice passed in is a private
// copy taken before the lock was released.
func (s *Store) persistHistory(history []Turn) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	if s.root == "" {
		return
	}
	raw, err := json.MarshalIndent(history, "", "  ")
	if err == nil {
		err = storefs.WriteFileAtomic(filepath.Join(s.root, historyFile), raw, 0o644)
	}
	s.notePersistFailure(err, storelocale.KeyStoreAssistantHistorySaveFailed)
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

// SelectDshbot captures an explicit Bot/room choice for this Host. It is not
// persisted: a cached roster or the currently open chat must never choose a
// recipient after reconnect or restart.
func (s *Store) SelectDshbot(id, origin string) {
	s.mu.Lock()
	s.view.DshbotID = strings.TrimSpace(id)
	s.view.DshbotOrigin = strings.TrimSpace(origin)
	s.pub.Publish(s.view)
	s.mu.Unlock()
	s.fire()
}

// AcceptPrompt queues a one-shot acknowledgement. The UI clears only the exact
// draft accepted by the Host, preserving edits made while the request ran.
func (s *Store) AcceptPrompt(text string) {
	s.mu.Lock()
	s.acceptedPrompt = strings.TrimSpace(text)
	s.mu.Unlock()
	s.fire()
}

func (s *Store) ConsumeAcceptedDraft(draft string) string {
	s.mu.Lock()
	accepted := s.acceptedPrompt
	s.acceptedPrompt = ""
	s.mu.Unlock()
	if accepted != "" && strings.TrimSpace(draft) == accepted {
		return ""
	}
	return draft
}
