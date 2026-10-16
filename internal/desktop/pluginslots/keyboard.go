package pluginslots

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const keyboardPath = "/__whale_keyboard"
const keyboardProtocol = "whaleisle.product-keyboard.v1"

type ProductKeyBinding struct {
	Code       string `json:"code"`
	Mods       uint8  `json:"mods"`
	SecondCode string `json:"secondCode,omitempty"`
}
type ProductKeyboardSnapshot struct {
	Page     uint64
	Revision string
	Active   bool
	Bindings []ProductKeyBinding
}
type productKeyboardBridge struct {
	mu      sync.Mutex
	socket  *websocket.Conn
	next    uint64
	state   ProductKeyboardSnapshot
	changed func()
}

func (g *Gateway) SetProductKeyboardChanged(fn func()) {
	g.keyboard.mu.Lock()
	g.keyboard.changed = fn
	g.keyboard.mu.Unlock()
}
func (g *Gateway) ProductKeyboard() ProductKeyboardSnapshot {
	g.keyboard.mu.Lock()
	defer g.keyboard.mu.Unlock()
	s := g.keyboard.state
	s.Bindings = append([]ProductKeyBinding(nil), s.Bindings...)
	return s
}
func (g *Gateway) DisconnectProductKeyboard() { g.disconnectProductKeyboard(nil) }
func (g *Gateway) disconnectProductKeyboard(expected *websocket.Conn) {
	b := &g.keyboard
	b.mu.Lock()
	if expected != nil && b.socket != expected {
		b.mu.Unlock()
		return
	}
	socket := b.socket
	b.socket = nil
	b.state = ProductKeyboardSnapshot{}
	changed := b.changed
	b.mu.Unlock()
	if socket != nil {
		_ = socket.CloseNow()
	}
	if changed != nil {
		changed()
	}
}

// Only physical native key delivery calls this; the HTTP endpoint has no
// command, file, Host-RPC, credential or arbitrary-message operation.
func (g *Gateway) DispatchProductKey(page uint64, revision, code string, mods uint8, secondCode ...string) error {
	b := &g.keyboard
	b.mu.Lock()
	socket, s := b.socket, b.state
	second := ""
	if len(secondCode) > 0 {
		second = secondCode[0]
	}
	accepted := false
	for _, key := range s.Bindings {
		if key.Code == code && key.Mods == mods && key.SecondCode == second {
			accepted = true
			break
		}
	}
	if socket == nil || !s.Active || s.Page != page || s.Revision != revision || !accepted {
		b.mu.Unlock()
		return errors.New("product keyboard source changed")
	}
	b.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return wsjson.Write(ctx, socket, struct {
		Type       string `json:"type"`
		Revision   string `json:"revision"`
		Code       string `json:"code"`
		Mods       uint8  `json:"mods"`
		SecondCode string `json:"secondCode,omitempty"`
	}{"key", revision, code, mods, second})
}
func validProductKey(b ProductKeyBinding) bool {
	if b.Mods > 15 || (b.SecondCode != "" && (b.Code >= b.SecondCode || !validProductCode(b.SecondCode))) {
		return false
	}
	return validProductCode(b.Code)
}
func validProductCode(c string) bool {
	if len(c) == 4 && strings.HasPrefix(c, "Key") && c[3] >= 'A' && c[3] <= 'Z' {
		return true
	}
	if len(c) == 6 && strings.HasPrefix(c, "Digit") && c[5] >= '0' && c[5] <= '9' {
		return true
	}
	switch c {
	case "Escape", "Space", "Tab", "Backspace", "Delete", "ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "F13", "F14", "F15", "F16", "F17", "F18", "F19", "F20", "F21", "F22", "F23", "F24", "Comma", "Period", "Slash", "Backslash", "Backquote", "Semicolon", "Quote", "BracketLeft", "BracketRight", "Minus", "Equal", "Enter", "F1", "F2", "F3", "F4", "F5", "F6", "F7", "F8", "F9", "F10", "F11", "F12":
		return true
	}
	return false
}
func decodeProductKeyboard(raw []byte) (ProductKeyboardSnapshot, error) {
	var input struct {
		Type     string              `json:"type"`
		Revision string              `json:"revision"`
		Active   bool                `json:"active"`
		Bindings []ProductKeyBinding `json:"bindings"`
	}
	if err := json.Unmarshal(raw, &input); err != nil {
		return ProductKeyboardSnapshot{}, err
	}
	if input.Type != "state" || len(input.Revision) == 0 || len(input.Revision) > 256 || len(input.Bindings) > 512 {
		return ProductKeyboardSnapshot{}, errors.New("invalid product keyboard state")
	}
	seen := map[ProductKeyBinding]bool{}
	for _, b := range input.Bindings {
		if !validProductKey(b) || seen[b] {
			return ProductKeyboardSnapshot{}, errors.New("invalid/conflicting product binding")
		}
		for previous := range seen {
			if previous.Mods == b.Mods && (previous.SecondCode == "" || b.SecondCode == "") && (previous.Code == b.Code || previous.Code == b.SecondCode || (previous.SecondCode != "" && (previous.SecondCode == b.Code || previous.SecondCode == b.SecondCode))) {
				return ProductKeyboardSnapshot{}, errors.New("overlapping product binding")
			}
		}
		seen[b] = true
	}
	return ProductKeyboardSnapshot{Revision: input.Revision, Active: input.Active, Bindings: input.Bindings}, nil
}
func (g *Gateway) serveProductKeyboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet || r.Header.Get("Origin") != g.browserOrigin || r.URL.RawQuery != "" || r.Header.Get("Sec-WebSocket-Protocol") != keyboardProtocol {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{Subprotocols: []string{keyboardProtocol}})
	if err != nil {
		return
	}
	socket.SetReadLimit(128 * 1024)
	b := &g.keyboard
	b.mu.Lock()
	old := b.socket
	b.next++
	page := b.next
	b.socket = socket
	b.state = ProductKeyboardSnapshot{Page: page}
	changed := b.changed
	b.mu.Unlock()
	if old != nil {
		_ = old.CloseNow()
	}
	if changed != nil {
		changed()
	}
	defer func() { g.disconnectProductKeyboard(socket); _ = socket.CloseNow() }()
	for {
		_, raw, err := socket.Read(r.Context())
		if err != nil {
			return
		}
		state, err := decodeProductKeyboard(raw)
		if err != nil {
			return
		}
		b.mu.Lock()
		if b.socket != socket {
			b.mu.Unlock()
			return
		}
		state.Page = page
		b.state = state
		changed = b.changed
		b.mu.Unlock()
		if changed != nil {
			changed()
		}
	}
}
