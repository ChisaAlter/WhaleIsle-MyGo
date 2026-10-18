package pluginslots

import (
	_ "embed"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"whaleisle/internal/shortcutprefs"
)

const shortcutPreferencesPath = "/__whale_shortcuts"

//go:embed shortcut_provider.js
var shortcutProviderScript string

func (g *Gateway) SetShortcutPreferences(store *shortcutprefs.Store) {
	g.preferencesMu.Lock()
	g.preferences = store
	g.preferencesMu.Unlock()
}
func (g *Gateway) shortcutPreferences() *shortcutprefs.Store {
	g.preferencesMu.Lock()
	defer g.preferencesMu.Unlock()
	return g.preferences
}
func (g *Gateway) ShortcutProviderScript() string {
	if g == nil || g.shortcutPreferences() == nil {
		return ""
	}
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	origin, _ := json.Marshal(g.browserOrigin)
	p, _ := json.Marshal(platform)
	return strings.ReplaceAll(strings.ReplaceAll(shortcutProviderScript, "__MYGO_ORIGIN__", string(origin)), "__MYGO_PLATFORM__", string(p))
}
func (g *Gateway) serveShortcutPreferences(w http.ResponseWriter, r *http.Request) {
	store := g.shortcutPreferences()
	if store == nil || r.Method != "POST" || r.Header.Get("Origin") != g.browserOrigin || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Forbidden", 403)
		return
	}
	var req struct {
		Operation   string          `json:"operation"`
		Definitions json.RawMessage `json:"definitions"`
		Edit        json.RawMessage `json:"edit"`
		Revision    string          `json:"revision"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		http.Error(w, "Invalid request", 400)
		return
	}
	owner := "product:" + g.browserOrigin
	var result any
	switch req.Operation {
	case "get":
		defs, err := shortcutprefs.ParseDefinitions(req.Definitions)
		if err != nil {
			http.Error(w, "Invalid catalog", 400)
			return
		}
		result = store.SetDefinitions(owner, defs)
	case "snapshot":
		result = store.Get()
	case "edit":
		edit, err := shortcutprefs.ParseEdit(req.Edit)
		if err != nil {
			http.Error(w, "Invalid edit", 400)
			return
		}
		result = store.Apply(owner, req.Revision, []shortcutprefs.Edit{edit})
	default:
		http.Error(w, "Unsupported operation", 400)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
