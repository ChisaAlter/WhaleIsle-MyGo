// Package schedule loads dsh-schedule list/catalog for the M9 native panel.
package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// Entry is one scheduled job row.
type Entry struct {
	ID          string
	Title       string
	Detail      string
	SessionID   string
	Origin      string
	Prompt      string
	Status      string
	ScheduledAt string          // Host-committed next instant, or final instant when inactive.
	TimeZone    string          // Rule zone; empty for absolute/interval rules.
	Raw         json.RawMessage // original list item for schedule/update expected
}

// ViewState is a UI-safe snapshot.
type ViewState struct {
	ConnectionEpoch uint64
	Origin          string
	ClientReady     bool
	Loading         bool
	CatalogReady    bool   // Successful all-session catalog, never a session-only fallback.
	CatalogSettled  bool   // Retained full-catalog facts, including during refresh/error.
	CatalogRevision uint64 // Advances only when that authoritative catalog settles.
	CatalogRequest  uint64
	Err             string
	Rows            []Entry
	CatalogHint     string
	SessionID       string
	LastAction      string
	Busy            bool
}

// Store fetches schedule/list for the selected session.
type Store struct {
	mu     sync.Mutex
	client *dsh.Client
	notify func()

	loading          bool
	err              error
	rows             []Entry
	catalogHint      string
	sessionID        string
	generation       uint64
	connectionEpoch  uint64
	eventsGeneration uint64
	receiptKeys      map[string]bool
	receiptReads     map[string]uint64
	requested        bool
	catalogReady     bool
	catalogRevision  uint64
	catalogRequest   uint64
	catalogSettled   uint64
	deleteSequence   uint64
	deleteNotice     *DeletionNotice
	deletePublished  bool
	lastAction       string
	busy             bool
}

// New returns an empty schedule store.
func New() *Store {
	return &Store{}
}

// Attach binds a dsh client.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	s.generation++
	if s.client != client {
		s.eventsGeneration++
		s.requested = false
		s.receiptKeys = nil
		s.receiptReads = nil
		s.connectionEpoch++
		s.loading = false
		s.busy = false
		s.rows = nil
		s.catalogHint = ""
		s.lastAction = ""
		s.catalogReady = false
		s.deleteNotice = nil
	}
	s.client = client
	s.notify = notify
	s.err = nil
	s.mu.Unlock()
}

// Detach clears the client.
func (s *Store) Detach() {
	s.mu.Lock()
	s.generation++
	s.connectionEpoch++
	s.eventsGeneration++
	s.requested = false
	s.receiptKeys, s.receiptReads = nil, nil
	s.client = nil
	s.loading = false
	s.busy = false
	s.rows = nil
	s.catalogHint = ""
	s.catalogReady = false
	s.deleteNotice = nil
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		ConnectionEpoch: s.connectionEpoch,
		ClientReady:     s.client != nil,
		Loading:         s.loading,
		CatalogReady:    s.catalogReady && !s.loading && s.err == nil,
		CatalogSettled:  s.catalogReady,
		CatalogRevision: s.catalogRevision,
		CatalogRequest:  s.catalogRequest,
		CatalogHint:     s.catalogHint,
		SessionID:       s.sessionID,
		LastAction:      s.lastAction,
		Busy:            s.busy,
	}
	if s.err != nil {
		st.Err = s.err.Error()
	}
	if s.client != nil {
		st.Origin = s.client.Endpoint.Origin
	}
	st.Rows = append([]Entry(nil), s.rows...)
	return st
}

// Refresh loads schedule/list (and optional catalog) for sessionID.
func (s *Store) Refresh(ctx context.Context, sessionID string) {
	s.refresh(ctx, sessionID, nil)
}

