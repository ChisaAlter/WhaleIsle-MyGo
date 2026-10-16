// Package pluginslots supplies the desktop's live Cordis view adapter and a
// short-lived, exact-Host browser transport. It never exposes native bindings.
package pluginslots

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Gateway owns one loopback listener and one already-authorized Host session.
// Neither the Host credential nor the browser capability is persisted.
type Gateway struct {
	keyboard productKeyboardBridge
	modules                                          clientModulesBridge
	connectionsMu                                    sync.Mutex
	connections                                      map[net.Conn]bool
	origin, cookie, browserOrigin, token, cookieName string
	target                                           *url.URL
	server                                           *http.Server
	once                                             sync.Once
}

func NewGateway(origin, cookie string) (*Gateway, error) {
	target, err := url.Parse(origin)
	if err != nil || target.User != nil || (target.Scheme != "http" && target.Scheme != "https") || target.RawQuery != "" || target.Fragment != "" || (target.Path != "" && target.Path != "/") {
		return nil, errors.New("plugin view: invalid Host origin")
	}
	ip := net.ParseIP(target.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return nil, errors.New("plugin view: Host must be an exact loopback address")
	}
	if strings.ContainsAny(cookie, "\r\n") {
		return nil, errors.New("plugin view: invalid Host session")
	}
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	g := &Gateway{connections: make(map[net.Conn]bool), origin: target.Scheme + "://" + target.Host, cookie: cookie, target: target, token: hex.EncodeToString(secret)}
	g.browserOrigin = "http://" + listener.Addr().String()
	g.cookieName = "whale-slot-" + g.token[:16]
	proxy := &httputil.ReverseProxy{
		Rewrite: func(p *httputil.ProxyRequest) {
			p.SetURL(target)
			p.Out.Host = target.Host
			// Drop every incoming credential and forwarding hint. Only this pinned
			// session reaches this pinned Host; redirects are never followed.
			p.Out.Header.Del("Cookie")
			p.Out.Header.Del("Authorization")
			if cookie != "" {
				p.Out.Header.Set("Cookie", cookie)
			}
			if p.In.Header.Get("Origin") != "" {
				p.Out.Header.Set("Origin", g.origin)
			}
			p.Out.Header.Del("Referer")
		},
		ModifyResponse: func(r *http.Response) error {
			r.Header.Del("Set-Cookie") // Host credentials stay in Go, including rotations.
			r.Header.Set("Referrer-Policy", "no-referrer")
			if raw := r.Header.Get("Location"); raw != "" {
				dest, err := r.Request.URL.Parse(raw)
				if err != nil || dest.User != nil || dest.Scheme != target.Scheme || dest.Host != target.Host {
					return errors.New("plugin view: external redirect blocked")
				}
				dest.Scheme = "http"
				dest.Host = listener.Addr().String()
				r.Header.Set("Location", dest.String())
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "Plugin Host request failed", http.StatusBadGateway)
		},
	}
	g.server = &http.Server{ReadHeaderTimeout: 10 * time.Second, ConnState: func(c net.Conn, state http.ConnState) {
		g.connectionsMu.Lock()
		defer g.connectionsMu.Unlock()
		if state == http.StateClosed {
			delete(g.connections, c)
		} else {
			g.connections[c] = true
		}
	}, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != listener.Addr().String() || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != g.browserOrigin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.URL.Path == "/__whale_slot_auth" {
			next, err := url.Parse(r.URL.Query().Get("next"))
			if r.Method != "GET" || !sameSecret(r.URL.Query().Get("key"), g.token) || err != nil || next.IsAbs() || next.Host != "" || !strings.HasPrefix(next.Path, "/") || strings.HasPrefix(next.Path, "//") {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: g.cookieName, Value: g.token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
			http.Redirect(w, r, next.String(), http.StatusSeeOther)
			return
		}
		session, err := r.Cookie(g.cookieName)
		if err != nil || !sameSecret(session.Value, g.token) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if r.URL.Path == keyboardPath {
			g.serveProductKeyboard(w, r)
			return
		}
		if r.URL.Path == clientModulesPath {
			g.serveClientModules(w, r)
			return
		}
		proxy.ServeHTTP(w, r)
	})}
	go func() { _ = g.server.Serve(listener) }()
	return g, nil
}
func sameSecret(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// EntryURL authenticates only the ephemeral browser origin, then removes that
// capability from the visible URL before loading the original Host path/query.
func (g *Gateway) EntryURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Scheme != g.target.Scheme || u.Host != g.target.Host {
		return "", errors.New("plugin view: target is outside the active Host")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	u.Scheme = ""
	u.Host = ""
	query := url.Values{"key": {g.token}, "next": {u.String()}}
	return g.browserOrigin + "/__whale_slot_auth?" + query.Encode(), nil
}

// LogicalURL hides the private browser capability/port from native chrome,
// Copy URL, restoration and error messages.
func (g *Gateway) LogicalURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if u.Scheme+"://"+u.Host != g.browserOrigin {
		return raw
	}
	if u.Path == "/__whale_slot_auth" {
		u, err = url.Parse(u.Query().Get("next"))
		if err != nil {
			return ""
		}
	}
	u.Scheme = g.target.Scheme
	u.Host = g.target.Host
	return u.String()
}
func (g *Gateway) Matches(origin, cookie string) bool {
	return g != nil && g.origin == strings.TrimRight(origin, "/") && g.cookie == cookie
}
func (g *Gateway) Close() {
	if g != nil {
		g.once.Do(func() {
			g.DisconnectProductKeyboard()
			g.disconnectClientModules(nil, true)
			_ = g.server.Close()
			g.connectionsMu.Lock()
			connections := make([]net.Conn, 0, len(g.connections))
			for connection := range g.connections {
				connections = append(connections, connection)
			}
			clear(g.connections)
			g.connectionsMu.Unlock()
			for _, connection := range connections {
				_ = connection.Close()
			}
		})
	}
}

func (g *Gateway) AllowsBrowserURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.User == nil && u.Scheme+"://"+u.Host == g.browserOrigin
}
