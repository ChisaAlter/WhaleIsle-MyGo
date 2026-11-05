package settings

import (
	"encoding/json"
	"testing"

	store "whaleisle/internal/store/settings"
)

func TestAccountModelsPreserveLiveCatalogAndWriteOnlyPerIDChanges(t *testing.T) {
	catalog := []any{map[string]any{"id": "live", "name": "Live", "inputModalities": []any{"text", "image"}}, map[string]any{"id": "legacy-hidden"}, map[string]any{"id": "new-live"}}
	legacy := []any{map[string]any{"id": "live", "contextWindow": float64(1000)}}
	defaults := []any{map[string]any{"id": "live"}, map[string]any{"id": "legacy-hidden"}}
	overrides := accountModelOverrides(legacy, map[string]any{"live": map[string]any{"contextWindow": float64(2000)}}, defaults)
	rows := accountModelRows(catalog, overrides)
	if len(rows) != 2 || configurationObjectValue(rows[0])["contextWindow"] != float64(2000) || configurationObjectValue(rows[1])["id"] != "new-live" {
		t.Fatalf("migration replaced live catalog or ignored explicit overrides: %#v", rows)
	}
	var edited []any
	raw, _ := json.Marshal(rows)
	_ = json.Unmarshal(raw, &edited)
	configurationObjectValue(edited[0])["name"] = "Edited"
	next := editedAccountOverrides(rows, edited, overrides)
	live := configurationObjectValue(next["live"])
	if live["name"] != "Edited" || live["contextWindow"] != float64(2000) || live["inputModalities"] != nil || next["new-live"] != nil || configurationObjectValue(overrides["live"])["name"] != nil {
		t.Fatalf("edit copied untouched catalog metadata or mutated baseline: %#v", next)
	}
	next = editedAccountOverrides(rows, edited[1:], overrides)
	if configurationObjectValue(next["live"])["disabled"] != true {
		t.Fatal("removed catalog ID was not disabled")
	}
	if !configurationInvalid(store.ViewState{SelectedNS: "llm-deepseek-account", Drafts: map[string]string{"@account-model-rows": `[{"id":""}]`}}) {
		t.Fatal("unfinished new model could be saved")
	}
}

func TestCredentialReferencesRemainOrdinaryFields(t *testing.T) {
	for _, field := range []store.Field{{Key: "apiKeyEnv", Kind: store.FieldString}, {Key: "secretName", Role: "credential-ref", Kind: store.FieldString}} {
		if secretField(field) || hasSecretField(&store.Namespace{Fields: []store.Field{field}}) {
			t.Fatalf("reference rendered as literal key: %+v", field)
		}
	}
	if !secretField(store.Field{Key: "apiKey", Kind: store.FieldString}) {
		t.Fatal("literal API key lost write-only masking")
	}
}
