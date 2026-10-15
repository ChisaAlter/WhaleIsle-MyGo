package providerdiscovery

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type fakeTransport func(*http.Request) (*http.Response, error)

func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func answer(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}
}

type fakeHost struct {
	revision       float64
	profile        map[string]any
	writes         []map[string]any
	fail, mismatch bool
	onCall         func(string)
}

func fixtureHost() *fakeHost {
	return &fakeHost{revision: 7, profile: map[string]any{"apiKeyEnv": "DEMO_KEY", "api": "openai-completions", "baseURL": "https://old.example/v1", "models": []any{map[string]any{"id": "existing", "name": "Tuned", "contextWindow": float64(999), "compat": map[string]any{"keep": true}}}}}
}
func (h *fakeHost) Call(ctx context.Context, method string, args, out any) error {
	if h.onCall != nil {
		h.onCall(method)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var result any
	profileSchema := map[string]any{"type": "object", "dict": map[string]any{"models": map[string]any{"type": "array"}, "baseURL": map[string]any{"type": "string"}, "api": map[string]any{"type": "string"}}}
	ns := func() map[string]any {
		return map[string]any{"ns": "llm-pi-ai", "revision": h.revision, "schema": map[string]any{"type": "object", "dict": map[string]any{"providers": map[string]any{"type": "dict", "inner": profileSchema}}}, "value": map[string]any{"providers": map[string]any{"demo": h.profile}}}
	}
	switch method {
	case "settings/describe":
		result = map[string]any{"writable": true, "namespaces": []any{ns()}}
	case "llm/listConfigurableProviders":
		result = []any{map[string]any{"provider": "demo", "displayName": "Demo", "settingsNs": "llm-pi-ai", "settingsPath": []string{"providers", "demo"}}}
	case "credentials/describe":
		result = map[string]any{"DEMO_KEY": map[string]any{"configured": true}}
	case "settings/mutate":
		if h.fail {
			return errors.New("raw-secret-error")
		}
		a := args.(map[string]any)
		if a["expectedRevision"] != h.revision {
			return errors.New("conflict")
		}
		h.writes = append(h.writes, a)
		for _, op := range a["ops"].([]map[string]any) {
			path := op["path"].([]string)
			if len(path) != 3 || path[0] != "providers" || path[1] != "demo" {
				return errors.New("wrong target")
			}
			h.profile[path[2]] = op["value"]
		}
		h.revision++
		result = ns()
		if h.mismatch {
			result.(map[string]any)["ns"] = "wrong"
		}
	default:
		return errors.New("unexpected call: " + method)
	}
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}
func loaded(t *testing.T) (*Controller, *fakeHost) {
	t.Helper()
	h := fixtureHost()
	c := &Controller{}
	c.Attach(h)
	c.Refresh(context.Background())
	s := c.Snapshot()
	if !s.Ready || len(s.Rows) != 1 || s.Rows[0].Blocked != "" {
		t.Fatalf("load failed: %#v", s)
	}
	c.Select(s.Ticket, s.Rows[0].ID)
	return c, h
}
func edit(c *Controller, field, value string) { c.Edit(c.Snapshot().Ticket, field, value) }
func stage(t *testing.T, c *Controller) {
	t.Helper()
	c.transport = fakeTransport(func(r *http.Request) (*http.Response, error) {
		return answer(200, `{"data":[{"id":"existing","name":"Server rewrite","context_window":1},{"id":"new","name":"New","context_window":200000,"max_output_tokens":8192},{"id":"new"}]}`), nil
	})
	c.Discover(context.Background(), c.Snapshot().Ticket)
	s := c.Snapshot()
	if s.Err != "" || len(s.Candidates) != 2 {
		t.Fatalf("discover: %#v", s)
	}
	c.Adopt(s.Ticket)
}

func TestSP23ExplicitProbeAdoptAndRevisionSave(t *testing.T) {
	c, h := loaded(t)
	edit(c, "endpoint", "https://draft.example/openai/v1")
	edit(c, "key", "temporary-fixture-key")
	calls := 0
	c.transport = fakeTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://draft.example/openai/v1/models" || r.Header.Get("Authorization") != "Bearer temporary-fixture-key" || r.Header.Get("Cookie") != "" {
			t.Fatal("wrong explicit probe target or headers")
		}
		return answer(200, `{"data":[{"id":"existing","name":"Server rewrite","context_window":1},{"id":"new","name":"New","context_window":200000,"max_output_tokens":8192}]}`), nil
	})
	if calls != 0 || len(h.writes) != 0 {
		t.Fatal("edited fields triggered an action")
	}
	oldTicket := c.Snapshot().Ticket
	c.Discover(context.Background(), oldTicket)
	c.Discover(context.Background(), oldTicket)
	s := c.Snapshot()
	if calls != 1 || s.Key != "" || len(s.Candidates) != 2 || s.Candidates[0].Picked || !s.Candidates[1].Picked {
		t.Fatalf("bad candidates/lifecycle: %#v", s)
	}
	c.Adopt(s.Ticket)
	s = c.Snapshot()
	if len(h.writes) != 0 || !s.Dirty || len(s.Models) != 2 || s.Models[0]["name"] != "Tuned" || s.Models[0]["compat"] == nil {
		t.Fatal("adoption wrote early or lost existing fields")
	}
	if c.Save(context.Background(), s.Ticket) {
		t.Fatal("save bypassed review")
	}
	c.Review(s.Ticket, true)
	if !c.Save(context.Background(), s.Ticket) {
		t.Fatalf("save: %s", c.Snapshot().Err)
	}
	if len(h.writes) != 1 || h.writes[0]["expectedRevision"] != float64(7) || c.Snapshot().Dirty || c.Snapshot().Err != "" {
		t.Fatal("save verification failed")
	}
	b, _ := json.Marshal(h.writes)
	if strings.Contains(string(b), "temporary-fixture-key") || strings.Contains(string(b), "apiKeyEnv") {
		t.Fatal("probe key or credential field persisted")
	}
}

