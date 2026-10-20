package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

// draftCard builds a store with two provider cards described, without a Host.
func draftCard() *Store {
	s := New()
	s.namespaces = []Namespace{
		{NS: "llm-deepseek", Fields: []Field{{Key: "apiKey", Kind: FieldString}}, Value: json.RawMessage(`{"apiKey":""}`)},
		{NS: "llm-openai", Fields: []Field{{Key: "apiKey", Kind: FieldString}}, Value: json.RawMessage(`{"apiKey":""}`)},
	}
	return s
}

func TestNamespaceDraftIsolatesCards(t *testing.T) {
	s := draftCard()
	s.Select("llm-deepseek")
	s.SetDraft("apiKey", "deepseek-key")
	s.Select("llm-openai")
	s.SetDraft("apiKey", "openai-key")

	deepseek, deepseekDirty := s.NamespaceDraft("llm-deepseek")
	openai, openaiDirty := s.NamespaceDraft("llm-openai")
	if !deepseekDirty || deepseek["apiKey"] != "deepseek-key" {
		t.Fatalf("the DeepSeek card reads %v dirty=%v", deepseek, deepseekDirty)
	}
	if !openaiDirty || openai["apiKey"] != "openai-key" {
		t.Fatalf("the OpenAI card reads %v dirty=%v", openai, openaiDirty)
	}
	if s.Snapshot().Drafts["apiKey"] != "openai-key" {
		t.Fatal("the generic editor stopped following the selection")
	}
}

func TestNamespaceDraftReturnsACopy(t *testing.T) {
	s := draftCard()
	s.Select("llm-deepseek")
	s.SetDraft("apiKey", "deepseek-key")
	drafts, _ := s.NamespaceDraft("llm-deepseek")
	drafts["apiKey"] = "mutated"
	if got, _ := s.NamespaceDraft("llm-deepseek"); got["apiKey"] != "deepseek-key" {
		t.Fatalf("a caller mutated the store: %v", got)
	}
}

func TestNamespaceDraftUnknownNamespace(t *testing.T) {
	s := draftCard()
	if drafts, dirty := s.NamespaceDraft("llm-acme"); drafts != nil || dirty {
		t.Fatalf("unknown namespace reported drafts=%v dirty=%v", drafts, dirty)
	}
	var nilStore *Store
	if drafts, dirty := nilStore.NamespaceDraft("llm-deepseek"); drafts != nil || dirty {
		t.Fatalf("nil store reported drafts=%v dirty=%v", drafts, dirty)
	}
}

func TestSaveNamespaceRefusesWithoutAHost(t *testing.T) {
	s := draftCard()
	if err := s.SaveNamespace(context.Background(), "llm-deepseek"); !errors.Is(err, ErrSettingsNotReady) {
		t.Fatalf("err = %v, want ErrSettingsNotReady", err)
	}
	var nilStore *Store
	if err := nilStore.SaveNamespace(context.Background(), "llm-deepseek"); !errors.Is(err, ErrSettingsNotReady) {
		t.Fatalf("nil store err = %v, want ErrSettingsNotReady", err)
	}
}

func TestSaveNamespaceRefusesUnknownNamespace(t *testing.T) {
	s := draftCard()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://host.invalid"}}
	s.writable = true
	if err := s.SaveNamespace(context.Background(), "llm-acme"); !errors.Is(err, ErrNamespaceUnknown) {
		t.Fatalf("err = %v, want ErrNamespaceUnknown", err)
	}
}

func TestSaveNamespaceRefusesReadOnlyAndBusy(t *testing.T) {
	s := draftCard()
	s.client = &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://host.invalid"}}
	s.writable = false
	if err := s.SaveNamespace(context.Background(), "llm-deepseek"); !errors.Is(err, ErrSettingsReadOnly) {
		t.Fatalf("err = %v, want ErrSettingsReadOnly", err)
	}
	s.writable = true
	s.saving = true
	if err := s.SaveNamespace(context.Background(), "llm-deepseek"); !errors.Is(err, ErrSettingsBusy) {
		t.Fatalf("err = %v, want ErrSettingsBusy", err)
	}
}

