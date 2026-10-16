package pluginslots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const clientModulesPath = "/__whale_client_modules"
const clientModulesProtocol = "whaleisle.client-modules.v1"

// ClientModuleFailure is the original page-owned entries.state failure shape.
// It is independent of Host bundle enablement and Loader inventory.
type ClientModuleFailure struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type ClientModulesSnapshot struct {
	Document                     *DocumentRecommendation
	Page                         uint64
	Connected, Syncing, Retrying bool
	Failures                     []ClientModuleFailure
	Error                        string
}

type clientModulesPage struct {
	socket      *websocket.Conn
	id, retryID uint64
}
type clientModulesBridge struct {
	documentSession, documentPath string
	mu                            sync.Mutex
	page                          *clientModulesPage
	nextPage, nextRetry           uint64
	closed                        bool
	state                         ClientModulesSnapshot
	changed                       func()
}

func (g *Gateway) SetClientModulesChanged(changed func()) {
	if g == nil {
		return
	}
	g.modules.mu.Lock()
	g.modules.changed = changed
	g.modules.mu.Unlock()
}

func (g *Gateway) ClientModules() ClientModulesSnapshot {
	if g == nil {
		return ClientModulesSnapshot{}
	}
	g.modules.mu.Lock()
	defer g.modules.mu.Unlock()
	state := g.modules.state
	if state.Document != nil {
		document := *state.Document
		state.Document = &document
	}
	state.Failures = append([]ClientModuleFailure(nil), state.Failures...)
	return state
}

// RetryClientModules addresses only the captured page generation. It sends no
// Host RPC and never refreshes inventory or changes bundle enablement.
func (g *Gateway) RetryClientModules(pageID uint64) error {
	if g == nil {
		return errors.New("client modules: no connected page")
	}
	b := &g.modules
	b.mu.Lock()
	page := b.page
	if b.closed || page == nil || !b.state.Connected || page.id != pageID {
		b.mu.Unlock()
		return errors.New("client modules: page is no longer connected")
	}
	if b.state.Retrying || b.state.Syncing {
		b.mu.Unlock()
		return errors.New("client modules: synchronization is already running")
	}
	b.nextRetry++
	page.retryID = b.nextRetry
	command := struct {
		Type string `json:"type"`
		ID   uint64 `json:"id"`
	}{"retry", page.retryID}
	b.state.Retrying, b.state.Error = true, ""
	changed := b.changed
	b.mu.Unlock()
	if changed != nil {
		changed()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, page.socket, command); err != nil {
		g.disconnectClientModules(page, false)
		return errors.New("client modules: page disconnected before retry")
	}
	return nil
}

// DisconnectClientModules revokes the current document on navigation, render
// failure or disposal. A later document must establish its own connection.
func (g *Gateway) DisconnectClientModules() { g.disconnectClientModules(nil, false) }

func (g *Gateway) disconnectClientModules(expected *clientModulesPage, closeBridge bool) {
	if g == nil {
		return
	}
	b := &g.modules
	b.mu.Lock()
	if expected != nil && b.page != expected {
		b.mu.Unlock()
		return
	}
	page := b.page
	b.page = nil
	b.state = ClientModulesSnapshot{}
	if closeBridge {
		b.closed = true
	}
	changed := b.changed
	b.mu.Unlock()
	if page != nil {
		_ = page.socket.CloseNow()
	}
	if changed != nil {
		changed()
	}
}

// serveClientModules is reached only after the gateway's existing ephemeral
// cookie and exact-origin checks. The fixed protocol accepts two data shapes;
// it exposes no native binding, arbitrary RPC, URL, script, or file operation.
func (g *Gateway) serveClientModules(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.Header.Get("Origin") != g.browserOrigin || r.URL.RawQuery != "" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	protocol := false
	for _, value := range strings.Split(r.Header.Get("Sec-WebSocket-Protocol"), ",") {
		if strings.TrimSpace(value) == clientModulesProtocol {
			protocol = true
		}
	}
	if !protocol {
		http.Error(w, "Unsupported page protocol", http.StatusBadRequest)
		return
	}
	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{clientModulesProtocol}})
	if err != nil {
		return
	}
	socket.SetReadLimit(256 * 1024)
	b := &g.modules
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		_ = socket.CloseNow()
		return
	}
	old := b.page
	b.nextPage++
	page := &clientModulesPage{socket: socket, id: b.nextPage}
	b.page = page
	b.state = ClientModulesSnapshot{Page: page.id}
	changed := b.changed
	b.mu.Unlock()
	if old != nil {
		_ = old.socket.CloseNow()
	}
	if changed != nil {
		changed()
	}
	defer g.disconnectClientModules(page, false)
	for {
		kind, data, err := socket.Read(r.Context())
		if err != nil {
			return
		}
		var message struct {
			Type     string                  `json:"type"`
			Document *DocumentRecommendation `json:"document"`
			ID       uint64                  `json:"id"`
			Error    string                  `json:"error"`
			Syncing  *bool                   `json:"syncing"`
			Failures *[]ClientModuleFailure  `json:"failures"`
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if kind != websocket.MessageText || decoder.Decode(&message) != nil || decoder.Decode(new(any)) != io.EOF {
			return
		}
		switch message.Type {
		case "state":
			if message.Document != nil || message.Syncing == nil || message.Failures == nil || message.ID != 0 || message.Error != "" || !validClientFailures(*message.Failures) {
				return
			}
		case "document":
			if message.Document == nil || message.Syncing != nil || message.Failures != nil || message.ID != 0 || message.Error != "" {
				return
			}
		case "retryResult":
			if message.Document != nil || message.ID == 0 || message.Syncing != nil || message.Failures != nil || len(message.Error) > 16*1024 {
				return
			}
		default:
			return
		}
		b.mu.Lock()
		if b.page != page || b.closed {
			b.mu.Unlock()
			return
		}
		if message.Type == "document" {
			if !b.acceptsDocument(message.Document) {
				b.mu.Unlock()
				return
			}
			document := *message.Document
			b.state.Document = &document
		} else if message.Type == "state" {
			b.state.Connected, b.state.Syncing = true, *message.Syncing
			b.state.Failures = append([]ClientModuleFailure(nil), (*message.Failures)...)
		} else if b.state.Retrying && message.ID == page.retryID {
			b.state.Retrying, b.state.Error = false, message.Error
		}
		changed = b.changed
		b.mu.Unlock()
		if changed != nil {
			changed()
		}
	}
}

func validClientFailures(failures []ClientModuleFailure) bool {
	if len(failures) > 512 {
		return false
	}
	for _, failure := range failures {
		if failure.ID == "" || len(failure.ID) > 4096 || len(failure.Message) > 16*1024 {
			return false
		}
	}
	return true
}
