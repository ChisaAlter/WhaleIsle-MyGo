package pet

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf16"

	"whaleisle/internal/desktop/pluginrpc"
	"whaleisle/internal/dsh"
	"whaleisle/internal/dsh/api"
)

// These are the original dsh-whale pet/state and pet/select-model wire shapes.
// History is a projection of the persistent assistant session, never a local
// conversation, optimistic assistant message, or direct-model fallback.
type quickChatModel struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ReasoningEffort string `json:"reasoningEffort"`
}
type quickChatEffort struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type quickChatCatalogModel struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Efforts       []quickChatEffort `json:"efforts"`
	DefaultEffort string            `json:"defaultEffort"`
}
type quickChatGroup struct {
	ID     string                  `json:"id"`
	Name   string                  `json:"name"`
	Models []quickChatCatalogModel `json:"models"`
}
type quickChatRow struct {
	Role string  `json:"role"`
	Text string  `json:"text"`
	Seq  float64 `json:"seq"`
}
type quickChatSnapshot struct {
	OK        bool             `json:"ok"`
	SessionID string           `json:"sessionId"`
	Name      string           `json:"name"`
	Model     *quickChatModel  `json:"model"`
	Groups    []quickChatGroup `json:"groups"`
	History   []quickChatRow   `json:"history"`
}

func (s *quickChatSnapshot) replaceHistory(next quickChatSnapshot) bool {
	changed := s.SessionID != next.SessionID
	groups := s.Groups
	*s = next
	if !changed {
		s.Groups = groups
	}
	return changed
}
func (s quickChatSnapshot) allows(choice quickChatModel) bool {
	for _, g := range s.Groups {
		if g.ID != choice.Provider {
			continue
		}
		for _, m := range g.Models {
			if m.ID != choice.Model {
				continue
			}
			if choice.ReasoningEffort == "" {
				return true
			}
			for _, e := range m.Efforts {
				if e.ID == choice.ReasoningEffort {
					return true
				}
			}
		}
	}
	return false
}
func quickChatText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", errors.New("empty prompt")
	}
	// The original endpoint checks JavaScript String.length, not UTF-8 bytes.
	if len(utf16.Encode([]rune(text))) > 2000 {
		return "", errors.New("prompt exceeds 2000 UTF-16 units")
	}
	return text, nil
}

// quickChatRequest is owned by the UI thread. Cancellation changes the serial
// before a queued Window.Update can report the previous operation's result.
type quickChatRequest struct {
	serial uint64
	cancel context.CancelFunc
}

func (r *quickChatRequest) busy() bool { return r.cancel != nil }
func (r *quickChatRequest) begin(parent context.Context, timeout time.Duration) (context.Context, uint64) {
	if r.busy() {
		return nil, 0
	}
	r.serial++
	ctx, cancel := context.WithTimeout(parent, timeout)
	r.cancel = cancel
	return ctx, r.serial
}
func (r *quickChatRequest) finish(serial uint64) bool {
	if !r.busy() || serial != r.serial {
		return false
	}
	r.cancel()
	r.cancel = nil
	return true
}
func (r *quickChatRequest) abort() {
	if r.cancel != nil {
		r.cancel()
	}
	r.cancel = nil
	r.serial++
}

type quickChatClient struct {
	plugin  *pluginrpc.Client
	session *dsh.Client
}

func newQuickChatClient(origin, cookie string) (*quickChatClient, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.Opaque != "" || (u.Scheme != "http" && u.Scheme != "https") || (u.Path != "" && u.Path != "/") || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawFragment != "" {
		return nil, errors.New("invalid Host origin")
	}
	httpClient := &http.Client{Timeout: 140 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return &quickChatClient{
		plugin:  &pluginrpc.Client{BaseURL: origin, Cookie: cookie, HTTPClient: httpClient},
		session: &dsh.Client{Endpoint: dsh.Endpoint{Origin: origin, Cookie: cookie}, HTTP: httpClient},
	}, nil
}
func (c *quickChatClient) state(ctx context.Context, catalog bool) (quickChatSnapshot, error) {
	var out quickChatSnapshot
	raw, err := c.plugin.WhalePetState(ctx, catalog)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	out.SessionID = strings.TrimSpace(out.SessionID)
	if !out.OK || out.SessionID == "" {
		return out, errors.New("pet state unavailable")
	}
	return out, nil
}
func (c *quickChatClient) sameSession(ctx context.Context, expected string) error {
	st, err := c.state(ctx, false)
	if err != nil {
		return err
	}
	if expected == "" || st.SessionID != expected {
		return errQuickChatSessionChanged
	}
	return ctx.Err()
}

var errQuickChatSessionChanged = errors.New("pet session changed")

func (c *quickChatClient) send(ctx context.Context, expected, text string) error {
	text, err := quickChatText(text)
	if err != nil {
		return err
	}
	if err := c.sameSession(ctx, expected); err != nil {
		return err
	}
	_, err = c.plugin.WhalePetChatTyped(ctx, text)
	return err
}
func (c *quickChatClient) selectModel(ctx context.Context, expected string, choice quickChatModel) (*quickChatModel, error) {
	st, err := c.state(ctx, true)
	if err != nil {
		return nil, err
	}
	if expected == "" || st.SessionID != expected {
		return nil, errQuickChatSessionChanged
	}
	if !st.allows(choice) {
		return nil, errors.New("model choice no longer available")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out struct {
		OK       bool            `json:"ok"`
		Selected *quickChatModel `json:"selected"`
	}
	if err := c.plugin.Call(ctx, "dsh-whale", "pet/select-model", choice, &out); err != nil {
		return nil, err
	}
	if !out.OK || out.Selected == nil || out.Selected.Provider == "" || out.Selected.Model == "" {
		return nil, errors.New("model selection rejected or missing receipt")
	}
	return out.Selected, nil
}

// The actual native contract cancels the currently active turn of this session.
// It has no requestId/turnId selector. Do not call this "cancel this message".
func (c *quickChatClient) stop(ctx context.Context, expected string) error {
	if err := c.sameSession(ctx, expected); err != nil {
		return err
	}
	var out api.SessionCancelValue
	if err := c.session.Call(ctx, "session/cancel", map[string]any{"request": map[string]string{"sessionId": expected}}, &out); err != nil {
		return err
	}
	if !out.Accepted {
		return errors.New("turn cancellation not accepted")
	}
	return nil
}

// A fixed-size server tail can rotate without changing its length. Keep row
// identity by durable seq so scrolling/selection do not follow a different row.
func quickChatHistoryKey(state quickChatSnapshot, index int) any {
	if index < 0 || index >= len(state.History) {
		return index
	}
	row := state.History[index]
	if row.Seq > 0 {
		return struct {
			Session string
			Seq     float64
			Role    string
		}{state.SessionID, row.Seq, row.Role}
	}
	return struct {
		Session    string
		Index      int
		Role, Text string
	}{state.SessionID, index, row.Role, row.Text}
}
