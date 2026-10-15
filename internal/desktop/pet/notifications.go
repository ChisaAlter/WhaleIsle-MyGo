package pet

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"whaleisle/internal/petwatch"
)

//go:embed assets/pet-notifications.js
var notificationScript []byte

type notificationState struct {
	owner    string
	snapshot petwatch.Snapshot
	ack      func(string)
}

// NotificationsActive is called only on the UI thread. Closing/hiding the pet
// ends the reader, while a later explicit show can resume its durable backlog.
func (p *Companion) NotificationsActive() bool {
	return p.ready && p.win != nil && !p.win.IsDestroyed() && p.Store != nil && p.Store.DesktopPet().Enabled
}
func (p *Companion) UpdateNotifications(owner string, snap petwatch.Snapshot, ack func(string)) {
	if !p.NotificationsActive() {
		return
	}
	p.assetMu.Lock()
	p.notifications = notificationState{owner: owner, snapshot: snap, ack: ack}
	p.assetMu.Unlock()
}
func (p *Companion) clearNotifications() {
	p.assetMu.Lock()
	p.notifications = notificationState{}
	p.assetMu.Unlock()
	if p.OnNotificationsClosed != nil {
		p.OnNotificationsClosed()
	}
}
func (p *Companion) ClearNotificationSource() {
	p.assetMu.Lock()
	p.notifications = notificationState{}
	p.assetMu.Unlock()
}

// notificationAction only accepts an exact token issued to this pet instance.
// Page-supplied session ids, paths, URLs and stale Host receipt tokens are denied.
func (p *Companion) notificationAction(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "whale-pet" || u.Host != "localhost" || u.User != nil || u.Fragment != "" {
		return false
	}
	if u.Path != "/notification-ack" && u.Path != "/notification-open" {
		return false
	}
	if !p.NotificationsActive() {
		return true
	}
	p.assetMu.RLock()
	n := p.notifications
	p.assetMu.RUnlock()
	for _, e := range n.snapshot.Events {
		if raw != "whale-pet://localhost"+u.Path+"?owner="+url.QueryEscape(n.owner)+"&id="+url.QueryEscape(e.ID) {
			continue
		}
		if u.Path == "/notification-ack" && n.ack != nil {
			n.ack(e.ID)
		}
		if u.Path == "/notification-open" && e.SessionID != "" && p.OnOpenNotificationSession != nil {
			p.OnOpenNotificationSession(n.owner, e.SessionID)
		}
		return true
	}
	return true
}
func renderNotificationPage(body []byte) []byte {
	return []byte(strings.Replace(string(body), "</body>", `<script src="pet-notifications.js" defer></script></body>`, 1))
}

// This is invoked only after Companion.ServeHTTP validates same-origin local
// GET/HEAD with no URL query. It never opens files or accepts a raw media URL.
func (p *Companion) serveNotifications(w http.ResponseWriter, r *http.Request) bool {
	var body []byte
	switch r.URL.Path {
	case "/pet-notifications.js":
		body = notificationScript
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case "/notifications.json":
		p.assetMu.RLock()
		n := p.notifications
		p.assetMu.RUnlock()
		body, _ = json.Marshal(struct {
			Owner string `json:"owner"`
			petwatch.Snapshot
		}{n.owner, n.snapshot})
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	default:
		return false
	}
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	if r.Method != "HEAD" {
		_, _ = w.Write(body)
	}
	return true
}