func (s *Store) refresh(ctx context.Context, sessionID string, owner *EventsOwner) {
	if ctx == nil || ctx.Err() != nil {
		return
	}
	s.mu.Lock()
	if owner != nil {
		if !s.requested || !s.eventsOwnerCurrentLocked(owner) {
			s.mu.Unlock()
			return
		}
		sessionID = s.sessionID
	}
	client := s.client
	if client == nil {
		s.mu.Unlock()
		return
	}
	hadCatalog := s.catalogReady
	s.generation++
	generation := s.generation
	s.catalogRequest++
	request := s.catalogRequest
	s.sessionID = sessionID
	s.requested = true
	s.loading = true
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		var catalog api.CatalogResult
		catalogHint := ""
		catalogErr := client.Call(ctx, "schedule/catalog", nil, &catalog)
		if catalogErr == nil {
			catalogHint = "全部会话 / All sessions · " + formatCatalogHint(catalog)
		}

		var rows []Entry
		var callErr error
		if catalogErr == nil {
			rows = parseScheduleList(catalog)
		} else if hadCatalog {
			// A transient catalog failure cannot replace known all-session
			// facts with a partial list from whichever Session is selected.
			callErr = catalogErr
		} else if sessionID == "" {
			callErr = fmt.Errorf("Host catalog unavailable; select a Session to list its active schedules")
		} else {
			catalogHint = "仅当前会话 / Current session only: " + sessionID + " · 全局目录不可用 / Host catalog unavailable"
			var list api.ListResult4
			req := map[string]any{"sessionId": sessionID}
			callErr = client.Call(ctx, "schedule/list", map[string]any{"request": req}, &list)
			if callErr == nil {
				rows = parseScheduleList(list)
			}
		}
		s.mu.Lock()
		if s.generation != generation || s.client != client {
			s.mu.Unlock()
			return
		}
		if ctx.Err() != nil || (owner != nil && !s.eventsOwnerCurrentLocked(owner)) {
			s.loading = false
			s.mu.Unlock()
			s.fire()
			return
		}
		for i := range rows {
			rows[i].Origin = client.Endpoint.Origin
			if catalogErr != nil {
				// schedule/list returns active records for the requested Session.
				rows[i].SessionID = sessionID
				rows[i].Status = "active"
			}
		}
		s.loading = false
		s.err = callErr
		if callErr == nil {
			s.rows = rows
			s.catalogHint = catalogHint
			s.catalogReady = catalogErr == nil
			if s.catalogReady {
				s.catalogRevision++
				s.catalogSettled = request
			}
		}
		s.mu.Unlock()
		s.fire()
	}()
}

