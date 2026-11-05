package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"whaleisle/internal/desktop/credentialrefs"
	"whaleisle/internal/dsh"
	store "whaleisle/internal/store/settings"
	"whaleisle/internal/ui/i18n"
)

// In-process transport only: no listener, Host, real environment or credentials.
type capabilitiesUIFixture struct {
	mu        sync.Mutex
	namespace map[string]any
	directory []map[string]any
	states    map[string]map[string]any
	missing   map[string]bool
	writes    []string
}

func newCapabilitiesUIFixture() *capabilitiesUIFixture {
	schema := json.RawMessage(`{"type":"object","dict":{"providers":{"type":"dict","inner":{"type":"object","dict":{"apiKeyEnv":{"type":"string","meta":{"role":"credential-ref"}},"baseURL":{"type":"string"},"models":{"type":"array","inner":{"type":"object","dict":{"id":{"type":"string"}}}}}}}}}`)
	h := &capabilitiesUIFixture{namespace: map[string]any{"ns": "llm-pi-ai", "revision": float64(7), "schema": schema, "value": map[string]any{"providers": map[string]any{}}, "user": map[string]any{}, "base": map[string]any{}}, states: map[string]map[string]any{}}
	add := func(route string, auth any, endpoint any, env []string) {
		p := map[string]any{"provider": route, "displayName": route, "settingsNs": "llm-pi-ai", "settingsPath": []string{"providers", route}}
		if auth != nil {
			p["nativeAuth"] = auth
		}
		if endpoint != nil {
			p["defaultBaseURL"] = endpoint
		}
		if env != nil {
			p["baseURLEnv"] = env
		}
		h.directory = append(h.directory, p)
	}
	add("openai", map[string]any{"env": []string{"OPENAI_API_KEY"}, "oauth": false}, "https://api.openai.com/v1", nil)
	add("azure", map[string]any{"env": []string{"AZURE_OPENAI_API_KEY", "AZURE_OPENAI_BASE_URL"}, "oauth": false}, nil, []string{"AZURE_OPENAI_BASE_URL", "AZURE_OPENAI_RESOURCE_NAME"})
	add("anthropic", map[string]any{"env": []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_OAUTH_TOKEN", "ANTHROPIC_API_KEY"}, "oauth": true}, "https://api.anthropic.com", nil)
	add("openai-codex", map[string]any{"env": []string{}, "oauth": true}, "https://chatgpt.com/backend-api", nil)
	add("custom", nil, nil, nil)
	return h
}

func (h *capabilitiesUIFixture) Call(ctx context.Context, method string, args, out any) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	a, _ := args.(map[string]any)
	var value any
	switch method {
	case "settings/describe":
		value = map[string]any{"writable": true, "namespaces": []any{h.namespace}}
	case "llm/listConfigurableProviders":
		value = h.directory
	case "llm/listProviders":
		value = []any{}
	case "credentials/describe":
		states := map[string]any{}
		for _, ref := range a["refs"].([]string) {
			if h.missing[ref] {
				continue
			}
			state := h.states[ref]
			if state == nil {
				state = map[string]any{"configured": false, "writable": true}
			}
			states[ref] = state
		}
		value = states
	case "settings/mutate":
		if a["expectedRevision"] != h.namespace["revision"] {
			return fmt.Errorf("fixture revision conflict")
		}
		for _, op := range a["ops"].([]map[string]any) {
			path := op["path"].([]string)
			node := h.namespace["value"].(map[string]any)
			for _, key := range path[:len(path)-1] {
				child, _ := node[key].(map[string]any)
				if child == nil {
					child = map[string]any{}
					node[key] = child
				}
				node = child
			}
			node[path[len(path)-1]] = op["value"]
		}
		h.namespace["revision"] = h.namespace["revision"].(float64) + 1
		h.writes = append(h.writes, method)
		value = h.namespace
	case "credentials/set":
		h.states[a["ref"].(string)] = map[string]any{"configured": true, "writable": true, "source": "file"}
		h.writes = append(h.writes, method)
	default:
		return fmt.Errorf("unexpected fixture method %s", method)
	}
	if out != nil {
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}
		return json.Unmarshal(b, out)
	}
	return nil
}

