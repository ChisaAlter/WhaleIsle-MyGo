package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"whaleisle/internal/dsh"
	"whaleisle/internal/store/storelocale"
)

// providerReceipt is the shape the Host answers a provider write with: the
// effective value carries every schema default the write never sent (context
// windows, compat flags, token budgets), and only the user layer holds exactly
// what was stored.
const providerReceiptValue = `{
	"providers": {
		"magpie": {
			"api": "openai-completions",
			"baseURL": "http://127.0.0.1:3425/v1",
			"displayName": "magpie",
			"models": [{"id": "auto/gpt-6-luna", "input": [], "compat": {"chatTemplateArgs": {}, "chatTemplateKwargs": {}}}],
			"compat": {"chatTemplateArgs": {}, "chatTemplateKwargs": {}},
			"defaultContextWindow": 262144,
			"defaultInput": ["text"],
			"defaultMaxTokens": 32768,
			"headers": {},
			"maxRequestImageBytes": 20971520,
			"modelOverrides": {},
			"requestImageMaxBytes": 1048576,
			"requestImagePixelBudget": 4194304,
			"streamIdleTimeoutMs": 300000,
			"thinkingBudgets": {}
		}
	}
}`

const providerReceiptUser = `{
	"providers": {
		"magpie": {
			"api": "openai-completions",
			"baseURL": "http://127.0.0.1:3425/v1",
			"displayName": "magpie",
			"models": [{"id": "auto/gpt-6-luna"}]
		}
	}
}`

func providerCreateStore(t *testing.T, user string) (*Store, *NamespaceBinding, *[]map[string]any) {
	t.Helper()
	sent := []map[string]any{}
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://provider.receipt.invalid"}, HTTP: &http.Client{Transport: namespaceTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		args := request["payload"].(map[string]any)["args"].(map[string]any)
		sent = append(sent, args)
		w := httptest.NewRecorder()
		rpcOK(w, request["rpcId"], map[string]any{
			"ns":       "llm-pi-ai",
			"revision": 1,
			"value":    json.RawMessage(providerReceiptValue),
			"base":     map[string]any{"providers": map[string]any{}},
			"user":     json.RawMessage(user),
		})
		return w.Result(), nil
	})}}
	s := New()
	s.Attach(client, nil)
	s.mu.Lock()
	s.describedClient, s.writable = client, true
	s.namespaces = []Namespace{{NS: "llm-pi-ai", Revision: 0, Fields: []Field{{Key: "providers", Kind: FieldJSON}}, Value: json.RawMessage(`{"providers":{}}`), User: json.RawMessage(`{}`)}}
	s.mu.Unlock()
	binding := s.BindProviderEditor("llm-pi-ai", "@custom", nil)
	if binding == nil {
		t.Fatal("no provider editor")
	}
	binding.SetDraft("providers", `{"magpie":{"api":"openai-completions","baseURL":"http://127.0.0.1:3425/v1","displayName":"magpie","models":[{"id":"auto/gpt-6-luna"}]}}`)
	return s, binding, &sent
}

// A Host that stores the provider and answers with its defaulted profile is an
// accepted creation. Reading the effective value as the receipt made this look
// like a refusal while the provider was already stored.
func TestProviderCreateAcceptedWhenReceiptFillsDefaults(t *testing.T) {
	s, binding, sent := providerCreateStore(t, providerReceiptUser)
	written, err := binding.SaveProviderSettings(context.Background())
	if err != nil {
		t.Fatalf("a stored provider was reported as refused: %v", err)
	}
	if len(*sent) != 1 || (*sent)[0]["expectedRevision"] != float64(0) {
		t.Fatalf("unexpected write: %v", *sent)
	}
	if written.Ns != "llm-pi-ai" || written.Revision != 1 {
		t.Fatalf("receipt was not kept: %+v", written)
	}
	if state := binding.Snapshot(); state.Dirty || state.Err != "" {
		t.Fatalf("accepted creation kept drafts or an error: %+v", state)
	}
	if !strings.Contains(string(s.Snapshot().Value.Namespaces[0].Value), "magpie") {
		t.Fatal("the stored profile was not adopted")
	}
}

// A receipt that does not hold the submitted provider is a refusal: the drafts
// stay, the sentence a page shows names the next step, and the Host's own words
// wait in the details instead of standing in for that sentence.
func TestProviderWriteRefusalKeepsDraftsAndNamesTheNextStep(t *testing.T) {
	_, binding, _ := providerCreateStore(t, `{}`)
	_, err := binding.SaveProviderSettings(context.Background())
	if err == nil {
		t.Fatal("a write whose receipt lacked the provider was reported as accepted")
	}
	sentence := storelocale.T(storelocale.KeyStoreSettingsWriteNotStored)
	if err.Error() != sentence {
		t.Fatalf("page sentence is not the localized one: %q", err.Error())
	}
	for _, leaked := range []string{"llm-pi-ai", "settings/mutate", "revision"} {
		if strings.Contains(err.Error(), leaked) {
			t.Fatalf("sentence carries internal text %q: %q", leaked, err.Error())
		}
	}
	state := binding.Snapshot()
	if !state.Dirty || !strings.Contains(state.Drafts["providers"], "magpie") {
		t.Fatal("a refused write dropped its drafts")
	}
	if state.ErrDetail == "" || !strings.Contains(state.ErrDetail, "llm-pi-ai") {
		t.Fatalf("the Host's own words did not reach the details: %q", state.ErrDetail)
	}
	// The provider list is drawn from this value, so a refusal must leave it
	// without the provider: the page cannot keep what the Host did not store.
	if value := state.Namespaces[0].Value; strings.Contains(string(value), "magpie") {
		t.Fatalf("a refused write reached the provider list: %s", value)
	}
}
