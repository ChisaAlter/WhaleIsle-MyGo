package pluginslots

import (
	"encoding/json"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"whaleisle/internal/shortcutprefs"
)

func TestLegacyShortcutImportGateway(t *testing.T) {
	platform := runtime.GOOS
	if platform == "darwin" {
		platform = "macos"
	}
	var raw []byte
	prefs := shortcutprefs.New("desktop:"+platform, func() ([]byte, error) { return raw, nil }, func(b []byte) error { raw = b; return nil })
	g := &Gateway{browserOrigin: "http://127.0.0.1:1234"}
	g.SetShortcutPreferences(prefs)
	doc := `{"schemaVersion":2,"profiles":{"web:` + platform + `":{"plugin.dormant":{"code":"KeyJ","secondCode":"KeyK","modifiers":["control","alt"]}}}}`
	call := func(body, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", shortcutImportPath, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		g.serveShortcutImport(w, r)
		return w
	}
	preview := `{"operation":"preview","document":` + doc + `}`
	if call(preview, "https://foreign.test").Code != 403 {
		t.Fatal("foreign import")
	}
	response := call(preview, g.browserOrigin)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var plan shortcutprefs.ImportPreview
	if err := json.Unmarshal(response.Body.Bytes(), &plan); err != nil {
		t.Fatal(err)
	}
	if raw != nil {
		t.Fatal("preview wrote preferences")
	}
	body := `{"operation":"import","document":` + doc + `,"revision":"` + plan.Snapshot.Revision + `","replaceIds":[]}`
	response = call(body, g.browserOrigin)
	var result shortcutprefs.Result
	_ = json.Unmarshal(response.Body.Bytes(), &result)
	if result.Status != "saved" {
		t.Fatal(response.Body.String())
	}
	if result.Snapshot.Document.Profiles["desktop:"+platform]["plugin.dormant"].SecondCode != "KeyK" {
		t.Fatal("unknown double binding lost")
	}
	if call(`{"operation":"preview","document":`+doc+`,"sourceProfile":"desktop:windows"}`, g.browserOrigin).Code != 400 {
		t.Fatal("caller chose profile")
	}
}
