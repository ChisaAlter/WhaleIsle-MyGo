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
	ID     string
	Title  string
	Detail string
	Raw    json.RawMessage // original list item for schedule/update expected
}

// ViewState is a UI-safe snapshot.
type ViewState struct {
	ClientReady bool
	Loading     bool
	Err         string
	Rows        []Entry
	CatalogHint string
}

// Store fetches schedule/list for the selected session.
type Store struct {
	mu     sync.Mutex
	client *dsh.Client
	notify func()

	loading     bool
	err         error
	rows        []Entry
	catalogHint string
}

// New returns an empty schedule store.
func New() *Store {
	return &Store{}
}

// Attach binds a dsh client.
func (s *Store) Attach(client *dsh.Client, notify func()) {
	s.mu.Lock()
	s.client = client
	s.notify = notify
	s.err = nil
	s.mu.Unlock()
}

// Detach clears the client.
func (s *Store) Detach() {
	s.mu.Lock()
	s.client = nil
	s.loading = false
	s.rows = nil
	s.catalogHint = ""
	s.mu.Unlock()
	s.fire()
}

// Snapshot copies observable state.
func (s *Store) Snapshot() ViewState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := ViewState{
		ClientReady: s.client != nil,
		Loading:     s.loading,
		CatalogHint: s.catalogHint,
	}
	if s.err != nil {
		st.Err = s.err.Error()
	}
	st.Rows = append([]Entry(nil), s.rows...)
	return st
}

// Refresh loads schedule/list (and optional catalog) for sessionID.
func (s *Store) Refresh(ctx context.Context, sessionID string) {
	s.mu.Lock()
	client := s.client
	if client == nil {
		s.mu.Unlock()
		return
	}
	s.loading = true
	s.err = nil
	s.mu.Unlock()
	s.fire()

	go func() {
		var catalog api.CatalogResult
		catalogHint := ""
		if err := client.Call(ctx, "schedule/catalog", nil, &catalog); err == nil {
			catalogHint = formatCatalogHint(catalog)
		}

		var rows []Entry
		var callErr error
		if sessionID == "" {
			callErr = fmt.Errorf("select a session to list schedules")
		} else {
			var list api.ListResult4
			req := map[string]any{"sessionId": sessionID}
			callErr = client.Call(ctx, "schedule/list", map[string]any{"request": req}, &list)
			if callErr == nil {
				rows = parseScheduleList(list)
			}
		}
		s.finish(callErr, rows, catalogHint)
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
	raw, _ := json.MarshalIndent(out, "", "  ")
	if len(raw) > 4000 {
		return string(raw[:4000]) + "\n…", nil
	}
	return string(raw), nil
}

// Delete removes one schedule by id in the given session.
func (s *Store) Delete(ctx context.Context, sessionID, id string) {
	id = trim(id)
	sessionID = trim(sessionID)
	if id == "" || sessionID == "" {
		return
	}
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()
	if client == nil {
		return
	}
	go func() {
		req := api.ScheduleDeleteRequest{Id: id, SessionId: sessionID}
		var out api.ScheduleDeleteResult
		err := client.Call(ctx, "schedule/delete", map[string]any{"request": req}, &out)
		if err != nil {
			s.mu.Lock()
			s.err = err
			s.mu.Unlock()
			s.fire()
			return
		}
		s.Refresh(ctx, sessionID)
	}()
}

func (s *Store) finish(err error, rows []Entry, catalogHint string) {
	s.mu.Lock()
	s.loading = false
	s.err = err
	s.rows = rows
	s.catalogHint = catalogHint
	s.mu.Unlock()
	s.fire()
}

func parseScheduleList(list api.ListResult4) []Entry {
	out := make([]Entry, 0, len(list))
	for _, raw := range list {
		var item struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Prompt string `json:"prompt"`
			Cron   string `json:"cron"`
			Status string `json:"status"`
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
		detail := item.Status
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
		out = append(out, Entry{ID: item.ID, Title: title, Detail: detail, Raw: append(json.RawMessage(nil), raw...)})
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
	id = trim(id)
	sessionID = trim(sessionID)
	if id == "" || sessionID == "" {
		return fmt.Errorf("missing schedule id/session")
	}
	s.mu.Lock()
	client := s.client
	var expected json.RawMessage
	for _, row := range s.rows {
		if row.ID == id {
			expected = append(json.RawMessage(nil), row.Raw...)
			break
		}
	}
	s.mu.Unlock()
	if client == nil {
		return fmt.Errorf("not connected")
	}
	if len(expected) == 0 {
		expected = json.RawMessage(`{}`)
	}
	req := api.ScheduleUpdateRequest{
		Expected:  expected,
		Id:        id,
		SessionId: sessionID,
		Title:     title,
		Prompt:    prompt,
	}
	var out api.ScheduleUpdateResult
	if err := client.Call(ctx, "schedule/update", map[string]any{"request": req}, &out); err != nil {
		return err
	}
	s.Refresh(ctx, sessionID)
	return nil
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
