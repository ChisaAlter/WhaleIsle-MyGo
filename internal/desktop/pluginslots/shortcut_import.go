package pluginslots

import (
	"encoding/json"
	"io"
	"net/http"
	"runtime"
)

const shortcutImportPath = "/__whale_shortcut_import"

func (g *Gateway) serveShortcutImport(w http.ResponseWriter, r *http.Request) {
	store := g.shortcutPreferences()
	if store == nil || r.Method != "POST" || r.Header.Get("Origin") != g.browserOrigin || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Forbidden", 403)
		return
	}
	var req struct {
		Operation  string          `json:"operation"`
		Document   json.RawMessage `json:"document"`
		Revision   string          `json:"revision"`
		ReplaceIDs []string        `json:"replaceIds"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		http.Error(w, "Invalid request", 400)
		return
	}
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	var result any
	switch req.Operation {
	case "preview":
		result = store.PreviewImport(req.Document, "web:"+platform)
	case "import":
		result = store.Import(req.Document, "web:"+platform, req.Revision, req.ReplaceIDs)
	default:
		http.Error(w, "Unsupported import operation", 400)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
