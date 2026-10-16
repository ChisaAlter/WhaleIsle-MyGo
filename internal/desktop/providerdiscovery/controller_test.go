package providerdiscovery

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

type fakeHost struct {
	revision       float64
	profile        map[string]any
	writes         []map[string]any
	fail, mismatch bool
	onCall         func(string)
	discoveries    []map[string]any
	discover       func(context.Context, map[string]any) (any, error)
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
	case "llm/discoverModels":
		encoded, err := json.Marshal(args)
		if err != nil {
			return err
		}
		var a map[string]any
		if err := json.Unmarshal(encoded, &a); err != nil {
			return err
		}
		h.discoveries = append(h.discoveries, a)
		if h.discover != nil {
			result, err = h.discover(ctx, a)
			if err != nil {
				return err
			}
		} else {
			result = []any{
				map[string]any{"id": "existing", "name": "Server rewrite", "contextWindow": 1},
				map[string]any{"id": "new", "name": "New", "contextWindow": 200000, "maxTokens": 8192, "inputModalities": []string{"text", "image"}},
				map[string]any{"id": "new"},
			}
		}
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
	c.Discover(context.Background(), c.Snapshot().Ticket)
	s := c.Snapshot()
	if s.Err != "" || len(s.Candidates) != 2 {
		t.Fatalf("discover: %#v", s)
	}
	c.Adopt(s.Ticket)
}

