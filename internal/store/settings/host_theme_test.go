package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
)

func TestThemeQueueCASAndNewerDraftSurviveOldReceipt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		gate := make(chan struct{})
		requests := make(chan map[string]any, 2)
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://fixture.invalid"}, HTTP: &http.Client{Transport: namespaceTransport(func(r *http.Request) (*http.Response, error) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			args := body["payload"].(map[string]any)["args"].(map[string]any)
			requests <- args
			<-gate
			w := httptest.NewRecorder()
			rpcOK(w, body["rpcId"], map[string]any{"ns": "ui-theme", "revision": 8, "value": map[string]any{"fontSize": 16, "fontSizeCode": 13}, "base": map[string]any{}, "user": map[string]any{}, "secrets": []any{}})
			return w.Result(), nil
		})}}
		s := New()
		s.Attach(client, nil)
		s.mu.Lock()
		s.describedClient = client
		s.writable = true
		s.namespaces = []Namespace{{NS: "ui-theme", Revision: 7, Value: json.RawMessage(`{"fontSize":14,"fontSizeCode":13}`), Fields: []Field{{Key: "fontSize", Kind: FieldNumber}, {Key: "fontSizeCode", Kind: FieldNumber}}}}
		s.mu.Unlock()
		b := s.BindTheme()
		if err := b.Queue(map[string]any{"fontSize": 16}); err != nil {
			t.Fatal(err)
		}
		b.Flush()
		first := <-requests
		if first["expectedRevision"] != float64(7) {
			t.Fatal(first)
		}
		if err := b.Queue(map[string]any{"fontSize": 18}); err != nil {
			t.Fatal(err)
		}
		close(gate)
		synctest.Wait()
		var effective map[string]any
		_ = json.Unmarshal(b.EffectiveValue(), &effective)
		if effective["fontSize"] != float64(18) || !b.Pending() {
			t.Fatalf("old acceptance erased newer queued edit: %v", effective)
		}
		// Retire before the debounce fires; no old queue may write to a new Host.
		s.Attach(&dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://new.fixture.invalid"}}, nil)
		b.Flush()
		synctest.Wait()
		if b.Pending() || b.EffectiveValue() != nil {
			t.Fatal("old Host binding kept authority")
		}
		select {
		case request := <-requests:
			t.Fatalf("old queue crossed Host: %v", request)
		default:
		}
	})
}

func TestThemeQueueRejectsUnknownFieldAtomically(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := New()
		client := &dsh.Client{}
		s.client = client
		s.describedClient = client
		s.writable = true
		s.namespaces = []Namespace{{NS: "ui-theme", Value: json.RawMessage(`{"fontSize":14}`), Fields: []Field{{Key: "fontSize", Kind: FieldNumber}}}}
		b := s.BindTheme()
		if err := b.Queue(map[string]any{"fontSize": 16, "fontFamilySans": "local fallback"}); err == nil {
			t.Fatal("undeclared field accepted")
		}
		if b.Pending() || !sameJSON(b.EffectiveValue(), json.RawMessage(`{"fontSize":14}`)) {
			t.Fatal("rejected batch partially queued")
		}
	})
}
