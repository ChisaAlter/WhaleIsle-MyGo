package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type triggerTransport func(*http.Request) (*http.Response, error)

func (f triggerTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestBotRoutineTriggerExplicitWireOnly(t *testing.T) {
	calls := 0
	client := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: triggerTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if r.URL.Path != "/dshbot/bot/trigger-token" || req.Method != "bot/trigger-token" || len(req.Payload.(map[string]any)) != 0 {
			t.Fatal("invented or trigger-execution endpoint", req, r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":{"token":"synthetic-fixture-token-not-a-real-credential"}}}`))}, nil
	})}}
	token, err := client.DshbotObtainRoutineTriggerToken(context.Background())
	if err != nil || calls != 1 || string(token) != "synthetic-fixture-token-not-a-real-credential" {
		t.Fatal("single explicit get-or-create failed", err, calls)
	}
	clear(token)
	for _, b := range token {
		if b != 0 {
			t.Fatal("owned fixture buffer not cleared")
		}
	}
}
func TestBotRoutineTriggerErrorsDoNotLeakOrRetry(t *testing.T) {
	for _, body := range []string{`synthetic-secret-in-http-error`, `{"result":{"ok":false,"error":{"code":"rejected","message":"synthetic-secret-in-error"}}}`, `{"result":{"ok":true,"value":{"token":"synthetic\nsecret"}}}`} {
		calls := 0
		status := 200
		if body == "synthetic-secret-in-http-error" {
			status = 500
		}
		client := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: triggerTransport(func(*http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}}
		token, err := client.DshbotObtainRoutineTriggerToken(context.Background())
		if err == nil || len(token) != 0 || strings.Contains(err.Error(), "synthetic") || calls != 1 {
			t.Fatal("credential leaked or retried", err, calls)
		}
	}
}
func TestBotRoutineTriggerAddressAuthority(t *testing.T) {
	got, err := DshbotRoutineTriggerURL("http://127.0.0.1:8787", "r-1")
	if err != nil || got != "http://127.0.0.1:8787/dshbot-hook/routine/r-1" {
		t.Fatal(got, err)
	}
	for _, tc := range []struct{ origin, id string }{{"file:///tmp", "r"}, {"https://user:secret@fixture.invalid", "r"}, {"http://127.0.0.1/path", "r"}, {"http://127.0.0.1?secret=x", "r"}, {"http://127.0.0.1", "../x"}, {"http://127.0.0.1", "r#x"}} {
		if _, err := DshbotRoutineTriggerURL(tc.origin, tc.id); err == nil {
			t.Fatal("unsafe address accepted")
		}
	}
}
