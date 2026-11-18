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
)

func describedDeepSeekToolsStore(t *testing.T, origin string) *Store {
	t.Helper()
	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: origin}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.describedClient = client
	st.writable = true
	st.namespaces = []Namespace{
		{NS: "unrelated", Revision: 2, Fields: []Field{{Key: "keep", Kind: FieldString}}, Value: json.RawMessage(`{"keep":"stored"}`)},
		{NS: DeepSeekToolsNamespace, Revision: 7, Fields: []Field{
			{Key: "baseURL", Kind: FieldString}, {Key: "apiKeyEnv", Kind: FieldString},
			{Key: "tts", Kind: FieldBool}, {Key: "asr", Kind: FieldBool}, {Key: "ocr", Kind: FieldBool},
			{Key: "voice", Kind: FieldEnum, EnumOptions: []string{"stella", "tide"}}, {Key: "filterMarkdown", Kind: FieldBool},
		}, Value: json.RawMessage(`{"baseURL":"https://api.deepseek.com/v1/tools","apiKeyEnv":"DEEPSEEK_API_KEY","tts":false,"asr":false,"ocr":false,"voice":"stella","filterMarkdown":true}`)},
	}
	st.mu.Unlock()
	return st
}

func TestDeepSeekToolsViewReadsTheDescribedNamespace(t *testing.T) {
	st := describedDeepSeekToolsStore(t, "http://host.invalid")
	view := st.SnapshotDeepSeekTools()
	if !view.Connected || !view.Present || !view.Writable {
		t.Fatalf("view = %+v, want a present writable namespace", view)
	}
	if view.BaseURL != "https://api.deepseek.com/v1/tools" || view.APIKeyEnv != "DEEPSEEK_API_KEY" || view.Voice != "stella" || !view.FilterMarkdown {
		t.Fatalf("stored values parsed wrong: %+v", view)
	}
	if view.TTS || view.ASR || view.OCR {
		t.Fatalf("new tools must read as off: %+v", view)
	}

	binding := st.DeepSeekToolsBinding()
	if binding == nil {
		t.Fatal("a described namespace must produce a binding")
	}
	binding.SetDraft("tts", "true")
	staged := st.SnapshotDeepSeekTools()
	if !staged.DirtyFields["tts"] || staged.Drafts["tts"] != "true" || staged.TTS {
		t.Fatalf("draft did not stage: %+v", staged)
	}
}

func TestDeepSeekToolsBindingWritesOnlyItsOwnNamespace(t *testing.T) {
	var calls atomic.Int32
	var method string
	var namespace string
	var patch map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		args := body["payload"].(map[string]any)["args"].(map[string]any)
		method = r.URL.Path
		namespace, _ = args["ns"].(string)
		patch, _ = args["patch"].(map[string]any)
		calls.Add(1)
		rpcOK(w, body["rpcId"], map[string]any{"ns": DeepSeekToolsNamespace, "revision": 8,
			"value": map[string]any{"baseURL": "https://api.deepseek.com/v1/tools", "apiKeyEnv": "DEEPSEEK_API_KEY", "tts": true, "asr": false, "ocr": false, "voice": "stella", "filterMarkdown": true}})
	}))
	defer srv.Close()

	st := describedDeepSeekToolsStore(t, srv.URL)

	binding := st.DeepSeekToolsBinding()
	binding.SetDraft("tts", "true")
	binding.Save(context.Background())

	deadline := time.Now().Add(3 * time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if !strings.HasSuffix(method, "/settings/update") || namespace != DeepSeekToolsNamespace {
		t.Fatalf("write hit %q for %q, want settings/update for %q", method, namespace, DeepSeekToolsNamespace)
	}
	if patch["tts"] != true || len(patch) != 1 {
		t.Fatalf("patch = %#v, want only the changed switch", patch)
	}
	deadline = time.Now().Add(3 * time.Second)
	for st.Snapshot().Saving && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	after := st.SnapshotDeepSeekTools()
	if after.Err != "" || after.TTS != true || after.DirtyFields["tts"] {
		t.Fatalf("saved namespace did not settle: %+v", after)
	}
}

func TestDeepSeekToolsViewReportsAnUnmountedPluginAsItsOwnState(t *testing.T) {
	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://host.invalid"}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.describedClient, st.writable = client, true
	st.namespaces = []Namespace{{NS: "general", Fields: []Field{{Key: "lang", Kind: FieldString}}, Value: json.RawMessage(`{"lang":"zh"}`)}}
	st.mu.Unlock()

	view := st.SnapshotDeepSeekTools()
	if !view.Connected || view.Present || view.Err != "" {
		t.Fatalf("view = %+v, want connected without the namespace and without an error", view)
	}
	if st.DeepSeekToolsBinding() != nil {
		t.Fatal("an undescribed namespace must not bind")
	}
}

func TestDeepSeekKeyStateStartsUnknownAndTracksDescribe(t *testing.T) {
	described := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/credentials/describe") {
			t.Errorf("unexpected method %s", r.URL.Path)
			return
		}
		described = true
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		rpcOK(w, body["rpcId"], map[string]any{"DEEPSEEK_API_KEY": map[string]any{"configured": true, "writable": true}})
	}))
	defer srv.Close()

	st := New()
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: srv.URL}}
	st.Attach(client, nil)
	st.mu.Lock()
	st.describedClient = client
	st.mu.Unlock()

	st.RefreshDeepSeekKey(context.Background(), "DEEPSEEK_API_KEY")
	deadline := time.Now().Add(3 * time.Second)
	for st.SnapshotDeepSeekKey().Loading && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	key := st.SnapshotDeepSeekKey()
	if !described || !key.Configured || !key.Writable || key.Err != "" || key.Ref != "DEEPSEEK_API_KEY" {
		t.Fatalf("key state = %+v described=%v", key, described)
	}
}
