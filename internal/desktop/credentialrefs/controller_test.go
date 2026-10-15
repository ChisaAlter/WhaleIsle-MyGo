package credentialrefs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type fakeHost struct {
	settings   map[string]any
	providers  []map[string]any
	configured map[string]bool
	writable   map[string]bool
	calls      []string
	writes     []map[string]any
	failMethod string
	hook       func(string)
}

func fixtureHost() *fakeHost {
	profile := map[string]any{"type": "object", "dict": map[string]any{"apiKeyEnv": map[string]any{"type": "string"}}}
	return &fakeHost{
		settings: map[string]any{"writable": true, "namespaces": []any{
			map[string]any{"ns": "llm-pi-ai", "revision": float64(7), "schema": map[string]any{"uid": 1, "refs": map[string]any{"1": map[string]any{"type": "object", "dict": map[string]any{"providers": 2}}, "2": map[string]any{"type": "dict", "inner": 3}, "3": profile}}, "value": map[string]any{"providers": map[string]any{"anthropic": map[string]any{"apiKeyEnv": "ANTHROPIC_CUSTOM"}, "minimax-cn": map[string]any{}}}},
			map[string]any{"ns": "web-search-deepseek", "revision": float64(4), "schema": profile, "value": map[string]any{"apiKeyEnv": "SEARCH_CUSTOM"}},
		}},
		providers: []map[string]any{
			{"provider": "anthropic", "displayName": "Anthropic", "settingsNs": "llm-pi-ai", "settingsPath": []string{"providers", "anthropic"}},
			{"provider": "minimax-cn", "displayName": "MiniMax", "settingsNs": "llm-pi-ai", "settingsPath": []string{"providers", "minimax-cn"}},
		}, configured: map[string]bool{"ANTHROPIC_CUSTOM": true, "SEARCH_CUSTOM": true}, writable: map[string]bool{},
	}
}
func (h *fakeHost) Call(ctx context.Context, method string, args, out any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	h.calls = append(h.calls, method)
	if h.hook != nil {
		h.hook(method)
	}
	if h.failMethod == method {
		return errors.New("rejected fake-secret-do-not-expose")
	}
	var result any
	a, _ := args.(map[string]any)
	switch method {
	case "settings/describe":
		result = h.settings
	case "llm/listConfigurableProviders":
		result = h.providers
	case "credentials/describe":
		result = map[string]any{}
		refs := a["refs"].([]string)
		if len(refs) > 64 {
			return errors.New("too many refs")
		}
		for _, ref := range refs {
			w, ok := h.writable[ref]
			if !ok {
				w = true
			}
			result.(map[string]any)[ref] = map[string]any{"configured": h.configured[ref], "writable": w}
		}
	case "settings/mutate":
		h.writes = append(h.writes, a)
		for _, v := range h.settings["namespaces"].([]any) {
			ns := v.(map[string]any)
			if ns["ns"] == a["ns"] {
				if ns["revision"] != a["expectedRevision"] {
					return errors.New("conflict")
				}
				for _, op := range a["ops"].([]map[string]any) {
					path := op["path"].([]string)
					node := ns["value"].(map[string]any)
					for _, key := range path[:len(path)-1] {
						next, _ := node[key].(map[string]any)
						if next == nil {
							next = map[string]any{}
							node[key] = next
						}
						node = next
					}
					node[path[len(path)-1]] = op["value"]
				}
				ns["revision"] = ns["revision"].(float64) + 1
				result = ns
			}
		}
	case "credentials/set":
		h.writes = append(h.writes, a)
		h.configured[a["ref"].(string)] = true
	case "credentials/unset":
		h.writes = append(h.writes, a)
		h.configured[a["ref"].(string)] = false
	default:
		return errors.New("unexpected method")
	}
	if out != nil {
		b, _ := json.Marshal(result)
		return json.Unmarshal(b, out)
	}
	return nil
}
func selectRef(t *testing.T, c *Controller, ref string) Snapshot {
	t.Helper()
	for _, r := range c.Snapshot().Rows {
		if r.Ref == ref {
			c.Select(r.ID)
			return c.Snapshot()
		}
	}
	t.Fatalf("reference not selectable: %s", ref)
	return Snapshot{}
}
func TestProviderAndSearchWriteOnlySelectedReference(t *testing.T) {
	for _, ref := range []string{"ANTHROPIC_CUSTOM", "SEARCH_CUSTOM"} {
		t.Run(ref, func(t *testing.T) {
			h := fixtureHost()
			var c Controller
			c.Attach(h)
			c.Refresh(context.Background())
			s := selectRef(t, &c, ref)
			c.Edit(s.Ticket, "  fake-secret-do-not-expose  ")
			c.Save(context.Background(), s.Ticket)
			if len(h.writes) != 1 || h.writes[0]["ref"] != ref || h.writes[0]["value"] != "fake-secret-do-not-expose" {
				t.Fatalf("write targeted wrong reference or value: %d", len(h.writes))
			}
			if c.Snapshot().Err != "" || c.Snapshot().Draft != "" {
				t.Fatal("save did not clear secret draft or reported failure")
			}
		})
	}
}
func TestBlankDraftPreservesExistingSecret(t *testing.T) {
	h := fixtureHost()
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	s := selectRef(t, &c, "SEARCH_CUSTOM")
	c.Edit(s.Ticket, "  ")
	before := len(h.calls)
	c.Save(context.Background(), s.Ticket)
	if len(h.writes) != 0 || len(h.calls) != before {
		t.Fatal("blank draft must not issue a credential request")
	}
}
func TestDerivedProviderReferenceUsesMinimalRevisionedMutation(t *testing.T) {
	h := fixtureHost()
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	s := selectRef(t, &c, "MINIMAX_CN_API_KEY")
	c.Edit(s.Ticket, "fake-key")
	c.Save(context.Background(), s.Ticket)
	if len(h.writes) != 2 {
		t.Fatal("missing profile bind or key write")
	}
	op := h.writes[0]
	if op["ns"] != "llm-pi-ai" || op["expectedRevision"] != float64(7) || h.writes[1]["ref"] != "MINIMAX_CN_API_KEY" {
		t.Fatal("wrong namespace, revision or credential ref")
	}
	ops := op["ops"].([]map[string]any)
	p := ops[0]["path"].([]string)
	if len(ops) != 1 || strings.Join(p, "/") != "providers/minimax-cn/apiKeyEnv" {
		t.Fatal("must mutate only selected reference field")
	}
}
func TestClearIsExplicitAndSelected(t *testing.T) {
	h := fixtureHost()
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	s := selectRef(t, &c, "SEARCH_CUSTOM")
	c.StageClear(s.Ticket, true)
	c.Save(context.Background(), s.Ticket)
	if len(h.writes) != 1 || h.writes[0]["ref"] != "SEARCH_CUSTOM" || h.configured["SEARCH_CUSTOM"] || !h.configured["ANTHROPIC_CUSTOM"] {
		t.Fatal("clear affected wrong credential")
	}
}
func TestLockedAndChangedRevisionNeverWrite(t *testing.T) {
	for _, mode := range []string{"locked", "revision", "reference"} {
		t.Run(mode, func(t *testing.T) {
			h := fixtureHost()
			var c Controller
			c.Attach(h)
			c.Refresh(context.Background())
			s := selectRef(t, &c, "SEARCH_CUSTOM")
			c.Edit(s.Ticket, "fake-key")
			switch mode {
			case "locked":
				h.writable["SEARCH_CUSTOM"] = false
			case "revision":
				h.settings["namespaces"].([]any)[1].(map[string]any)["revision"] = float64(5)
			case "reference":
				h.settings["namespaces"].([]any)[1].(map[string]any)["value"] = map[string]any{"apiKeyEnv": "CHANGED"}
			}
			c.Save(context.Background(), s.Ticket)
			if len(h.writes) != 0 || c.Snapshot().Err == "" {
				t.Fatal("stale or read-only target was not refused")
			}
		})
	}
}
func TestHostSwitchDiscardsOldDraftAndStopsFollowOnWrites(t *testing.T) {
	h := fixtureHost()
	next := fixtureHost()
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	s := selectRef(t, &c, "MINIMAX_CN_API_KEY")
	c.Edit(s.Ticket, "fake-key")
	h.hook = func(method string) {
		if method == "settings/mutate" {
			c.Attach(next)
		}
	}
	c.Save(context.Background(), s.Ticket)
	if len(h.writes) != 1 || len(next.writes) != 0 || c.Snapshot().Draft != "" || len(c.Snapshot().Rows) != 0 {
		t.Fatal("stale completion or draft crossed Host boundary")
	}
	c.Save(context.Background(), s.Ticket)
	if len(next.writes) != 0 {
		t.Fatal("stale save reached next Host")
	}
}
func TestFailureNeverEchoesSecretAndRetryKeepsAdvancedRevision(t *testing.T) {
	h := fixtureHost()
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	s := selectRef(t, &c, "MINIMAX_CN_API_KEY")
	c.Edit(s.Ticket, "fake-secret-do-not-expose")
	h.failMethod = "credentials/set"
	c.Save(context.Background(), s.Ticket)
	if c.Snapshot().Err == "" || strings.Contains(c.Snapshot().Err, "fake-secret") {
		t.Fatal("missing or unsafe failure")
	}
	h.failMethod = ""
	c.Save(context.Background(), s.Ticket)
	if c.Snapshot().Err != "" || len(h.writes) != 2 {
		t.Fatal("retry repeated committed profile mutation or remained stale")
	}
}
func TestSchemaAndRefGrammarGateTargets(t *testing.T) {
	h := fixtureHost()
	h.settings["namespaces"].([]any)[1].(map[string]any)["value"] = map[string]any{"apiKeyEnv": "BAD-REF"}
	h.settings["namespaces"].([]any)[0].(map[string]any)["schema"] = map[string]any{"type": "object", "dict": map[string]any{}}
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	if len(c.Snapshot().Rows) != 0 {
		t.Fatal("unsupported schema or invalid ref was editable")
	}
}

