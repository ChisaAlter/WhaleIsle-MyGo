package session

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/dsh/api"
)

// projectionValues decodes one projection-values fixture into the typed record
// the pinned runtime publishes.
func projectionValues(t *testing.T, raw string) api.SessionListValueItemsItemProjectionsValues {
	t.Helper()
	var values api.SessionListValueItemsItemProjectionsValues
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		t.Fatalf("projection values fixture: %v", err)
	}
	return values
}
