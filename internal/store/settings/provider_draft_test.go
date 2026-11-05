package settings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"whaleisle/internal/dsh"
)

func TestProviderCardDraftsStayIsolatedAndLeaveRetiresCallbacks(t *testing.T) {
	writes := 0
	accepted := json.RawMessage(`{"providers":{"alpha":{"baseURL":"https://alpha.new"},"beta":{"baseURL":"https://beta.saved"}}}`)
	client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://provider.fixture.invalid"}, HTTP: &http.Client{Transport: namespaceTransport(func(r *http.Request) (*http.Response, error) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		writes++
		args := request["payload"].(map[string]any)["args"].(map[string]any)
		if request["method"] != "settings/mutate" || args["expectedRevision"] != float64(7) {
			t.Fatalf("wrong card save: %v", request)
		}
		ops := args["ops"].([]any)
		if len(ops) != 1 {
			t.Fatalf("save included other card fields: %v", ops)
		}
		op := ops[0].(map[string]any)
		path := op["path"].([]any)
		if len(path) != 3 || path[0] != "providers" || path[1] != "alpha" || path[2] != "baseURL" || op["value"] != "https://alpha.new" {
			t.Fatalf("save crossed provider path: %v", op)
		}
		w := httptest.NewRecorder()
		rpcOK(w, request["rpcId"], map[string]any{"ns": "llm-pi-ai", "revision": 8, "value": accepted, "user": accepted})
		return w.Result(), nil
	})}}
	s := New()
	s.Attach(client, nil)
	s.describedClient, s.writable = client, true
	initial := json.RawMessage(`{"providers":{"alpha":{"baseURL":"https://alpha.saved"},"beta":{"baseURL":"https://beta.saved"}}}`)
	s.namespaces = []Namespace{{NS: "llm-pi-ai", Revision: 7, Fields: []Field{{Key: "providers", Kind: FieldJSON}}, Value: initial, User: initial}}
	a := s.BindProviderEditor("llm-pi-ai", "alpha", []string{"providers", "alpha"})
	b := s.BindProviderEditor("llm-pi-ai", "beta", []string{"providers", "beta"})
	b.SetDraft("providers", `{"alpha":{"baseURL":"https://alpha.saved"},"beta":{"baseURL":"https://beta.draft"}}`)
	a.SetDraft("providers", `{"alpha":{"baseURL":"https://alpha.new"},"beta":{"baseURL":"https://must-not-write"}}`)
	if _, err := a.SaveProviderSettings(context.Background()); err != nil {
		t.Fatal(err)
	}
	if a.Snapshot().Dirty || !b.Snapshot().Dirty || s.BindNamespace("llm-pi-ai").Snapshot().Dirty || writes != 1 {
		t.Fatal("card save acknowledged another editor")
	}
	s.LeaveProviderForms()
	b.SetDraft("providers", `{"beta":{"baseURL":"https://late"}}`)
	if _, err := b.SaveProviderSettings(context.Background()); !errors.Is(err, ErrSettingsNotReady) || b.Snapshot().ClientReady || writes != 1 {
		t.Fatal("a departed card kept write authority", err)
	}
	current := s.BindProviderEditor("llm-pi-ai", "beta", []string{"providers", "beta"}).Snapshot()
	if current.Dirty || current.Drafts["providers"] == b.Snapshot().Drafts["providers"] || !sameJSON(current.Namespaces[0].Value, accepted) {
		t.Fatal("reopened card inherited the departed draft")
	}
}
