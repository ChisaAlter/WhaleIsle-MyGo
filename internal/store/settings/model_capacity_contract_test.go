package settings

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/dsh/api"
)

func TestPublishedResultModelCapacityPresence(t *testing.T) {
	var catalog api.ModelCatalog
	if err := json.Unmarshal([]byte(`{"groups":[{"id":"owned","models":[{"id":"present","name":"Present","contextWindow":131072,"maxTokens":8192},{"id":"absent","name":"Absent"},{"id":"explicit-zero","name":"Zero","contextWindow":0,"maxTokens":0}]}]}`), &catalog); err != nil {
		t.Fatal(err)
	}
	s := New()
	s.modelCatalog = catalog
	rows, _, err := s.ConfigurationModels("owned")
	if err != "" || len(rows) != 3 {
		t.Fatal("live models missing")
	}
	present, absent, zero := rows[0].(map[string]any), rows[1].(map[string]any), rows[2].(map[string]any)
	if present["contextWindow"] != float64(131072) || present["maxTokens"] != float64(8192) {
		t.Fatalf("live capacity lost: %+v", present)
	}
	if _, ok := absent["contextWindow"]; ok {
		t.Fatal("absent capacity became zero")
	}
	if _, ok := absent["maxTokens"]; ok {
		t.Fatal("absent output limit became zero")
	}
	if zero["contextWindow"] != float64(0) || zero["maxTokens"] != float64(0) {
		t.Fatal("explicit zero conflated with omitted metadata")
	}
	present["contextWindow"] = float64(1)
	if again, _, _ := s.ConfigurationModels("owned"); again[0].(map[string]any)["contextWindow"] != float64(131072) {
		t.Fatal("public model projection mutated stored Host catalog")
	}
}
