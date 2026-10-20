package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/config"
	"whaleisle/internal/desktop/pet"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
	"whaleisle/internal/ui/theme"
)

// twoProviderStore describes two api-key namespaces: one whose environment
// variable is missing and one whose variable is present.
func twoProviderStore(t *testing.T) *store.Store {
	t.Helper()
	schema := json.RawMessage(`{"uid":1,"refs":{"1":{"type":"object","dict":{"apiKey":2,"baseURL":2,"retries":3}},"2":{"type":"string"},"3":{"type":"number"}}}`)
	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://127.0.0.1:1"}, HTTP: &http.Client{Transport: galleryTransport(func(r *http.Request) (*http.Response, error) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		rec := httptest.NewRecorder()
		value := map[string]any{"writable": true, "namespaces": []map[string]any{
			{"ns": "llm-deepseek", "revision": 3, "schema": schema, "value": map[string]any{}},
			{"ns": "llm-pi-ai", "revision": 2, "schema": schema, "value": map[string]any{"apiKey": "sk-test-not-real"}},
		}}
		_ = json.NewEncoder(rec).Encode(map[string]any{"type": "server-response", "rpcId": req["rpcId"], "result": map[string]any{"ok": true, "value": value}})
		return rec.Result(), nil
	})}}, nil)
	s.RefreshDescribe(context.Background())
	deadline := time.Now().Add(2 * time.Second)
	for s.Snapshot().Loading && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(s.Snapshot().Namespaces) != 2 {
		t.Fatalf("provider fixture = %+v", s.Snapshot())
	}
	return s
}

type galleryTransport func(*http.Request) (*http.Response, error)

func (f galleryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func galleryEnv(namespace string) ([]string, bool) {
	if namespace == "llm-deepseek" {
		return []string{"DEEPSEEK_API_KEY"}, false
	}
	return []string{"PI_AI_API_KEY"}, true
}

func modelBodyTester(t *testing.T, s *store.Store, namespace string) *ui.Tester {
	t.Helper()
	return ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			modelNamespaceBody(c, Model{Store: s, Models: &ModelsAccess{ProviderEnv: galleryEnv}}, namespace)
		})
	}, 780, 900)
}

func TestModelCardShowsTheMissingEnvironmentHint(t *testing.T) {
	s := twoProviderStore(t)
	hint := fmt.Sprintf(i18n.T(i18n.KeySettingsModelsEnvHint), "DEEPSEEK_API_KEY")
	missing := modelBodyTester(t, s, "llm-deepseek")
	if !missing.HasText(hint) {
		t.Fatalf("missing env hint: %v", missing.Texts())
	}
	if !missing.HasText(i18n.T(i18n.KeySettingsModelsAPIKey)) {
		t.Fatalf("api key row missing: %v", missing.Texts())
	}
	if !missing.HasText(i18n.T(i18n.KeySettingsModelsCustomize)) {
		t.Fatalf("custom settings disclosure missing: %v", missing.Texts())
	}
	present := modelBodyTester(t, s, "llm-pi-ai")
	if present.HasText(hint) || present.HasText(fmt.Sprintf(i18n.T(i18n.KeySettingsModelsEnvHint), "PI_AI_API_KEY")) {
		t.Fatalf("detected environment still warns: %v", present.Texts())
	}
}

func TestModelCardDraftsAndCancelStayInsideOneNamespace(t *testing.T) {
	s := twoProviderStore(t)
	other := s.BindNamespace("llm-pi-ai")
	if other == nil {
		t.Fatal("missing second namespace")
	}
	other.SetDraft("baseURL", "https://fixture.invalid")
	tt := modelBodyTester(t, s, "llm-deepseek")
	cancel := i18n.T(i18n.KeySettingsWhaleCancel)
	if err := tt.Click(i18n.T(i18n.KeySettingsModelsAPIKey)); err != nil {
		t.Fatal(err)
	}
	tt.Type("sk-typed-not-real")
	if err := tt.Click(cancel); err != nil {
		t.Fatalf("cancel action missing: %v", err)
	}
	if snap := s.BindNamespace("llm-deepseek").Snapshot(); snap.Dirty {
		t.Fatalf("cancel kept the card draft dirty: %+v", snap.Drafts)
	}
	if snap := other.Snapshot(); !snap.Dirty || snap.Drafts["baseURL"] != "https://fixture.invalid" {
		t.Fatalf("cancel touched another namespace: %+v", snap)
	}
}

// TestSettingsGalleryWhaleAndModels exports the two Pass 3 comparison shots.
func TestSettingsGalleryWhaleAndModels(t *testing.T) {
	dir := os.Getenv("SETTINGS_GALLERY_DIR")
	if dir == "" {
		t.Skip("set SETTINGS_GALLERY_DIR to export the settings gallery")
	}
	rec := &whaleRecorder{}
	petShot := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			petPane(c, theme.Resolve("").Active(false), &pet.Companion{Enabled: true, Behavior: config.DefaultPetBehavior()},
				ui.RGB(20, 20, 20), ui.RGB(120, 120, 120), rec.access(whaleFixture()))
		})
	}, 1000, 800)
	writeGalleryPNG(t, dir, "ours-settings-pet.png", petShot)

	s := twoProviderStore(t)
	fg, muted := ui.RGB(20, 20, 20), ui.RGB(120, 120, 120)
	modelShot := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() {
			ui.Column(c).Gap(14).Children(func() {
				galleryModelCard(c, s, "llm-deepseek", fg, muted)
				galleryModelCard(c, s, "llm-pi-ai", fg, muted)
				modelAddProviderCard(c, Model{}, &ModelsAccess{}, fg, muted)
			})
		})
	}, 1000, 800)
	writeGalleryPNG(t, dir, "ours-settings-models.png", modelShot)
}

// galleryModelCard is the provider card with its body expanded, the way the
// reference shows the card under edit.
func galleryModelCard(c *ui.Context, s *store.Store, namespace string, fg, muted ui.Color) {
	ui.Column(c).FillWidth().MinWidth(0).Gap(8).Padding(12, 16).Radius(12).
		Border(1, c.Theme().Border).Background(c.Theme().Surface).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				ui.Text(c, modelProviderTitle(namespace)).FontSize(theme.InterfaceSize(14)).Bold().TextColor(fg)
				ui.Text(c, namespace).FontSize(theme.InterfaceSize(12)).TextColor(muted)
			})
		})
		modelNamespaceBody(c, Model{Store: s, Models: &ModelsAccess{ProviderEnv: galleryEnv}}, namespace)
	})
}

func writeGalleryPNG(t *testing.T, dir, name string, tt *ui.Tester) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
