package pet

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo/ui"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"whaleisle/internal/petwatch"
)

//go:embed assets/pet-notifications.js
var notificationScript []byte

type notificationState struct {
	owner                string
	snapshot             petwatch.Snapshot
	ack                  func(string)
	mediaHome, mediaRoot string
	chatImages           map[string]bool
}

// NotificationsActive is called only on the UI thread. Closing/hiding the pet
// ends the reader, while a later explicit show can resume its durable backlog.
func (p *Companion) NotificationsActive() bool {
	return p.ready && p.win != nil && !p.win.IsDestroyed() && p.Store != nil && p.Store.DesktopPet().Enabled
}
func (p *Companion) UpdateNotifications(owner string, snap petwatch.Snapshot, ack func(string), home, stickerRoot string) {
	if !p.NotificationsActive() {
		return
	}
	inChat := map[string]bool{}
	if q := p.QuickChat; q != nil && q.check() {
		for _, e := range snap.Events {
			if e.MediaPath != "" && q.presentSticker(owner, e, home, stickerRoot, ack) {
				inChat[e.ID] = true
			}
		}
	}
	p.assetMu.Lock()
	p.notifications = notificationState{owner: owner, snapshot: snap, ack: ack, mediaHome: home, mediaRoot: stickerRoot, chatImages: inChat}
	p.assetMu.Unlock()
}

// Sticker decoding uses the same confined roots as notification media. The
// receipt is acknowledged by the visible virtual-list image paint, not by a
// background read or merely opening a card.
func (q *QuickChat) presentSticker(owner string, e petwatch.Event, home, root string, ack func(string)) bool {
	key := owner + ":" + e.ID
	if shown, known := q.stickerSeen[key]; known {
		return shown
	}
	q.stickerSeen[key] = true
	go func() {
		data, _, err := petwatch.ReadSticker(home, root, e.MediaPath)
		var bitmap *ui.Bitmap
		if err == nil {
			bitmap, err = ui.DecodeBitmap(data)
		}
		clear(data)
		q.win.Update(func() {
			if !q.check() {
				return
			}
			q.owner.assetMu.RLock()
			current := q.owner.notifications.owner == owner
			q.owner.assetMu.RUnlock()
			if !current {
				return
			}
			if err != nil {
				q.stickerSeen[key] = false
				return
			}
			var once sync.Once
			row := quickChatRow{Role: "her", Text: e.Text, Image: bitmap, Receipt: key, Acknowledge: func() {
				once.Do(func() {
					if ack != nil {
						ack(e.ID)
					}
				})
			}}
			q.stickers = append(q.stickers, row)
			q.state.History = append(q.state.History, row)
			q.list.ScrollToEnd()
		})
	}()
	return true
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
	if strings.HasPrefix(r.URL.Path, "/notification-image/") {
		p.assetMu.RLock()
		n := p.notifications
		p.assetMu.RUnlock()
		for _, e := range n.snapshot.Events {
			if r.URL.Path != "/notification-image/"+n.owner+"/"+e.ID {
				continue
			}
			data, mime, err := petwatch.ReadSticker(n.mediaHome, n.mediaRoot, e.MediaPath)
			if err != nil {
				http.NotFound(w, r)
				return true
			}
			// Disk decoding may race a Host switch or receipt acknowledgement.
			// Recheck the issuing source before publishing any image bytes.
			p.assetMu.RLock()
			current := p.notifications.owner == n.owner && p.notifications.mediaHome == n.mediaHome && p.notifications.mediaRoot == n.mediaRoot
			found := false
			if current {
				for _, active := range p.notifications.snapshot.Events {
					if active.ID == e.ID && active.MediaPath == e.MediaPath {
						found = true
						break
					}
				}
			}
			p.assetMu.RUnlock()
			if !current || !found {
				http.NotFound(w, r)
				return true
			}
			w.Header().Set("Content-Type", mime)
			w.Header().Set("Content-Length", fmt.Sprint(len(data)))
			if r.Method != "HEAD" {
				_, _ = w.Write(data)
			}
			return true
		}
		http.NotFound(w, r)
		return true
	}
	switch r.URL.Path {
	case "/pet-notifications.js":
		body = notificationScript
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case "/notifications.json":
		p.assetMu.RLock()
		chatOpen := p.quickChatOpen
		n := p.notifications
		p.assetMu.RUnlock()
		type eventView struct {
			ID        string `json:"id"`
			Type      string `json:"type"`
			Text      string `json:"text"`
			SessionID string `json:"sessionID,omitempty"`
			Pinned    bool   `json:"pinned,omitempty"`
			Image     string `json:"image,omitempty"`
		}
		events := make([]eventView, 0, len(n.snapshot.Events))
		for _, e := range n.snapshot.Events {
			if chatOpen && n.chatImages[e.ID] {
				continue
			}
			if chatOpen && n.chatImages[e.ID] {
				continue
			}
			image := ""
			if e.MediaPath != "" {
				image = "notification-image/" + n.owner + "/" + e.ID
			}
			events = append(events, eventView{e.ID, e.Type, e.Text, e.SessionID, e.Pinned, image})
		}
		body, _ = json.Marshal(struct {
			Owner  string      `json:"owner"`
			State  string      `json:"state"`
			Events []eventView `json:"events"`
		}{n.owner, n.snapshot.State, events})
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

func notificationStateMode(s petwatch.Snapshot) string {
	for _, e := range s.Events {
		if e.Type == "error" {
			return "error"
		}
	}
	for _, e := range s.Events {
		if e.Pinned {
			return "review"
		}
	}
	return ""
}