// History loads schedule/history for one entry (Electron delivery history).
func (s *Store) History(ctx context.Context, sessionID, id string) (string, error) {
	id = trim(id)
	sessionID = trim(sessionID)
	if id == "" || sessionID == "" {
		return "", fmt.Errorf("missing schedule id")
	}
	s.mu.Lock()
	client := s.client
	epoch := s.connectionEpoch
	s.mu.Unlock()
	if client == nil {
		return "", fmt.Errorf("not connected")
	}
	req := api.ScheduleDeliveryHistoryRequest{
		Id: id, SessionId: sessionID, Limit: 20,
	}
	var out api.ScheduleDeliveryHistoryResult
	if err := client.Call(ctx, "schedule/history", map[string]any{"request": req}, &out); err != nil {
		return "", err
	}
	s.mu.Lock()
	current := s.client == client && s.connectionEpoch == epoch
	s.mu.Unlock()
	if !current {
		return "", fmt.Errorf("Host connection changed; reload task history")
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	if err := resultError(out); err != nil {
		return "", err
	}
	return string(raw), nil
}

// Delete removes one schedule by id in the given session.
func (s *Store) Delete(ctx context.Context, sessionID, id string) {
	go func() { _ = s.DeleteChecked(ctx, sessionID, id) }()
}

func (s *Store) finish(err error, rows []Entry, catalogHint string) {
	s.mu.Lock()
	s.loading = false
	s.err = err
	if err == nil {
		s.rows = rows
	}
	s.catalogHint = catalogHint
	s.mu.Unlock()
	s.fire()
}

func parseScheduleList(list api.ListResult4) []Entry {
	out := make([]Entry, 0, len(list))
	for _, raw := range list {
		var item struct {
			ID           string `json:"id"`
			Title        string `json:"title"`
			Prompt       string `json:"prompt"`
			Cron         string `json:"cron"`
			Status       string `json:"status"`
			SessionID    string `json:"sessionId"`
			Kind         string `json:"kind"`
			ScheduledAt  string `json:"scheduledAt"`
			Time         string `json:"time"`
			TimeZone     string `json:"timeZone"`
			Expression   string `json:"expression"`
			EverySeconds int    `json:"everySeconds"`
		}
		if json.Unmarshal(raw, &item) != nil {
			out = append(out, Entry{Title: "schedule", Detail: string(raw), Raw: append(json.RawMessage(nil), raw...)})
			continue
		}
		title := item.Title
		if title == "" {
			title = item.ID
		}
		if title == "" {
			title = "schedule"
		}
		detail := strings.Join([]string{item.Status, item.Kind, item.ScheduledAt, item.Time, item.TimeZone, item.Expression}, " · ")
		if item.EverySeconds > 0 {
			detail += fmt.Sprintf(" · %ds", item.EverySeconds)
		}
		if item.Cron != "" {
			if detail != "" {
				detail += " · "
			}
			detail += item.Cron
		}
		if item.Prompt != "" {
			if detail != "" {
				detail += " · "
			}
			detail += truncate(item.Prompt, 80)
		}
		out = append(out, Entry{ID: item.ID, Title: title, Detail: detail, SessionID: item.SessionID, Prompt: item.Prompt, Status: item.Status, ScheduledAt: item.ScheduledAt, TimeZone: item.TimeZone, Raw: append(json.RawMessage(nil), raw...)})
	}
	return out
}

// UpdateTitle patches title via schedule/update (expected = last list raw).
func (s *Store) UpdateTitle(ctx context.Context, sessionID, id, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("missing schedule id/title")
	}
	return s.updateFields(ctx, sessionID, id, &title, nil)
}

// UpdatePrompt patches prompt via schedule/update.
func (s *Store) UpdatePrompt(ctx context.Context, sessionID, id, prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("missing schedule id/prompt")
	}
	return s.updateFields(ctx, sessionID, id, nil, &prompt)
}

func (s *Store) updateFields(ctx context.Context, sessionID, id string, title, prompt *string) error {
	s.mu.Lock()
	var row Entry
	for _, r := range s.rows {
		if r.ID == id && (r.SessionID == sessionID || r.SessionID == "") {
			row = r
			break
		}
	}
	s.mu.Unlock()
	if row.ID == "" {
		return fmt.Errorf("schedule no longer loaded; refresh before editing")
	}
	if row.SessionID == "" {
		row.SessionID = sessionID
	}
	return s.Update(ctx, row, title, prompt, nil)
}

func formatCatalogHint(catalog api.CatalogResult) string {
	if len(catalog) == 0 {
		return "catalog: empty"
	}
	names := make([]string, 0, 4)
	for _, raw := range catalog {
		var item struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Name  string `json:"name"`
			Kind  string `json:"kind"`
		}
		_ = json.Unmarshal(raw, &item)
		label := item.Title
		if label == "" {
			label = item.Name
		}
		if label == "" {
			label = item.ID
		}
		if label == "" {
			label = item.Kind
		}
		if label == "" {
			continue
		}
		names = append(names, label)
		if len(names) >= 4 {
			break
		}
	}
	if len(names) == 0 {
		return fmt.Sprintf("catalog entries: %d", len(catalog))
	}
	more := ""
	if len(catalog) > len(names) {
		more = fmt.Sprintf(" (+%d)", len(catalog)-len(names))
	}
	return "catalog: " + strings.Join(names, ", ") + more
}

func trim(s string) string { return strings.TrimSpace(s) }

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func (s *Store) fire() {
	s.mu.Lock()
	fn := s.notify
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}