func TestSP23HostAdapterDiscoveryAndRevisionSave(t *testing.T) {
	c, h := loaded(t)
	edit(c, "endpoint", "https://draft.example/openai/v1")
	edit(c, "key", "temporary-fixture-key")
	h.discover = func(ctx context.Context, args map[string]any) (any, error) {
		request := args["request"].(map[string]any)
		if args["settingsNs"] != "llm-pi-ai" || request["provider"] != "demo" || request["baseURL"] != "https://draft.example/openai/v1" || request["api"] != "openai-completions" || request["apiKey"] != "temporary-fixture-key" || len(request) != 4 {
			t.Fatalf("wrong Host adapter request: %#v", args)
		}
		return []any{
			map[string]any{"id": "existing", "name": "Server rewrite", "contextWindow": 1},
			map[string]any{"id": "new", "name": "New", "contextWindow": 200000, "maxTokens": 8192, "inputModalities": []string{"text", "image"}},
			map[string]any{"id": "new"},
		}, nil
	}
	if len(h.discoveries) != 0 || len(h.writes) != 0 {
		t.Fatal("edited fields triggered an action")
	}
	oldTicket := c.Snapshot().Ticket
	c.Discover(context.Background(), oldTicket)
	c.Discover(context.Background(), oldTicket)
	s := c.Snapshot()
	if len(h.discoveries) != 1 || s.Key != "" || len(s.Candidates) != 2 || s.Candidates[0].Picked || !s.Candidates[1].Picked {
		t.Fatalf("bad candidates/lifecycle: %#v", s)
	}
	s.Candidates[1].InputModalities[0] = "changed copy"
	c.Adopt(s.Ticket)
	s = c.Snapshot()
	if got := s.Models[1]["input"]; !reflect.DeepEqual(got, []any{"text", "image"}) {
		t.Fatalf("catalog capabilities lost or aliased: %#v", got)
	}
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
	c, h := loaded(t)
	for _, key := range []string{"********", "line\nbreak", "非法"} {
		edit(c, "key", key)
		c.Discover(ctx, c.Snapshot().Ticket)
		if c.Snapshot().Err == "" || len(h.discoveries) != 0 || c.Snapshot().Key != "" {
			t.Fatal("invalid temporary key was sent or retained")
		}
	}
	h.discover = func(context.Context, map[string]any) (any, error) {
		return nil, errors.New("raw-secret-error")
	}
	c.Discover(ctx, c.Snapshot().Ticket)
	if s := c.Snapshot(); s.Err == "" || strings.Contains(s.Err, "raw-secret-error") || s.Busy || len(s.Candidates) != 0 {
		t.Fatal("Host discovery failure leaked or published candidates")
	}
	c, h = loaded(t)
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
	h.discover = func(context.Context, map[string]any) (any, error) {
		c.Attach(fixtureHost())
		return []any{map[string]any{"id": "stale"}}, nil
	}
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
	c, h := loaded(t)
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
	h.discover = func(ctx context.Context, _ map[string]any) (any, error) {
		c.Suspend()
		if ctx.Err() == nil {
			t.Fatal("hide did not cancel Host discovery context")
		}
		return []any{map[string]any{"id": "late-model"}}, nil
	}
	c.Discover(context.Background(), c.Snapshot().Ticket)
	if s := c.Snapshot(); s.Busy || len(s.Candidates) != 0 || !s.Dirty {
		t.Fatal("hidden probe published or lost non-secret draft")
	}
	c, h = loaded(t)
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

func TestSP23NativeCatalogAndSavedHostProfile(t *testing.T) {
	for _, protocol := range []string{"", "google-generative-ai"} {
		t.Run("native-"+protocol, func(t *testing.T) {
			h := fixtureHost()
			delete(h.profile, "baseURL")
			delete(h.profile, "api")
			if protocol != "" {
				h.profile["api"] = protocol
			}
			c := &Controller{}
			c.Attach(h)
			c.Refresh(context.Background())
			s := c.Snapshot()
			c.Select(s.Ticket, s.Rows[0].ID)
			h.discover = func(_ context.Context, args map[string]any) (any, error) {
				r := args["request"].(map[string]any)
				if args["settingsNs"] != "llm-pi-ai" || r["provider"] != "demo" || r["baseURL"] != nil || r["apiKey"] != nil {
					t.Fatalf("native lookup invented endpoint/key: %#v", args)
				}
				if (protocol == "" && r["api"] != nil) || (protocol != "" && r["api"] != protocol) {
					t.Fatalf("native protocol replaced: %#v", r)
				}
				return []any{map[string]any{"id": "native", "inputModalities": []string{"text", "image"}}}, nil
			}
			c.Discover(context.Background(), c.Snapshot().Ticket)
			s = c.Snapshot()
			if s.Err != "" || len(s.Candidates) != 1 || s.Candidates[0].Name != "native" {
				t.Fatalf("native discovery: %#v", s)
			}
			c.Adopt(s.Ticket)
			c.Review(c.Snapshot().Ticket, true)
			if !c.Save(context.Background(), c.Snapshot().Ticket) {
				t.Fatalf("native save: %s", c.Snapshot().Err)
			}
			ops := h.writes[0]["ops"].([]map[string]any)
			if len(ops) != 1 || !reflect.DeepEqual(ops[0]["path"], []string{"providers", "demo", "models"}) {
				t.Fatalf("catalog adoption overwrote native defaults: %#v", ops)
			}
		})
	}
	t.Run("saved-credential-and-headers", func(t *testing.T) {
		c, h := loaded(t)
		h.profile["headers"] = map[string]any{"X-Deployment": "fixture-deployment"}
		storedKeys := map[string]string{"DEMO_KEY": "host-only-fixture-key"}
		h.discover = func(_ context.Context, args map[string]any) (any, error) {
			r := args["request"].(map[string]any)
			if r["provider"] != "demo" || r["apiKey"] != nil || r["headers"] != nil {
				t.Fatalf("wrong profile discovery: %#v", r)
			}
			// Only the Host fixture resolves private credentials and deployment headers.
			if storedKeys[h.profile["apiKeyEnv"].(string)] != "host-only-fixture-key" || h.profile["headers"].(map[string]any)["X-Deployment"] != "fixture-deployment" {
				t.Fatal("saved Host profile unavailable")
			}
			return []any{map[string]any{"id": "authenticated"}}, nil
		}
		c.Discover(context.Background(), c.Snapshot().Ticket)
		s := c.Snapshot()
		if s.Err != "" || len(s.Candidates) != 1 || s.Candidates[0].ID != "authenticated" {
			t.Fatalf("saved profile not used: %#v", s)
		}
		encoded, _ := json.Marshal(s)
		if strings.Contains(string(encoded), "host-only-fixture-key") || strings.Contains(string(encoded), "fixture-deployment") {
			t.Fatal("Host secrets copied into desktop state")
		}
	})
}
