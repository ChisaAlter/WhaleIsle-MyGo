package pluginrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type calendarTransport func(*http.Request) (*http.Response, error)

func (f calendarTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBotRoutineCalendarHostRejectionBoundary(t *testing.T) {
	for _, s := range []string{
		"pluginrpc: failed: Invalid cron schedule: Constraint error, got value 61 expected range 0-59",
		"pluginrpc: failed: Advanced schedules must use a five-field cron expression.",
		"pluginrpc: failed: Invalid routine schedule.",
		"pluginrpc: failed: Routine interval is out of range.",
	} {
		if !routineCreatePreCommitRejection(errors.New(s)) {
			t.Fatal("known pre-write rejection not classified", s)
		}
	}
	for _, s := range []string{
		"Invalid cron schedule: Constraint error",
		"pluginrpc: HTTP 500: Invalid cron schedule: Constraint error",
		"pluginrpc: failed: post-write error: Invalid cron schedule: Constraint error",
		"pluginrpc: failed: Invalid cron schedule: ",
		"pluginrpc: failed: disk write failed",
	} {
		if routineCreatePreCommitRejection(errors.New(s)) {
			t.Fatal("uncertain response misclassified", s)
		}
	}
}

func TestBotRoutineCalendarHostScheduleReceipt(t *testing.T) {
	for _, reject := range []bool{false, true} {
		t.Run(map[bool]string{false: "authority", true: "rejected"}[reject], func(t *testing.T) {
			calls := 0
			cmd := DshbotRoutineCreateCommand{BotID: "b", SessionID: "s", RequestID: "local-fixture", Revision: 7, Edit: DshbotRoutineEdit{Name: "Month end", Prompt: "Fixture prompt", Schedule: "30 2 31 * *", Timezone: "America/New_York"}}
			client := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: calendarTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				var request clientRequest
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				payload := request.Payload.(map[string]any)
				if request.Method != "routine/save" || payload["schedule"] != cmd.Edit.Schedule || payload["timezone"] != cmd.Edit.Timezone {
					t.Fatal("changed schedule or endpoint", request)
				}
				for _, key := range []string{"id", "requestId", "nextRunAt"} {
					if _, ok := payload[key]; ok {
						t.Fatal("client invented identity/time", key)
					}
				}
				var result map[string]any
				if reject {
					result = map[string]any{"ok": false, "error": map[string]any{"code": "failed", "message": "Invalid cron schedule: Constraint error"}}
				} else {
					view := DshbotRoutineView{NS: "dsh-bot", Revision: 8}
					view.Value.Items = []DshbotCatalogItem{{ID: "b", SessionID: "s"}}
					view.Value.Routines = []DshbotRoutine{{ID: "new", BotID: "b", Name: cmd.Edit.Name, Prompt: cmd.Edit.Prompt, Schedule: cmd.Edit.Schedule, Timezone: cmd.Edit.Timezone, SilentAllowed: true, NextRunAt: 987654321}}
					result = map[string]any{"ok": true, "value": map[string]any{"view": view}}
				}
				raw, _ := json.Marshal(map[string]any{"result": result})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
			})}}
			result, err := client.DshbotCreateRoutine(context.Background(), cmd)
			if calls != 1 {
				t.Fatal("creation was retried")
			}
			if reject {
				if err == nil || !result.Rejected {
					t.Fatal("Host rejection not preserved", result, err)
				}
			} else if err != nil || result.ID != "new" || result.View.Value.Routines[0].NextRunAt != 987654321 {
				t.Fatal("Host authority altered", result, err)
			}
		})
	}
}
