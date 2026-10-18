package pet

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type quickChatTestTransport func(*http.Request) (*http.Response, error)

func (f quickChatTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func quickChatTestReply(rpcID, value string) *http.Response {
	body := `{"type":"server-response","rpcId":"` + rpcID + `","result":{"ok":true,"value":` + value + `}}`
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func TestQuickChatHTTPSharedStateAndSelection(t *testing.T) {
	client, err := newQuickChatClient("http://127.0.0.1:8787", "dsh-auth=test")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	client.plugin.HTTPClient.Transport = quickChatTestTransport(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		if r.Header.Get("Cookie") != "dsh-auth=test" {
			t.Fatal("missing Host authentication")
		}
		var body struct {
			RPCID   string          `json:"rpcId"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		switch r.URL.Path {
		case "/dsh-whale/pet/state":
			var request struct {
				IncludeCatalog bool `json:"includeCatalog"`
			}
			if err := json.Unmarshal(body.Payload, &request); err != nil || !request.IncludeCatalog {
				t.Fatal("selection must revalidate live catalog")
			}
			return quickChatTestReply(body.RPCID, `{"ok":true,"sessionId":"whale","model":{"provider":"p","model":"m","reasoningEffort":"high"},"groups":[{"id":"p","models":[{"id":"m","efforts":[{"id":"high","name":"High"}]}]}],"history":[{"role":"her","text":"shared reply","seq":2}]}`), nil
		case "/dsh-whale/pet/select-model":
			var choice quickChatModel
			if err := json.Unmarshal(body.Payload, &choice); err != nil || choice != (quickChatModel{"p", "m", "high"}) {
				t.Fatalf("wrong choice: %#v", choice)
			}
			return quickChatTestReply(body.RPCID, `{"ok":true,"selected":{"provider":"p","model":"m","reasoningEffort":"high"}}`), nil
		default:
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
		return nil, nil
	})
	state, err := client.state(context.Background(), true)
	if err != nil || state.SessionID != "whale" || len(state.History) != 1 || state.History[0].Text != "shared reply" {
		t.Fatalf("shared state: %#v, %v", state, err)
	}
	selected, err := client.selectModel(context.Background(), "whale", quickChatModel{"p", "m", "high"})
	if err != nil || selected == nil || selected.ReasoningEffort != "high" {
		t.Fatalf("selection receipt: %#v, %v", selected, err)
	}
	if len(paths) != 3 || paths[2] != "/dsh-whale/pet/select-model" {
		t.Fatalf("wrong mutation sequence: %v", paths)
	}
	if err := client.plugin.HTTPClient.CheckRedirect(nil, nil); err != http.ErrUseLastResponse {
		t.Fatal("redirect must not forward Host cookie")
	}
}

func TestQuickChatStopTargetsConfirmedPetSessionOnly(t *testing.T) {
	client, err := newQuickChatClient("http://127.0.0.1:8787", "")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client.plugin.HTTPClient.Transport = quickChatTestTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var body struct {
			RPCID   string          `json:"rpcId"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if r.URL.Path == "/dsh-whale/pet/state" {
			return quickChatTestReply(body.RPCID, `{"ok":true,"sessionId":"whale","history":[]}`), nil
		}
		if r.URL.Path != "/api/session/cancel" {
			t.Fatalf("unexpected endpoint %s", r.URL.Path)
		}
		var payload struct {
			Args struct {
				Request struct {
					SessionID string `json:"sessionId"`
				} `json:"request"`
			} `json:"args"`
		}
		if err := json.Unmarshal(body.Payload, &payload); err != nil || payload.Args.Request.SessionID != "whale" {
			t.Fatal("stop did not target the confirmed pet session")
		}
		return quickChatTestReply(body.RPCID, `{"accepted":true}`), nil
	})
	if err := client.stop(context.Background(), "old-session"); err != errQuickChatSessionChanged || calls != 1 {
		t.Fatalf("stale identity reached cancel: %d, %v", calls, err)
	}
	if err := client.stop(context.Background(), "whale"); err != nil || calls != 3 {
		t.Fatalf("stop receipt: %d, %v", calls, err)
	}
}
