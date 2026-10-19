package assistant

import (
	"reflect"
	"testing"

	"whaleisle/internal/desktop/pluginrpc"
)

func taskFilterIDs(rows []pluginrpc.DshbotTask) []string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids
}

func routineFilterIDs(rows []pluginrpc.DshbotRoutine) []string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}
	return ids
}

func TestBotTaskSearchStatusAndOrderingRegression(t *testing.T) {
	rows := []pluginrpc.DshbotTask{
		{ID: "zero", Status: "queued"},
		{ID: "older", Status: "delivered", UpdatedAt: 100, CreatedAt: 1},
		{ID: "newer", Status: "completed", UpdatedAt: 100, CreatedAt: 2},
		{ID: "failure", Status: "failed", UpdatedAt: 200, Error: "Rate limit"},
		{ID: "pause", Status: "paused", UpdatedAt: 50},
		{ID: "cancel", Status: "cancelled", UpdatedAt: 10},
		{ID: "future", Status: "new-host-status", UpdatedAt: 5},
	}
	original := append([]pluginrpc.DshbotTask(nil), rows...)
	want := []string{"failure", "newer", "older", "pause", "cancel", "future", "zero"}
	if got := taskFilterIDs(visibleBotTasks(rows, " \t", "all")); !reflect.DeepEqual(got, want) {
		t.Fatalf("original updated/created ordering: %v; want %v", got, want)
	}
	for i, status := range []string{"queued", "delivered", "completed", "failed", "paused", "cancelled"} {
		if got := visibleBotTasks(rows, "", status); len(got) != 1 || got[0].ID != rows[i].ID {
			t.Fatalf("status %s selected %#v", status, got)
		}
	}
	if got := visibleBotTasks(rows, "  RATE LIMIT ", "failed"); len(got) != 1 || got[0].ID != "failure" {
		t.Fatalf("case/trimmed error search lost failed task: %#v", got)
	}
	if got := visibleBotTasks(rows, "rate limit", "completed"); len(got) != 0 {
		t.Fatal("query bypassed status filter", got)
	}
	if !reflect.DeepEqual(rows, original) {
		t.Fatal("display filtering reordered or changed authoritative input")
	}
}

func TestBotTaskSearchUsesOriginalParticipantAndResultFields(t *testing.T) {
	fields := []pluginrpc.DshbotTask{
		{ID: "needle"}, {Task: "needle"}, {FromName: "needle"}, {ToName: "needle"},
		{FromID: "needle"}, {ToID: "needle"}, {ResultSummary: "needle"}, {Error: "needle"},
	}
	for i, row := range fields {
		if len(visibleBotTasks([]pluginrpc.DshbotTask{row}, " NEEDLE ", "all")) != 1 {
			t.Fatalf("original search field %d unavailable", i)
		}
	}
	if len(visibleBotTasks(nil, "needle", "all")) != 0 {
		t.Fatal("empty catalog produced a match")
	}
}

func TestBotRoutineSearchAndNextRunOrderingRegression(t *testing.T) {
	rows := []pluginrpc.DshbotRoutine{
		{ID: "late", Name: "A", NextRunAt: 20, Enabled: true},
		{ID: "tie-z", Name: "Z", NextRunAt: 10},
		{ID: "no-next-run", Name: "Paused", NextRunAt: 0},
		{ID: "tie-a", Name: "A", NextRunAt: 10, LastError: "Watch FAILED"},
	}
	original := append([]pluginrpc.DshbotRoutine(nil), rows...)
	want := []string{"no-next-run", "tie-a", "tie-z", "late"}
	if got := routineFilterIDs(visibleBotRoutines(rows, "Assistant", "")); !reflect.DeepEqual(got, want) {
		t.Fatalf("next-run/name ordering: %v; want %v", got, want)
	}
	if got := visibleBotRoutines(rows, "Assistant", " watch failed "); len(got) != 1 || got[0].ID != "tie-a" {
		t.Fatalf("last error search: %#v", got)
	}
	if len(visibleBotRoutines(rows, "Assistant", " ASSISTANT ")) != len(rows) {
		t.Fatal("Bot name search lost its routines")
	}
	if len(visibleBotRoutines(rows, "Assistant", "no-match")) != 0 {
		t.Fatal("unmatched query showed routines")
	}
	if !reflect.DeepEqual(rows, original) {
		t.Fatal("routine display sorting changed authoritative input")
	}
	for i, row := range []pluginrpc.DshbotRoutine{{ID: "needle"}, {Name: "needle"}, {Prompt: "needle"}, {LastError: "needle"}, {BotID: "needle"}} {
		if len(visibleBotRoutines([]pluginrpc.DshbotRoutine{row}, "other", "NEEDLE")) != 1 {
			t.Fatalf("routine search field %d unavailable", i)
		}
	}
}
