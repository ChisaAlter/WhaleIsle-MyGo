package pet

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"whaleisle/internal/desktop/pluginrpc"
)

// ChatStatus contains presentation state only. Host credentials and the ensured
// assistant identity never enter the pet page or its action URL.
type ChatStatus struct {
	Busy    bool
	Message string
}

type chatEntry struct {
	status ChatStatus
	serial uint64
	cancel context.CancelFunc
}

// Caller holds Companion.assetMu. A closed/recreated window cannot finish an
// old attempt; duplicate clicks cannot create concurrent ensure requests.
func (e *chatEntry) begin() (context.Context, uint64) {
	if e.status.Busy {
		return nil, 0
	}
	e.serial++
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	e.cancel = cancel
	e.status = ChatStatus{Busy: true, Message: "正在打开宠物会话… / Opening pet chat…"}
	return ctx, e.serial
}
func (e *chatEntry) finish(serial uint64, message string) bool {
	if serial != e.serial || !e.status.Busy {
		return false
	}
	e.cancel()
	e.cancel = nil
	e.status = ChatStatus{Message: message}
	return true
}
func (e *chatEntry) close() {
	if e.cancel != nil {
		e.cancel()
		e.cancel = nil
	}
	e.serial++
	e.status = ChatStatus{}
}

// StartChat runs on the UI thread, as does its completion callback. A nil
// context means the instance is closed or already opening its conversation.
func (p *Companion) StartChat() (context.Context, func(string) bool) {
	if p.win == nil || p.win.IsDestroyed() || !p.ChatAllowed() {
		return nil, nil
	}
	p.assetMu.Lock()
	ctx, serial := p.chat.begin()
	p.assetMu.Unlock()
	if ctx == nil {
		return nil, nil
	}
	return ctx, func(message string) bool {
		p.assetMu.Lock()
		defer p.assetMu.Unlock()
		return p.chat.finish(serial, message)
	}
}

// EnsureChatSession implements the original pet ↗ / __dshWhaleOpen contract:
// assistant/ensure owns creation, identity, preset and persisted settings. It
// never sends a prompt, changes the current task, or enables a missing plugin.
func EnsureChatSession(ctx context.Context, origin, cookie string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || u.RawPath != "" || u.Fragment != "" {
		return "", fmt.Errorf("pet chat: invalid Host origin")
	}
	client := &pluginrpc.Client{BaseURL: origin, Cookie: cookie, HTTPClient: &http.Client{
		Timeout: 30 * time.Second,
		// Even a same-host redirect must not forward this session's cookie to
		// another endpoint. Only the fixed authenticated ensure RPC is allowed.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
	result, err := client.WhaleEnsureTyped(ctx)
	if err != nil {
		return "", err
	}
	return result.SessionID, nil
}
