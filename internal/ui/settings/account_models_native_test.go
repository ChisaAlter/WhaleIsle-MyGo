package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

func TestAccountModelsNativeEditsUseLivePerIDOverrides(t *testing.T) {
	previous := i18n.Locale()
	i18n.SetLocale("en")
	t.Cleanup(func() { i18n.SetLocale(previous) })
	schema := json.RawMessage(`{"type":"object","dict":{"models":{"type":"array","inner":{"type":"object","dict":{"id":{"type":"string"},"name":{"type":"string"},"contextWindow":{"type":"number"},"maxTokens":{"type":"number"},"inputModalities":{"type":"array","inner":{"type":"string"}}}}},"modelOverrides":{"type":"dict","inner":{"type":"object","dict":{}}}}}`)
	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: galleryTransport(func(r *http.Request) (*http.Response, error) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		var value any
		switch req["method"] {
		case "settings/describe":
			value = map[string]any{"writable": true, "namespaces": []any{map[string]any{"ns": "llm-deepseek-account", "schema": schema, "revision": 1, "value": map[string]any{}, "base": map[string]any{}}}}
		case "session/modelCatalog":
			value = map[string]any{"groups": []any{map[string]any{"id": "deepseek-account", "name": "Account", "models": []any{map[string]any{"id": "live-model", "name": "Live account model", "inputModalities": []string{"text", "image"}}, map[string]any{"id": "new-model", "name": "New account model", "inputModalities": []string{"text"}}}}}, "failures": []any{}}
		default:
			t.Errorf("unexpected model fixture RPC: %v", req["method"])
		}
		rec := httptest.NewRecorder()
		_ = json.NewEncoder(rec).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return rec.Result(), nil
	})}}, nil)
	s.RefreshDescribe(context.Background())
	s.RefreshModelConfigurationCatalog(context.Background())
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		_, loading, _ := s.ConfigurationModels("deepseek-account")
		if !s.Snapshot().Loading && !loading {
			break
		}
		time.Sleep(time.Millisecond)
	}
	binding := s.BindProviderEditor("llm-deepseek-account", "deepseek-account", nil)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			modelNamespaceBody(c, Model{Store: s, Models: &ModelsAccess{configurationBinding: binding}}, "llm-deepseek-account")
		})
	}, 900, 1300)
	if err := tt.Click(i18n.T(i18n.KeySettingsModelsCustomize)); err != nil {
		t.Fatal(err)
	}
	modelRows := 0
	for _, text := range tt.Texts() {
		if text == "Model ID" {
			modelRows++
		}
	}
	if modelRows != 2 || !tt.HasText("Model options") || tt.HasText("inputModalities") {
		t.Fatalf("live catalog was not consumed by native editor: %v", tt.Texts())
	}
	if err := tt.Click("Model options"); err != nil {
		t.Fatal(err)
	}
	if dir := os.Getenv("SETTINGS_GALLERY_DIR"); dir != "" {
		writeGalleryPNG(t, dir, "account-live-model-editor.png", tt)
	}
	if err := tt.Click("Image"); err != nil {
		t.Fatal(err)
	}
	state := binding.Snapshot()
	overrides := configurationObject(state.Drafts["modelOverrides"])
	if !state.Resets["models"] || configurationObjectValue(overrides["live-model"])["inputModalities"] == nil || overrides["new-model"] != nil {
		t.Fatalf("native edit failed to write minimal per-ID settings: %+v", state.Drafts)
	}
	if err := tt.Click("Add model"); err != nil {
		t.Fatal(err)
	}
	state = binding.Snapshot()
	if !configurationInvalid(state) {
		t.Fatal("unfinished new row did not block Save")
	}
	tt.Scroll(450, 650, 0, 1800)
	if err := tt.Click(i18n.T(i18n.KeySettingsWhaleCancel)); err != nil {
		t.Fatal(err)
	}
	state = binding.Snapshot()
	if state.Dirty || state.Drafts["@account-model-rows"] != "" {
		t.Fatal("cancel kept the account editing baseline")
	}
}
