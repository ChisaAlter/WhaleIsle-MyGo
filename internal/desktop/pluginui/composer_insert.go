package pluginui

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

const composerInsertPath = "/__whale_plugin_ui/composer/insert"

// The public insertText boolean means already applied. Legacy synchronous
// callers use this one closed same-origin operation; no arbitrary RPC, file or
// desktop binding is exposed. Shipped voice callers use the async socket path.
func (o *Owner) serveComposerInsert(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != composerInsertPath {
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodPost || r.URL.RawQuery != "" || r.Header.Get("Origin") != "http://"+r.Host {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return true
	}
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		http.Error(w, "Invalid insertion", http.StatusBadRequest)
		return true
	}
	var request struct {
		Page    uint64              `json:"page"`
		Request string              `json:"request"`
		Owner   ComposerBinding     `json:"owner"`
		Replace ComposerReplacement `json:"replace"`
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024+16385))
	if err != nil || len(raw) > 1024*1024+16384 || json.Unmarshal(raw, &request) != nil || request.Request == "" || len(request.Request) > 64 || request.Replace.Start < 0 || request.Replace.End < request.Replace.Start || len(request.Replace.Text) > 1024*1024 {
		http.Error(w, "Invalid insertion", http.StatusBadRequest)
		return true
	}
	o.mu.Lock()
	page, bridge, masked := o.page, o.composer, o.masked
	o.mu.Unlock()
	if masked || bridge == nil || !o.composerTransferCurrent(page, request.Owner) || page.id != request.Page {
		http.Error(w, "Plugin composer owner expired", http.StatusConflict)
		return true
	}
	err = bridge.Apply(ComposerEdit{Owner: request.Owner, Replace: &request.Replace})
	reply := map[string]any{"type": "composerReply", "page": page.id, "request": request.Request, "ok": err == nil}
	if err != nil {
		reply["error"] = err.Error()
	} else if draft, readErr := bridge.Read(request.Owner.Subject); readErr != nil {
		reply["syncError"] = readErr.Error()
	} else if draft.Owner.Subject.equal(request.Owner.Subject) {
		reply["draft"] = draft
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(reply)
	o.notify()
	return true
}
