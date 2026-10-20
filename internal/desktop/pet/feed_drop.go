package pet

import (
	_ "embed"
	"net/http"
	"net/url"
	"strings"
	"time"
	"whaleisle/internal/store/storelocale"
)

//go:embed assets/pet-drop.js
var fileDropScript []byte

func serveFileDrop(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/pet-drop.js" {
		return false
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(fileDropScript)
	return true
}

func droppedFileName(raw string) (string, bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "whale-pet" || u.Host != "localhost" || u.Path != "/feed-drop" || u.User != nil || u.Fragment != "" {
		return "", false
	}
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(values) != 1 || len(values["name"]) != 1 {
		return "", false
	}
	name := strings.TrimSpace(values.Get("name"))
	if name == "" || len(name) > 1024 || strings.ContainsAny(name, "/\\\x00\r\n") {
		return "", false
	}
	return name, true
}

func (p *Companion) fileDropAction(raw string) bool {
	name, ok := droppedFileName(raw)
	if !ok {
		return false
	}
	if p.win == nil || p.win.IsDestroyed() || !p.ready || p.Store == nil || !p.Store.DesktopPet().Enabled {
		return true
	}
	p.assetMu.Lock()
	defer p.assetMu.Unlock()
	if p.feeding {
		return true
	}
	if err := p.Store.FeedPetFile(time.Now()); err != nil {
		p.feedMessage = storelocale.T(storelocale.KeyDesktopPetSaveFailedNotApplied)
		return true
	}
	p.careAction = newCareAnimation("file", name, 0)
	_ = p.rest.wake(nil)
	p.feedMessage = "吃掉了「" + name + storelocale.T(storelocale.KeyDesktopPetFed2)
	return true
}
