package settings

import "testing"

func TestPublishedResultAccountModelCapacityOverrides(t *testing.T) {
	catalog := []any{
		map[string]any{"id": "live", "name": "Live", "contextWindow": float64(131072), "maxTokens": float64(8192)},
		map[string]any{"id": "absent", "name": "Absent"},
	}
	// Empty/null are explicit user overrides, not missing Host metadata.
	overrides := map[string]any{"live": map[string]any{"name": "", "maxTokens": nil}}
	rows := accountModelRows(catalog, accountModelOverrides(nil, overrides, nil))
	live, absent := configurationObjectValue(rows[0]), configurationObjectValue(rows[1])
	if live["contextWindow"] != float64(131072) || live["name"] != "" {
		t.Fatalf("live capacity or explicit empty name lost: %+v", live)
	}
	if value, exists := live["maxTokens"]; !exists || value != nil {
		t.Fatalf("explicit null became inherited capacity: %+v", live)
	}
	if _, exists := absent["contextWindow"]; exists {
		t.Fatal("missing capacity copied from a different model")
	}
	if original := configurationObjectValue(catalog[0]); original["maxTokens"] != float64(8192) || original["name"] != "Live" {
		t.Fatal("editing override mutated Host catalog baseline")
	}
}
