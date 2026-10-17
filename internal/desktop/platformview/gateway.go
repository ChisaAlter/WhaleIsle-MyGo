package platformview

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Gateway is a single document lease. A fresh loopback origin and transient
// WebView storage isolate every account/open. All upstream requests are pinned
// to the exact authenticated origin; arbitrary URL proxying is never supported.
type Gateway struct {
	origin, key, cookieName string
	session                 Session
	server                  *http.Server
	transport               *http.Transport
	cancel                  context.CancelFunc
	once                    sync.Once
}
type ClientMetadata struct {
	Version, Locale, Platform string
	TimezoneOffsetSeconds     int
}

func New(session Session, metadata ClientMetadata) (*Gateway, error) {
	if err := session.Validate(); err != nil {
		return nil, err
	}
	if strings.HasPrefix(strings.ToLower(metadata.Locale), "zh") {
		metadata.Locale = "zh_CN"
	} else {
		metadata.Locale = "en_US"
	}
	session.Origin = strings.TrimRight(session.Origin, "/")
	target, _ := url.Parse(session.Origin)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, errors.New("Platform transport unavailable")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, errors.New("Platform transport unavailable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	g := &Gateway{origin: "http://" + listener.Addr().String(), key: hex.EncodeToString(b), session: session, cancel: cancel}
	g.cookieName = "whale-platform-" + g.key[:16]
	g.transport = http.DefaultTransport.(*http.Transport).Clone()
	// No ambient proxy configuration can receive account headers.
	g.transport.Proxy = nil
	proxy := &httputil.ReverseProxy{Transport: g.transport, Rewrite: func(p *httputil.ProxyRequest) {
		p.SetURL(target)
		p.Out.Host = target.Host
		authorized := equal(p.In.Header.Get("Authorization"), "Bearer "+session.Token)
		for _, k := range []string{"Cookie", "Authorization", "Proxy-Authorization", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "Referer"} {
			p.Out.Header.Del(k)
		}
		for k, v := range session.RequestHeaders {
			p.Out.Header.Set(k, v)
		}
		if authorized {
			p.Out.Header.Set("Authorization", "Bearer "+session.Token)
		}
		p.Out.Header.Set("x-client-platform", metadata.Platform)
		p.Out.Header.Set("x-client-version", metadata.Version)
		p.Out.Header.Set("x-client-locale", metadata.Locale)
		p.Out.Header.Set("x-client-bundle-id", "")
		p.Out.Header.Set("x-client-timezone-offset", strconv.Itoa(metadata.TimezoneOffsetSeconds))
		if p.In.Header.Get("Origin") != "" {
			p.Out.Header.Set("Origin", session.Origin)
		}
	}, ModifyResponse: func(r *http.Response) error {
		r.Header.Del("Set-Cookie")
		r.Header.Del("Access-Control-Allow-Origin")
		r.Header.Del("Access-Control-Allow-Credentials")
		r.Header.Set("Cache-Control", "no-store")
		r.Header.Set("Referrer-Policy", "no-referrer")
		// This additional policy never relaxes upstream CSP. Network and nested
		// browsing stay in this pinned lease; a third-party resource cannot see auth.
		r.Header.Add("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; frame-src 'none'; worker-src 'none'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
		if raw := r.Header.Get("Location"); raw != "" {
			u, e := target.Parse(raw)
			if e != nil || u.User != nil || u.Scheme != target.Scheme || u.Host != target.Host {
				return errors.New("Platform redirect blocked")
			}
			u.Scheme = "http"
			u.Host = listener.Addr().String()
			r.Header.Set("Location", u.String())
		}
		return nil
	}, ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, "Platform request failed", http.StatusBadGateway)
	}}
	g.server = &http.Server{ReadHeaderTimeout: 10 * time.Second, BaseContext: func(net.Listener) context.Context { return ctx }, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if r.Host != listener.Addr().String() || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != g.origin) || r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "Forbidden", 403)
			return
		}
		if r.URL.Path == "/__whale_platform_entry" {
			page := r.URL.Query().Get("page")
			if r.Method != "GET" || !equal(r.URL.Query().Get("key"), g.key) || (page != "usage" && page != "top-up") {
				http.Error(w, "Forbidden", 403)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: g.cookieName, Value: g.key, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			path := "/usage"
			if page == "top-up" {
				path = "/top_up"
			}
			if session.EmbeddedPageDist != "" {
				path += "?dist=" + url.QueryEscape(session.EmbeddedPageDist)
			}
			http.Redirect(w, r, path, http.StatusSeeOther)
			return
		}
		cookie, e := r.Cookie(g.cookieName)
		if e != nil || !equal(cookie.Value, g.key) {
			http.Error(w, "Forbidden", 403)
			return
		}
		if r.Method == "CONNECT" || r.Header.Get("Upgrade") != "" {
			http.Error(w, "Forbidden", 403)
			return
		}
		proxy.ServeHTTP(w, r)
	})}
	go func() { _ = g.server.Serve(listener) }()
	return g, nil
}
func equal(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (g *Gateway) Entry(page string) (string, error) {
	if page != "usage" && page != "top-up" {
		return "", errors.New("Unknown Platform page")
	}
	return g.origin + "/__whale_platform_entry?key=" + g.key + "&page=" + page, nil
}
func (g *Gateway) Allows(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.User == nil && u.Scheme+"://"+u.Host == g.origin
}
func (g *Gateway) Close() {
	if g == nil {
		return
	}
	g.once.Do(func() { g.cancel(); _ = g.server.Close(); g.transport.CloseIdleConnections() })
}
func (g *Gateway) Preload(locale string) string {
	if strings.HasPrefix(strings.ToLower(locale), "zh") {
		locale = "zh_CN"
	} else {
		locale = "en_US"
	}
	payload, _ := json.Marshal(map[string]string{"browserOrigin": g.origin, "origin": g.session.Origin, "token": g.session.Token, "locale": locale})
	return `(()=>{'use strict';const s=` + string(payload) + `;if(window!==window.top||location.origin!==s.browserOrigin)return;
 const rewrite=value=>{const u=new URL(value,location.href);if(u.origin===s.origin){return s.browserOrigin+u.pathname+u.search+u.hash;}return u.href;};
 const nativeFetch=window.fetch.bind(window);window.fetch=(input,init)=>{if(input instanceof Request){const mapped=rewrite(input.url);return nativeFetch(mapped===input.url?input:new Request(mapped,input),init);}return nativeFetch(rewrite(String(input)),init);};
 const open=XMLHttpRequest.prototype.open;XMLHttpRequest.prototype.open=function(method,url,...args){return open.call(this,method,rewrite(String(url)),...args);};
 let locale=s.locale;const listeners=new Set();Object.defineProperty(window,'dsh',{configurable:false,writable:false,value:Object.freeze({protocolVersion:1,displayMode:'embedded',getAuthToken:()=>s.token,getLocale:()=>locale,onLocaleChange:fn=>{listeners.add(fn);return()=>listeners.delete(fn);}})});
 // Native locale updates contain no credential and cannot replace the bridge.
 window.addEventListener('whale-platform-locale',e=>{if(e.detail!=='zh_CN'&&e.detail!=='en_US')return;locale=e.detail;for(const fn of listeners){try{fn(locale)}catch{}}});
 })();`
}
