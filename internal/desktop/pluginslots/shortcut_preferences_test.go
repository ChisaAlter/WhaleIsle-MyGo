package pluginslots

import (
	"net/http/httptest"
	"strings"
	"testing"
	"whaleisle/internal/shortcutprefs"
)

func TestUnifiedShortcutGatewayContract(t *testing.T) {
	var raw []byte
	prefs := shortcutprefs.New("desktop:windows", func() ([]byte, error) { return raw, nil }, func(b []byte) error { raw = b; return nil })
	g := &Gateway{browserOrigin: "http://127.0.0.1:32123"}
	g.SetShortcutPreferences(prefs)
	body := `{"operation":"get","definitions":[{"id":"plugin.example","defaults":{"desktop:windows":{"code":"KeyJ","modifiers":["primary","alt"]}}}]}`
	call := func(origin, body string) int {
		r := httptest.NewRequest("POST", shortcutPreferencesPath, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		g.serveShortcutPreferences(w, r)
		return w.Code
	}
	if call("https://arbitrary.example", body) != 403 {
		t.Fatal("foreign origin admitted")
	}
	if call(g.browserOrigin, body) != 200 {
		t.Fatal("catalog rejected")
	}
	if call(g.browserOrigin, `{"operation":"rpc","method":"file/write"}`) != 400 {
		t.Fatal("arbitrary operation admitted")
	}
	if len(prefs.Effective("product:"+g.browserOrigin)) != 1 {
		t.Fatal("plugin catalog not registered")
	}
	script := g.ShortcutProviderScript()
	if strings.Contains(script, "__MYGO_") || strings.Contains(script, "window.dshDesktop") {
		t.Fatal("invalid capability script")
	}
	var absent *Gateway
	if absent.ShortcutProviderScript() != "" {
		t.Fatal("non-product preload")
	}
}