func TestSP23SafetyAndFailureBoundaries(t *testing.T) {
	ctx := context.Background()
	for _, target := range []string{"http://provider.example/v1", "https://name:password@example.test", "https://example.test?api_key=secret", "file:///models"} {
		if _, err := ListingURL(target, "openai-completions"); err == nil {
			t.Fatalf("accepted %s", target)
		}
	}
	if _, err := ListingURL("https://example.test", "azure-openai-responses"); err == nil {
		t.Fatal("invented unsupported protocol")
	}
	calls := 0
	transport := fakeTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		response := answer(302, "")
		response.Header.Set("Location", "https://other.example/models")
		return response, nil
	})
	if _, err := probe(ctx, transport, "https://example.test/v1", "openai-completions", "fixture-key"); err == nil || calls != 1 {
		t.Fatal("redirect not stopped")
	}
	if _, err := probe(ctx, transport, "http://127.0.0.1:11434/v1", "openai-completions", "fixture-key"); err == nil || calls != 1 {
		t.Fatal("sent key over HTTP")
	}
	if _, err := probe(ctx, transport, "https://example.test/v1", "openai-completions", "********"); err == nil || calls != 1 {
		t.Fatal("sent masked key")
	}
	anthropic := fakeTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://example.test/prefix/v1/models?limit=1000" || r.Header.Get("x-api-key") != "fixture" || r.Header.Get("anthropic-version") != "2023-06-01" || r.Header.Get("Authorization") != "" {
			t.Fatal("wrong Anthropic contract")
		}
		return answer(200, `{"models":{"alias":{"id":"canonical","display_name":"Named","limit":{"context":1000,"output":100}}}}`), nil
	})
	models, err := probe(ctx, anthropic, "https://example.test/prefix/v1/", "anthropic-messages", "fixture")
	if err != nil || len(models) != 1 || models[0].ID != "alias" || models[0].MaxTokens != 100 {
		t.Fatal("enriched listing failed")
	}
	large := fakeTransport(func(*http.Request) (*http.Response, error) {
		r := answer(200, "")
		r.ContentLength = maxResponseBytes + 1
		return r, nil
	})
	if _, err := probe(ctx, large, "https://example.test", "openai-responses", ""); err == nil {
		t.Fatal("unbounded response")
	}
	c, h := loaded(t)
	stage(t, c)
	h.revision++
	c.Review(c.Snapshot().Ticket, true)
	if c.Save(ctx, c.Snapshot().Ticket) || len(h.writes) != 0 || !c.Snapshot().Dirty {
		t.Fatal("revision conflict lost draft or wrote")
	}
	c, h = loaded(t)
	stage(t, c)
	h.fail = true
	c.Review(c.Snapshot().Ticket, true)
	if c.Save(ctx, c.Snapshot().Ticket) || strings.Contains(c.Snapshot().Err, "raw-secret-error") || !c.Snapshot().Dirty {
		t.Fatal("write failure not safe/retryable")
	}
	h.fail = false
	c.Review(c.Snapshot().Ticket, true)
	if !c.Save(ctx, c.Snapshot().Ticket) {
		t.Fatal("explicit retry failed")
	}
	c, h = loaded(t)
	stage(t, c)
	h.mismatch = true
	c.Review(c.Snapshot().Ticket, true)
	if c.Save(ctx, c.Snapshot().Ticket) || c.Snapshot().Ready {
		t.Fatal("unverified acknowledgement reported success")
	}
}

