package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestServedPluginBodyHasOneBoundFormAndShellOverride(t *testing.T) {
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	defer i18n.SetLocale(old)
	s := store.New()
	loaded := make(chan struct{}, 1)
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: hostThemeTestTransport(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["method"] != "settings/describe" {
			t.Errorf("render made unexpected RPC %v", body["method"])
		}
		ns := func(name, field string, value any) map[string]any {
			return map[string]any{"ns": name, "revision": 1, "value": map[string]any{field: value}, "schema": map[string]any{"uid": 2, "refs": map[string]any{"1": map[string]any{"type": "number"}, "2": map[string]any{"type": "object", "dict": map[string]any{field: 1}}}}, "secrets": []any{}}
		}
		w := httptest.NewRecorder()
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "server-response", "rpcId": body["rpcId"], "result": map[string]any{"ok": true, "value": map[string]any{"writable": true, "namespaces": []any{ns("agent-loop", "maxParallelToolCalls", 4), ns("pwsh-sandbox", "timeoutMs", 1000), ns("bash-sandbox", "maxOutputBytes", 500)}}}})
		return w.Result(), nil
	})}}, func() {
		if snapshot := s.Snapshot(); !snapshot.Loading && len(snapshot.Namespaces) == 3 {
			select {
			case loaded <- struct{}{}:
			default:
			}
		}
	})
	s.RefreshDescribe(context.Background())
	select {
	case <-loaded:
	case <-time.After(3 * time.Second):
		t.Fatal("served namespace fixture did not finish describing")
	}
	for _, key := range []string{"agent-loop", "shell"} {
		mounted := false
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().Padding(16).Children(func() { mounted = ServedPluginBody(c, Model{Store: s}, key) })
		}, 700, 600)
		if !mounted {
			t.Fatalf("served %s was missing", key)
		}
		count := 0
		for _, text := range tt.Texts() {
			if text == "Save" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("%s rendered %d Save forms: %v", key, count, tt.Texts())
		}
		if key == "shell" && (!tt.HasText("Foreground timeout") || tt.HasText("Per-stream output cap")) {
			t.Fatal("shell combined the active PowerShell override with Bash's form")
		}
	}
	if s.Snapshot().SelectedNS != "agent-loop" {
		t.Fatal("plugin form changed settings rail selection")
	}
	mounted := true
	_ = ui.NewTester(func(c *ui.Context) { mounted = ServedPluginBody(c, Model{Store: s}, "subagent") }, 500, 400)
	if mounted {
		t.Fatal("unserved plugin invented a configuration form")
	}
}
