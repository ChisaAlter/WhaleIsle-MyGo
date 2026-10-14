package schedule

import (
	"encoding/json"
	"strings"
	"testing"

	"whaleisle/internal/dsh/api"
)

func TestFormatCatalogHint(t *testing.T) {
	a, _ := json.Marshal(map[string]any{"title": "Daily"})
	b, _ := json.Marshal(map[string]any{"name": "Weekly"})
	got := formatCatalogHint(api.CatalogResult{a, b})
	if !strings.Contains(got, "Daily") || !strings.Contains(got, "Weekly") {
		t.Fatalf("%q", got)
	}
}

func TestParseScheduleList(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"id": "s1", "title": "Morning", "cron": "0 9 * * *", "status": "active",
	})
	rows := parseScheduleList(api.ListResult4{raw})
	if len(rows) != 1 || rows[0].ID != "s1" || rows[0].Title != "Morning" {
		t.Fatalf("%+v", rows)
	}
	if rows[0].Detail == "" {
		t.Fatal("expected detail")
	}
	if string(rows[0].Raw) == "" {
		t.Fatal("expected raw for schedule/update expected")
	}
}