func TestSP23HostReplacementAndDraftIsolation(t *testing.T) {
	c, h := loaded(t)
	edit(c, "key", "fixture-key")
	old := c.Snapshot().Ticket
	c.transport = fakeTransport(func(*http.Request) (*http.Response, error) {
		c.Attach(fixtureHost())
		return answer(200, `{"data":[{"id":"stale"}]}`), nil
	})
	c.Discover(context.Background(), old)
	if s := c.Snapshot(); s.Ready || s.Key != "" || len(s.Candidates) > 0 || s.Busy {
		t.Fatalf("stale completion: %#v", s)
	}
	c.Edit(old, "key", "late-secret")
	if c.Snapshot().Key != "" {
		t.Fatal("stale frame wrote key")
	}
	c, h = loaded(t)
	stage(t, c)
	c.Review(c.Snapshot().Ticket, true)
	h.onCall = func(method string) {
		if method == "settings/describe" {
			c.Attach(fixtureHost())
		}
	}
	if c.Save(context.Background(), c.Snapshot().Ticket) || len(h.writes) > 0 {
		t.Fatal("Host changed during preflight but wrote settings")
	}
	c, _ = loaded(t)
	original := c.Snapshot().Models
	edit(c, "endpoint", "https://draft.example")
	c.Discard(c.Snapshot().Ticket)
	if c.Snapshot().Dirty || !reflect.DeepEqual(c.Snapshot().Models, original) {
		t.Fatal("discard changed saved models")
	}
}

func TestSP23HiddenPageRetiresSecretsAndPreservesDraft(t *testing.T) {
	c, _ := loaded(t)
	c.ObserveVisible(true)
	edit(c, "endpoint", "https://draft.example/v1")
	edit(c, "key", "hidden-fixture-key")
	old := c.Snapshot().Ticket
	c.Review(old, true)
	c.Suspend()
	s := c.Snapshot()
	if s.Key != "" || s.Review || s.Endpoint != "https://draft.example/v1" || !s.Dirty || s.Ticket == old {
		t.Fatal("hide did not retire only sensitive state")
	}
	c.Edit(old, "key", "stale-secret")
	if c.Snapshot().Key != "" || c.Save(context.Background(), old) {
		t.Fatal("hidden callback remained authorized")
	}
	c.ObserveVisible(true)
	c.transport = fakeTransport(func(r *http.Request) (*http.Response, error) {
		c.Suspend()
		if r.Context().Err() == nil {
			t.Fatal("hide did not cancel probe context")
		}
		return answer(200, `{"data":[{"id":"late-model"}]}`), nil
	})
	c.Discover(context.Background(), c.Snapshot().Ticket)
	if s := c.Snapshot(); s.Busy || len(s.Candidates) != 0 || !s.Dirty {
		t.Fatal("hidden probe published or lost non-secret draft")
	}
	c, h := loaded(t)
	c.ObserveVisible(true)
	stage(t, c)
	c.Review(c.Snapshot().Ticket, true)
	h.onCall = func(method string) {
		if method == "settings/mutate" {
			c.Suspend()
		}
	}
	if !c.Save(context.Background(), c.Snapshot().Ticket) || c.Snapshot().Busy || c.Snapshot().Dirty || len(h.writes) != 1 {
		t.Fatal("hide disrupted verification of already confirmed save")
	}
}