func TestSearchDefaultAndSettingsReadOnly(t *testing.T) {
	h := fixtureHost()
	ns := h.settings["namespaces"].([]any)[1].(map[string]any)
	ns["value"] = map[string]any{}
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	s := selectRef(t, &c, "DEEPSEEK_API_KEY")
	c.Edit(s.Ticket, "fake-key")
	h.settings["writable"] = false
	c.Save(context.Background(), s.Ticket)
	if len(h.writes) != 0 || c.Snapshot().Err == "" {
		t.Fatal("read-only settings must fence credential edits")
	}
}
func TestDescribeBatchesFollowHostLimit(t *testing.T) {
	h := fixtureHost()
	h.providers = nil
	values := map[string]any{}
	for i := 0; i < 70; i++ {
		name := fmt.Sprintf("route%d", i)
		h.providers = append(h.providers, map[string]any{"provider": name, "settingsNs": "llm-pi-ai", "settingsPath": []string{"providers", name}})
		values[name] = map[string]any{}
	}
	h.settings["namespaces"].([]any)[0].(map[string]any)["value"] = map[string]any{"providers": values}
	var c Controller
	c.Attach(h)
	c.Refresh(context.Background())
	if c.Snapshot().Err != "" || len(c.Snapshot().Rows) != 71 {
		t.Fatal("credential batches failed")
	}
	count := 0
	for _, m := range h.calls {
		if m == "credentials/describe" {
			count++
		}
	}
	if count != 2 {
		t.Fatal("expected bounded describe batches")
	}
}