// TestSaveNamespaceSavesOneCard drives a real save against a Host that
// describes two provider cards, and checks the other card keeps its draft.
func TestSaveNamespaceSavesOneCard(t *testing.T) {
	var mu sync.Mutex
	var writes []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch {
		case strings.HasSuffix(r.URL.Path, "/settings/describe"):
			rpcOK(w, body["rpcId"], map[string]any{
				"writable": true,
				"namespaces": []map[string]any{
					{
						"ns":           "llm-deepseek",
						"schema":       json.RawMessage(permissionSchema),
						"value":        map[string]any{"defaultPreset": "read-only"},
						"autoGenerate": true,
						"applies":      "live",
						"secrets":      []any{},
						"revision":     1.0,
					},
					{
						"ns":           "llm-openai",
						"schema":       json.RawMessage(permissionSchema),
						"value":        map[string]any{"defaultPreset": "read-only"},
						"autoGenerate": true,
						"applies":      "live",
						"secrets":      []any{},
						"revision":     4.0,
					},
				},
			})
		case strings.HasSuffix(r.URL.Path, "/settings/update"):
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			mu.Lock()
			writes = append(writes, args)
			mu.Unlock()
			ns := args["ns"].(string)
			value := map[string]any{"defaultPreset": "read-only"}
			for key, raw := range args["patch"].(map[string]any) {
				value[key] = raw
			}
			rpcOK(w, body["rpcId"], map[string]any{
				"ns":           ns,
				"schema":       json.RawMessage(permissionSchema),
				"value":        value,
				"autoGenerate": true,
				"applies":      "live",
				"secrets":      []any{},
				"revision":     2.0,
			})
		default:
			rpcOK(w, body["rpcId"], map[string]any{})
		}
	}))
	defer srv.Close()

	st := New()
	st.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}, nil)
	st.RefreshDescribe(context.Background())
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && len(st.Snapshot().Namespaces) < 2 {
		time.Sleep(10 * time.Millisecond)
	}
	if len(st.Snapshot().Namespaces) != 2 {
		t.Fatalf("describe did not land: %+v", st.Snapshot())
	}

	st.Select("llm-deepseek")
	st.SetDraft("defaultPreset", "workspace-write")
	st.Select("llm-openai")
	st.SetDraft("defaultPreset", "danger-full-access")

	if err := st.SaveNamespace(context.Background(), "llm-deepseek"); err != nil {
		t.Fatalf("SaveNamespace: %v", err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if st.Snapshot().LastAction == "settings saved" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if last := st.Snapshot().LastAction; last != "settings saved" {
		t.Fatalf("last action = %q, err = %q", last, st.Snapshot().Err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(writes) != 1 {
		t.Fatalf("wrote %d sections, want the one card", len(writes))
	}
	if ns := writes[0]["ns"]; ns != "llm-deepseek" {
		t.Fatalf("wrote %v, want llm-deepseek", ns)
	}
	patch := writes[0]["patch"].(map[string]any)
	if patch["defaultPreset"] != "workspace-write" {
		t.Fatalf("patch = %v", patch)
	}
	deepseek, deepseekDirty := st.NamespaceDraft("llm-deepseek")
	if deepseekDirty {
		t.Fatalf("the saved card is still dirty: %v", deepseek)
	}
	openai, openaiDirty := st.NamespaceDraft("llm-openai")
	if !openaiDirty || openai["defaultPreset"] != "danger-full-access" {
		t.Fatalf("the other card lost its draft: %v dirty=%v", openai, openaiDirty)
	}
	if selected := st.Snapshot().SelectedNS; selected != "llm-openai" {
		t.Fatalf("saving moved the rail selection to %q", selected)
	}
}
