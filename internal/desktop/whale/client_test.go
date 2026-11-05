package whale

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// rpcRecorder serves the dsh-whale control plane and records what the client
// sent, so tests can assert both the parsed result and the exact payload.
type rpcRecorder struct {
	mu      sync.Mutex
	method  string
	payload map[string]any
	calls   int
	handler func(method string, payload map[string]any) any
}

func (r *rpcRecorder) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	var envelope struct {
		RPCID   string         `json:"rpcId"`
		Method  string         `json:"method"`
		Payload map[string]any `json:"payload"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	method := strings.TrimPrefix(req.URL.Path, "/dsh-whale/")
	r.mu.Lock()
	r.method, r.payload, r.calls = method, envelope.Payload, r.calls+1
	handler := r.handler
	r.mu.Unlock()
	value := handler(method, envelope.Payload)
	result, _ := json.Marshal(map[string]any{"ok": true, "value": value})
	response, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": envelope.RPCID, "result": json.RawMessage(result)})
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(response)
}

func (r *rpcRecorder) last() (string, map[string]any, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.method, r.payload, r.calls
}

func newTestClient(t *testing.T, handler func(method string, payload map[string]any) any) (*Client, *rpcRecorder) {
	t.Helper()
	rec := &rpcRecorder{handler: handler}
	srv := httptest.NewServer(rec)
	t.Cleanup(srv.Close)
	client, err := New(srv.URL, "dsh-auth=fixture")
	if err != nil {
		t.Fatal(err)
	}
	return client, rec
}

// catalogValue mirrors the plugin: stored settings spread flat plus the
// resources the page lists.
func catalogValue() map[string]any {
	return map[string]any{
		"name":                 "鲸鱼娘",
		"personality":          "genki",
		"userTitle":            "船长",
		"personaText":          "custom persona",
		"modelProvider":        "deepseek-official",
		"modelModel":           "deepseek-chat",
		"modelReasoningEffort": "high",
		"imDefault":            true,
		"sessionId":            "session-1",
		"personalityOptions":   []string{"natural", "genki", "tsundere", "poison"},
		"skills":               []map[string]any{{"name": "diary", "enabled": true}, {"name": "art", "enabled": false}},
		"memory":               "# memory",
		"homeDir":              "/home/user/.dsh/whale-home",
	}
}

func TestCatalogParsesFlatModelAndResources(t *testing.T) {
	client, rec := newTestClient(t, func(string, map[string]any) any { return catalogValue() })
	catalog, err := client.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if method, _, _ := rec.last(); method != "catalog" {
		t.Fatalf("method = %q", method)
	}
	if catalog.Name != "鲸鱼娘" || catalog.UserTitle != "船长" || catalog.Personality != "genki" {
		t.Fatalf("identity = %+v", catalog)
	}
	if catalog.ModelProvider != "deepseek-official" || catalog.ModelModel != "deepseek-chat" || catalog.ModelReasoningEffort != "high" {
		t.Fatalf("flat model = %+v", catalog)
	}
	if !catalog.IMDefault || catalog.SessionID != "session-1" || catalog.HomeDir == "" || catalog.Memory == "" {
		t.Fatalf("flags/resources = %+v", catalog)
	}
	if len(catalog.PersonalityOptions) != 4 || len(catalog.Skills) != 2 || catalog.Skills[1].Name != "art" || catalog.Skills[1].Enabled {
		t.Fatalf("resources = %+v / %+v", catalog.PersonalityOptions, catalog.Skills)
	}
}

func TestUpdateSendsOnlyPatchedFields(t *testing.T) {
	client, rec := newTestClient(t, func(method string, payload map[string]any) any {
		value := catalogValue()
		if name, ok := payload["name"].(string); ok {
			value["name"] = name
		}
		if model, ok := payload["model"].(map[string]any); ok {
			value["modelProvider"] = model["provider"]
			value["modelModel"] = model["model"]
		}
		return value
	})
	name, provider, model := "小鲸鱼", "deepseek-official", "deepseek-reasoner"
	updated, err := client.Update(context.Background(), Patch{Name: &name, Provider: &provider, Model: &model})
	if err != nil {
		t.Fatal(err)
	}
	method, payload, _ := rec.last()
	if method != "settings/update" {
		t.Fatalf("method = %q", method)
	}
	if payload["name"] != "小鲸鱼" {
		t.Fatalf("name not sent: %+v", payload)
	}
	for _, untouched := range []string{"userTitle", "personaText", "personality", "imDefault"} {
		if _, ok := payload[untouched]; ok {
			t.Fatalf("patch sent untouched field %q: %+v", untouched, payload)
		}
	}
	// The plugin's sanitiser reads the nested model object only.
	modelPayload, ok := payload["model"].(map[string]any)
	if !ok || modelPayload["provider"] != provider || modelPayload["model"] != model {
		t.Fatalf("model payload = %+v", payload["model"])
	}
	if _, ok := modelPayload["reasoningEffort"]; ok {
		t.Fatalf("patch sent an empty effort: %+v", modelPayload)
	}
	if updated.Name != "小鲸鱼" || updated.ModelModel != "deepseek-reasoner" {
		t.Fatalf("returned snapshot = %+v", updated)
	}
}

func TestUpdateRejectsEmptyPatchWithoutCallingHost(t *testing.T) {
	client, rec := newTestClient(t, func(string, map[string]any) any { return catalogValue() })
	if _, err := client.Update(context.Background(), Patch{}); err == nil {
		t.Fatal("empty patch accepted")
	}
	if _, _, calls := rec.last(); calls != 0 {
		t.Fatalf("empty patch still called the Host %d times", calls)
	}
}

func TestReplaceMemorySendsTextAndReportsLength(t *testing.T) {
	client, rec := newTestClient(t, func(method string, payload map[string]any) any {
		text, _ := payload["text"].(string)
		return map[string]any{"ok": true, "chars": len([]rune(text))}
	})
	chars, err := client.ReplaceMemory(context.Background(), "记得喝水")
	if err != nil {
		t.Fatal(err)
	}
	method, payload, _ := rec.last()
	if method != "memory/replace" || payload["text"] != "记得喝水" {
		t.Fatalf("sent %q %+v", method, payload)
	}
	if chars != 4 {
		t.Fatalf("chars = %d", chars)
	}
}

func TestReplaceMemoryRejectsOversizedTextLocally(t *testing.T) {
	client, rec := newTestClient(t, func(string, map[string]any) any {
		return map[string]any{"ok": true, "chars": 0}
	})
	if _, err := client.ReplaceMemory(context.Background(), strings.Repeat("a", memoryMaxUTF16+1)); err == nil {
		t.Fatal("oversized memory accepted")
	}
	if _, _, calls := rec.last(); calls != 0 {
		t.Fatalf("oversized memory still called the Host %d times", calls)
	}
}

func TestMemoryAndSkillReceiptsSurfacePluginReasons(t *testing.T) {
	client, _ := newTestClient(t, func(method string, _ map[string]any) any {
		switch method {
		case "memory/clear":
			return map[string]any{"ok": false, "error": "missing-home"}
		case "skills/toggle":
			return map[string]any{"ok": false, "error": "skill-not-found"}
		}
		return map[string]any{"ok": false}
	})
	if err := client.ClearMemory(context.Background()); err == nil || !strings.Contains(err.Error(), "missing-home") {
		t.Fatalf("clear error = %v", err)
	}
	if err := client.ToggleSkill(context.Background(), "art", true); err == nil || !strings.Contains(err.Error(), "skill-not-found") {
		t.Fatalf("toggle error = %v", err)
	}
	if err := client.ToggleSkill(context.Background(), "bad name", true); err == nil {
		t.Fatal("invalid skill name accepted")
	}
}

func TestToggleSkillSendsNameAndEnabled(t *testing.T) {
	client, rec := newTestClient(t, func(string, map[string]any) any {
		return map[string]any{"ok": true, "changed": true}
	})
	if err := client.ToggleSkill(context.Background(), " diary ", false); err != nil {
		t.Fatal(err)
	}
	method, payload, _ := rec.last()
	if method != "skills/toggle" || payload["name"] != "diary" || payload["enabled"] != false {
		t.Fatalf("sent %q %+v", method, payload)
	}
}

func TestEnvelopeErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		var envelope struct {
			RPCID string `json:"rpcId"`
		}
		_ = json.Unmarshal(body, &envelope)
		_, _ = w.Write([]byte(`{"type":"server-response","rpcId":"` + envelope.RPCID + `","result":{"ok":false,"error":{"code":"plugin-error","message":"catalog unavailable"}}}`))
	}))
	defer srv.Close()
	client, err := New(srv.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Catalog(context.Background())
	if err == nil || !strings.Contains(err.Error(), "catalog unavailable") {
		t.Fatalf("error = %v", err)
	}
}

func TestNewValidatesHostOrigin(t *testing.T) {
	for _, origin := range []string{"", "ftp://127.0.0.1:1", "http://user:pw@127.0.0.1:1", "http://127.0.0.1:1/path", "http://127.0.0.1:1?x=1", "http://127.0.0.1:1#frag", "127.0.0.1:1"} {
		if _, err := New(origin, ""); err == nil {
			t.Fatalf("origin %q accepted", origin)
		}
	}
	if _, err := New("http://127.0.0.1:8787/", ""); err != nil {
		t.Fatalf("trailing slash origin rejected: %v", err)
	}
}

func TestCatalogToleratesMissingResources(t *testing.T) {
	client, _ := newTestClient(t, func(string, map[string]any) any {
		return map[string]any{"name": "鲸鱼娘"}
	})
	catalog, err := client.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Skills == nil || catalog.PersonalityOptions == nil {
		t.Fatalf("missing lists decoded as nil: %+v", catalog)
	}
}

func TestDecodeCatalog(t *testing.T) {
	catalog, err := DecodeCatalog(json.RawMessage(`{"name":"鲸鱼娘","modelProvider":"p","modelModel":"m"}`))
	if err != nil || catalog.Name != "鲸鱼娘" || catalog.ModelProvider != "p" || catalog.ModelModel != "m" {
		t.Fatalf("decode = %+v / %v", catalog, err)
	}
}
