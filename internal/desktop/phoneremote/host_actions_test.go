package phoneremote

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestHostSaveConfigPropagatesValidationFailure(t *testing.T) {
	child := &fixtureChild{}
	r := NewRuntime()
	r.child = child
	calls := 0
	r.SetHostActions(HostActions{SaveConfig: func(patch map[string]any) (map[string]any, error) {
		calls++
		if patch["theme"] != float64(42) {
			t.Fatalf("bridge changed the invalid field: %v", patch)
		}
		return nil, errors.New("theme must be a valid theme id")
	}})
	r.consume(child, []byte(`{"kind":"host-save-config","id":"1","patch":{"theme":42}}`))
	var result map[string]any
	if err := json.Unmarshal(child.input.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result["ok"] != false || result["error"] != "theme must be a valid theme id" || result["config"] != nil || result["id"] != "1" {
		t.Fatalf("validation error became a success: %v, calls=%d", result, calls)
	}
}

func TestHostActionsRejectRevokedChildAndInvalidIdentity(t *testing.T) {
	active, stale := &fixtureChild{}, &fixtureChild{}
	r := NewRuntime()
	r.child = active
	calls := 0
	r.SetHostActions(HostActions{
		SaveConfig:   func(map[string]any) (map[string]any, error) { calls++; return map[string]any{"theme": "deepseek"}, nil },
		OpenSettings: func(string) bool { calls++; return true },
	})
	for _, kind := range []string{"host-save-config", "host-open-settings"} {
		r.consume(stale, []byte(`{"kind":"`+kind+`","id":"1","patch":{"theme":"deepseek"},"section":"appearance"}`))
		for _, id := range []string{"", "bad", "1\n2", "123456789012345678901"} {
			r.handleHostAction(active, kind, id, "appearance", nil)
		}
	}
	if calls != 0 || stale.input.Len() != 0 || active.input.Len() != 0 {
		t.Fatalf("revoked/malformed action reached desktop: calls=%d", calls)
	}
	r.consume(active, []byte(`{"kind":"host-save-config","id":"2","patch":{"theme":"deepseek"}}`))
	var result map[string]any
	if err := json.Unmarshal(active.input.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || result["ok"] != true || result["error"] != nil {
		t.Fatalf("valid active action failed: %v", result)
	}
}
