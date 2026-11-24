package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"

	"github.com/egoist/mygo/ui"
)

// The editor's Save works like the reference's: it is blocked only while the
// card cannot write, not until something was edited. With nothing edited it just
// closes the card.
func TestModelEditorSaveDoesNotWaitForAnEdit(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	s := twoProviderStore(t)
	closed := false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			modelNamespaceEditor(c, Model{Store: s, Models: &ModelsAccess{ProviderEnv: galleryEnv}}, "llm-deepseek", func() { closed = true })
		})
	}, 780, 900)
	if err := tt.Click(i18n.T(i18n.KeySettingsSave)); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); !closed && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
		tt.Frame()
	}
	if !closed {
		t.Fatalf("saving an untouched card did not close it: %v", tt.Texts())
	}
}

// "Add provider" does not wait for the open card to be closed: it takes over the
// page's one open card, as in the reference.
func TestAddProviderStaysAvailableWhileACardIsOpen(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	s := twoProviderStore(t)
	editing, adding := "deepseek-official", false
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			modelAddProviderCard(c, Model{Store: s}, &ModelsAccess{}, &editing, &adding, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 780, 600)
	label := strings.TrimSpace(strings.TrimPrefix(i18n.T(i18n.KeySettingsModelsAddProvider), "+"))
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if !adding || editing != "" {
		t.Fatalf("add provider did not take over the open card: adding=%v editing=%q", adding, editing)
	}
}

// A failed model-catalog read is written into the page: the sentence and its
// retry sit in one row above the provider cards, and the cards move down to make
// room. Nothing floats over the content.
func TestModelsCatalogFailureIsInThePage(t *testing.T) {
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(i18n.DefaultLocale) })
	schema := json.RawMessage(`{"uid":1,"refs":{"1":{"type":"object","dict":{"baseURL":2}},"2":{"type":"string"}}}`)
	var catalogReads atomic.Int32
	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: galleryTransport(func(r *http.Request) (*http.Response, error) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		result := map[string]any{"ok": true, "value": map[string]any{"writable": true, "namespaces": []map[string]any{{"ns": "llm-deepseek", "revision": 1, "schema": schema, "value": map[string]any{}}}}}
		switch req["method"] {
		case "session/modelCatalog":
			catalogReads.Add(1)
			result = map[string]any{"ok": false, "error": map[string]any{"message": "DeepSeek: Invalid URL"}}
		case "llm/listConfigurableProviders", "llm/listProviders":
			result = map[string]any{"ok": true, "value": []any{}}
		}
		rec := httptest.NewRecorder()
		_ = json.NewEncoder(rec).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": result})
		return rec.Result(), nil
	})}}, nil)
	s.RefreshDescribe(context.Background())
	for deadline := time.Now().Add(time.Second); s.Snapshot().Value.Loading && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		modelsPane(c, Model{Store: s, Models: &ModelsAccess{}}, c.Theme().Text, c.Theme().TextMuted)
	}, 780, 600)
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		if _, loading, _ := s.ConfigurableProviders(); !loading {
			break
		}
		time.Sleep(time.Millisecond)
		tt.Frame()
	}
	// Let the directory's own empty state settle before measuring the page.
	tt.Frame()
	tt.Frame()
	add := strings.TrimSpace(strings.TrimPrefix(i18n.T(i18n.KeySettingsModelsAddProvider), "+"))
	before, ok := tt.Find(add)
	if !ok {
		t.Fatalf("add provider row missing: %v", tt.Texts())
	}
	s.RefreshModelConfigurationCatalog(context.Background())
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		if _, loading, message := s.ModelConfigurationCatalog(); !loading && message != "" {
			break
		}
		time.Sleep(time.Millisecond)
	}
	tt.Frame()
	tt.Frame()
	_, _, message := s.ModelConfigurationCatalog()
	if message == "" {
		t.Fatal("fixture did not fail the catalog read")
	}
	if !tt.HasText("The model directory could not be read. Try again.") {
		t.Fatalf("the catalog failure is not in the page: %v", tt.Texts())
	}
	after, ok := tt.Find(add)
	if !ok || after.Y <= before.Y {
		t.Fatalf("the failure is not in the page's flow: add provider row %v -> %v (%v)", before, after, tt.Texts())
	}
	retry, ok := tt.Find("Retry")
	if !ok || retry.Y >= after.Y {
		t.Fatalf("the retry is not written with the failure above the cards: %v ok=%v (%v)", retry, ok, tt.Texts())
	}
	if !detailHolds(t, tt, message) {
		t.Fatalf("the catalog failure did not keep the store's words behind 详情: %v", tt.Texts())
	}
	reads := catalogReads.Load()
	if err := tt.Click("Retry"); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); catalogReads.Load() == reads && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
		tt.Frame()
	}
	if catalogReads.Load() == reads {
		t.Fatal("the in-page retry did not read the catalog again")
	}
}