func (h *capabilitiesUIFixture) settings(t *testing.T) *store.Store {
	t.Helper()
	s := store.New()
	s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: galleryTransport(func(r *http.Request) (*http.Response, error) {
		var req struct {
			Method  string `json:"method"`
			RPCID   any    `json:"rpcId"`
			Payload struct {
				Args map[string]any `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
		args := req.Payload.Args
		if raw, ok := args["ops"].([]any); ok {
			ops := make([]map[string]any, 0, len(raw))
			for _, item := range raw {
				op := item.(map[string]any)
				path := []string{}
				for _, part := range op["path"].([]any) {
					path = append(path, part.(string))
				}
				op["path"] = path
				ops = append(ops, op)
			}
			args["ops"] = ops
		}
		var value any
		if err := h.Call(r.Context(), req.Method, args, &value); err != nil {
			return nil, err
		}
		rec := httptest.NewRecorder()
		_ = json.NewEncoder(rec).Encode(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": value}})
		return rec.Result(), nil
	})}}, nil)
	s.RefreshDescribe(context.Background())
	s.RefreshProviderDirectory(context.Background())
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		_, loading, message := s.ConfigurableProviders()
		if !s.Snapshot().Loading && !loading {
			if message != "" || s.Snapshot().Err != "" || len(s.Snapshot().Namespaces) != 1 {
				t.Fatalf("fixture not loaded: %s %+v", message, s.Snapshot())
			}
			return s
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("fixture settings read did not settle")
	return s
}

func capabilitiesUILocale(t *testing.T) {
	t.Helper()
	old := i18n.Locale()
	i18n.SetLocale(i18n.LocaleEN)
	t.Cleanup(func() { i18n.SetLocale(old) })
}

func capabilitiesUIController(t *testing.T, h *capabilitiesUIFixture, route string) *credentialrefs.Controller {
	t.Helper()
	c := new(credentialrefs.Controller)
	c.Attach(h)
	c.Refresh(context.Background())
	for _, row := range c.Snapshot().Rows {
		if row.Provider == route {
			c.Select(row.ID)
			return c
		}
	}
	t.Fatalf("fixture route %s missing: %+v", route, c.Snapshot())
	return c
}

func capabilitiesUIAccess(c *credentialrefs.Controller) *ReferencedCredentialsAccess {
	return &ReferencedCredentialsAccess{Snapshot: c.Snapshot(), OnRefresh: func() { c.Refresh(context.Background()) }, OnSelect: c.Select, OnEdit: c.Edit, OnClear: c.StageClear, OnSave: func(ticket uint64) { c.Save(context.Background(), ticket) }}
}

func capabilitiesUIFormAccess(c *credentialrefs.Controller) *ReferencedCredentialsAccess {
	a := capabilitiesUIAccess(c)
	a.OnSaveProvider = func(ticket uint64, binding *store.NamespaceBinding) {
		c.SaveWithSettings(context.Background(), ticket, binding.SaveProviderSettings)
	}
	return a
}

func TestProviderCapabilitiesNativeAuthenticationControls(t *testing.T) {
	capabilitiesUILocale(t)
	for _, route := range []string{"openai", "azure", "anthropic", "openai-codex", "custom"} {
		t.Run(route, func(t *testing.T) {
			h := newCapabilitiesUIFixture()
			h.states["ANTHROPIC_OAUTH_TOKEN"] = map[string]any{"configured": true, "writable": false, "source": "environment"}
			controller := capabilitiesUIController(t, h, route)
			tt := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
					referencedCredentialsPane(c, capabilitiesUIAccess(controller), "llm-pi-ai", c.Theme().Text, c.Theme().TextMuted)
				})
			}, 780, 900)
			if route == "openai-codex" {
				if _, ok := tt.Find("API Key"); ok || tt.HasText("Save this key") || !tt.HasText("supports an existing saved sign-in") {
					t.Fatalf("OAuth-only route offered key writes: %v", tt.Texts())
				}
				return
			}
			if route == "anthropic" && (!tt.HasText("Env var detected.") || tt.HasText("ANTHROPIC_OAUTH_TOKEN")) {
				t.Fatalf("hidden native credential presence was lost or exposed: %v", tt.Texts())
			}
			if route == "azure" && (tt.HasText("AZURE_OPENAI_BASE_URL") || !tt.HasText("AZURE_OPENAI_API_KEY")) {
				t.Fatal("endpoint reference was treated as native key help")
			}
			if err := tt.Click("API Key"); err != nil {
				t.Fatal(err)
			}
			tt.Type("synthetic-ui-key")
			if err := tt.Click("Save this key"); err != nil {
				t.Fatal(err)
			}
			if len(h.writes) != 2 || h.writes[0] != "settings/mutate" || h.writes[1] != "credentials/set" || controller.Snapshot().Draft != "" || !tt.HasText("Configured") {
				t.Fatalf("native key input/save failed: %v %+v", h.writes, controller.Snapshot())
			}
		})
	}
}

func TestProviderCapabilitiesNativeMissingAndExplicitReference(t *testing.T) {
	capabilitiesUILocale(t)
	h := newCapabilitiesUIFixture()
	h.namespace["value"].(map[string]any)["providers"].(map[string]any)["openai-codex"] = map[string]any{"apiKeyEnv": "CHOSEN_KEY"}
	h.missing = map[string]bool{"CHOSEN_KEY": true}
	controller := capabilitiesUIController(t, h, "openai-codex")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			referencedCredentialsPane(c, capabilitiesUIAccess(controller), "llm-pi-ai", c.Theme().Text, c.Theme().TextMuted)
		})
	}, 780, 900)
	if !tt.HasText("Target reference: CHOSEN_KEY") || !tt.HasText("State unknown") || tt.HasText("no API key is accepted") {
		t.Fatalf("explicit/unknown reference was not respected: %v", tt.Texts())
	}
	if err := tt.SetValue("API Key", "must-not-write"); err == nil || len(h.writes) != 0 {
		t.Fatal("unknown writable state admitted an edit")
	}
	delete(h.missing, "CHOSEN_KEY")
	if err := tt.Click("Refresh references and writable state"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("API Key"); err != nil {
		t.Fatal(err)
	}
	tt.Type("synthetic-explicit-key")
	if err := tt.Click("Save this key"); err != nil {
		t.Fatal(err)
	}
	if len(h.writes) != 1 || h.writes[0] != "credentials/set" {
		t.Fatal("explicit reference was rebound or remained unavailable")
	}
}

func TestProviderCapabilitiesNativeUnknownPresenceIsNotMissing(t *testing.T) {
	capabilitiesUILocale(t)
	h := newCapabilitiesUIFixture()
	h.missing = map[string]bool{"OPENAI_API_KEY": true}
	controller := capabilitiesUIController(t, h, "openai")
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Gap(10).Children(func() {
			referencedCredentialsPane(c, capabilitiesUIAccess(controller), "llm-pi-ai", c.Theme().Text, c.Theme().TextMuted)
		})
	}, 780, 900)
	if !tt.HasText("Environment credential status has not been confirmed.") || tt.HasText("No environment variable detected") {
		t.Fatalf("unknown native check became missing: %v", tt.Texts())
	}
	if err := tt.SetValue("API Key", "must-not-write"); err == nil || len(h.writes) != 0 {
		t.Fatal("unknown reference state admitted a key write")
	}
}

func TestProviderCapabilitiesNativeEndpointDraftAndEmptyAdoption(t *testing.T) {
	capabilitiesUILocale(t)
	h := newCapabilitiesUIFixture()
	h.namespace["value"].(map[string]any)["providers"].(map[string]any)["openai"] = map[string]any{}
	s := h.settings(t)
	tt := ui.NewTester(func(c *ui.Context) {
		ui.Scroll(c).Fill().Padding(20).Children(func() { modelNamespaceBody(c, Model{Store: s, Models: &ModelsAccess{}}, "llm-pi-ai") })
	}, 900, 1200)
	if !tt.HasText("Inherited endpoint: https://api.openai.com/v1") {
		t.Fatalf("published default endpoint not consumed: %v", tt.Texts())
	}
	if err := tt.SetValue("API address", "https://custom.invalid/v1"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Current endpoint: https://custom.invalid/v1") || tt.HasText("Inherited endpoint:") {
		t.Fatal("explicit draft endpoint lost precedence")
	}
	if err := tt.SetValue("API address", ""); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Inherited endpoint: https://api.openai.com/v1") {
		t.Fatal("clearing endpoint did not restore published inheritance")
	}
	editing, adding := "", false
	controller := capabilitiesUIController(t, h, "openai-codex")
	add := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().Padding(20).Children(func() {
			modelAddProviderCard(c, Model{Store: s, ReferencedCredentials: capabilitiesUIFormAccess(controller)}, &ModelsAccess{}, &editing, &adding, c.Theme().Text, c.Theme().TextMuted)
		})
	}, 900, 1000)
	if err := add.Click(strings.TrimSpace(strings.TrimPrefix(i18n.T(i18n.KeySettingsModelsAddProvider), "+"))); err != nil {
		t.Fatal(err)
	}
	if err := add.Click("Choose catalog provider"); err != nil {
		t.Fatal(err)
	}
	if err := add.Click("openai-codex"); err != nil {
		t.Fatalf("%v; controls: %v", err, add.Texts())
	}
	if err := add.Click(i18n.T(i18n.KeySettingsModelsCustomize)); err != nil {
		t.Fatal(err)
	}
	if add.HasText("API Key") || !add.HasText("Inherited endpoint: https://chatgpt.com/backend-api") {
		t.Fatal("catalog editor lost keyless/endpoint metadata", add.Texts())
	}
	if len(h.writes) != 0 || !adding {
		t.Fatal("catalog selection wrote or left the new-provider card")
	}
	if err := add.Click("Save"); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); adding && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
		add.Frame()
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	profiles := h.namespace["value"].(map[string]any)["providers"].(map[string]any)
	if profile, ok := profiles["openai-codex"].(map[string]any); !ok || len(profile) != 0 || len(h.writes) != 1 || h.writes[0] != "settings/mutate" || adding {
		t.Fatalf("empty profile save bound a key, failed to close, or performed an extra write: %+v writes=%v adding=%v", profiles, h.writes, adding)
	}
}

func TestProviderCapabilitiesNativeNormalAndNarrowGallery(t *testing.T) {
	capabilitiesUILocale(t)
	h := newCapabilitiesUIFixture()
	h.namespace["value"].(map[string]any)["providers"].(map[string]any)["azure"] = map[string]any{}
	s := h.settings(t)
	controller := capabilitiesUIController(t, h, "azure")
	i18n.SetLocale(i18n.LocaleZH)
	for _, width := range []int{1000, 400} {
		tt := ui.NewTester(func(c *ui.Context) {
			ui.Scroll(c).Fill().Padding(20).Children(func() {
				modelsPane(c, Model{Store: s, Models: &ModelsAccess{}, ReferencedCredentials: capabilitiesUIFormAccess(controller)}, c.Theme().Text, c.Theme().TextMuted)
			})
		}, width, 1000)
		if err := tt.Click("编辑 azure"); err != nil {
			t.Fatal(err)
		}
		if err := tt.Click(i18n.T(i18n.KeySettingsModelsCustomize)); err != nil {
			t.Fatal(err)
		}
		if !tt.HasText("端点由 Host 环境配置：AZURE_OPENAI_BASE_URL, AZURE_OPENAI_RESOURCE_NAME") || !tt.HasText(fmt.Sprintf(i18n.T(i18n.KeySettingsModelsEnvHint), "AZURE_OPENAI_API_KEY")) {
			t.Fatalf("native environment endpoint/auth metadata missing at width %d: %v", width, tt.Texts())
		}
		if dir := os.Getenv("PROVIDER_CAPABILITY_GALLERY_DIR"); dir != "" {
			writeGalleryPNG(t, dir, fmt.Sprintf("provider-capabilities-native-%d.png", width), tt)
		}
	}
}
