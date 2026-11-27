package pluginslots

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/shortcutprefs"
)

func TestShortcutPreferencesWaitLongPollWakesOnChange(t *testing.T) {
	if strings.Contains(shortcutProviderScript, "setInterval(") {
		t.Fatal("shortcut_provider.js still uses unconditional setInterval polling")
	}

	var raw []byte
	prefs := shortcutprefs.New("desktop:windows", func() ([]byte, error) { return raw, nil }, func(b []byte) error { raw = b; return nil })
	g := &Gateway{browserOrigin: "http://127.0.0.1:1234"}
	g.SetShortcutPreferences(prefs)

	initial := prefs.Get()
	req := httptest.NewRequest("POST", shortcutPreferencesPath, strings.NewReader(`{"operation":"wait","since":`+jsonUint(initial.Sequence)+`}`))
	req.Header.Set("Origin", g.browserOrigin)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		g.serveShortcutPreferences(rec, req)
		close(done)
	}()

	time.Sleep(25 * time.Millisecond)
	prefs.SetDefinitions("product:"+g.browserOrigin, []shortcutprefs.Definition{
		{ID: "cmd.open", Defaults: map[string]shortcutprefs.Binding{"desktop:windows": {Code: "KeyK", Modifiers: []string{"control"}}}},
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("long-poll wait did not wake on store change")
	}
	if rec.Code != 200 {
		t.Fatalf("unexpected wait status %d: %s", rec.Code, rec.Body.String())
	}
	var got shortcutprefs.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Sequence <= initial.Sequence {
		t.Fatalf("expected newer sequence after change: got %d <= %d", got.Sequence, initial.Sequence)
	}
}

func jsonUint(v uint64) string {
	b, _ := json.Marshal(v)
	return string(b)
}
