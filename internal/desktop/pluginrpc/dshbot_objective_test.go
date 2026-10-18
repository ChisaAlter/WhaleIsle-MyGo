package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type objectiveTransport func(*http.Request) (*http.Response, error)

func (f objectiveTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func objectiveStatusFixture() DshbotObjectiveStatus {
	return DshbotObjectiveStatus{Bot: DshbotObjectiveBot{DshbotCatalogItem: DshbotCatalogItem{ID: "b", SessionID: "s"}, Objective: "Fixture objective", ObjectiveMaxRounds: 16}, Revision: 3, GoalKnown: true}
}
func objectiveViewJSON(objective string) string {
	b, _ := json.Marshal(objective)
	return `{"ns":"dsh-bot","revision":3,"value":{"items":[{"id":"b","sessionId":"s","objective":` + string(b) + `,"objectiveMaxRounds":16}]}}`
}
func TestBotObjectiveReadAndExactActionWire(t *testing.T) {
	calls := []string{}
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: objectiveTransport(func(r *http.Request) (*http.Response, error) {
		var req clientRequest
		json.NewDecoder(r.Body).Decode(&req)
		p := req.Payload.(map[string]any)
		calls = append(calls, req.Method)
		response := ""
		switch req.Method {
		case "describe":
			if p["botId"] != "b" || len(p) != 1 {
				t.Fatal(p)
			}
			response = `{"view":` + objectiveViewJSON("Fixture objective") + `}`
		case "bot/activity":
			ids := p["ids"].([]any)
			if len(ids) != 1 || ids[0] != "b" {
				t.Fatal(p)
			}
			response = `{"entries":[{"botId":"b","sessionId":"s","goal":null}]}`
		case "bot/objective/start", "bot/objective/pause":
			if p["id"] != "b" || len(p) != 1 {
				t.Fatal("invented CAS", p)
			}
			response = `{"view":` + objectiveViewJSON("Fixture objective") + `,"goal":{"id":"host-goal","revision":7,"phase":"paused","activation":"disarmed","roundsStarted":2,"maxGoalRounds":16,"blockedReason":{"code":"fixture","message":"fixture-only"}}}`
		case "bot/objective/clear":
			if p["id"] != "b" || p["revision"] != float64(3) || len(p) != 2 {
				t.Fatal("clear CAS", p)
			}
			response = `{"view":` + objectiveViewJSON("") + `,"warnings":["fixture live goal clear incomplete"]}`
		default:
			t.Fatal("invented objective method", req.Method)
		}
		if r.URL.Path != "/dshbot/"+req.Method {
			t.Fatal(r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + response + `}}`))}, nil
	})}}
	state, err := c.DshbotObjective(context.Background(), "b")
	if err != nil || !state.GoalKnown || state.Goal != nil {
		t.Fatal(err, state)
	}
	for _, action := range []string{"start", "pause", "clear"} {
		out, err := c.DshbotControlObjective(context.Background(), DshbotObjectiveCommand{Action: action, Expected: state})
		if err != nil {
			t.Fatal(action, err)
		}
		if action == "clear" && (out.Status.GoalKnown || len(out.Warnings) != 1 || out.Status.Bot.Objective != "") {
			t.Fatal(out)
		}
	}
	if len(calls) != 5 {
		t.Fatal("automatic poll or repeated action", calls)
	}
}
func TestBotObjectivePreflightAndInvalidRequests(t *testing.T) {
	base := objectiveStatusFixture()
	if !SameDshbotObjective(base, base) {
		t.Fatal("same target rejected")
	}
	changes := []func(*DshbotObjectiveStatus){func(s *DshbotObjectiveStatus) { s.Revision++ }, func(s *DshbotObjectiveStatus) { s.Bot.SessionID = "other" }, func(s *DshbotObjectiveStatus) { s.Bot.Objective = "other" }, func(s *DshbotObjectiveStatus) { s.Bot.ObjectiveMaxRounds = 17 }, func(s *DshbotObjectiveStatus) { s.Goal = &DshbotObjectiveGoal{Phase: "active", Activation: "armed"} }}
	for _, change := range changes {
		fresh := CloneDshbotObjective(base)
		change(&fresh)
		if SameDshbotObjective(base, fresh) {
			t.Fatal("changed target passed preflight")
		}
	}
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: objectiveTransport(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid action reached transport")
		return nil, nil
	})}}
	for _, action := range []string{"resume", "save", "stop"} {
		if _, err := c.DshbotControlObjective(context.Background(), DshbotObjectiveCommand{Action: action, Expected: base}); err == nil {
			t.Fatal("invented endpoint")
		}
	}
	base.Bot.Objective = ""
	if _, err := c.DshbotControlObjective(context.Background(), DshbotObjectiveCommand{Action: "start", Expected: base}); err == nil {
		t.Fatal("started unconfigured objective")
	}
}

func TestBotObjectiveClearPreflightKeepsCatalogCAS(t *testing.T) {
	base := objectiveStatusFixture()
	fresh := CloneDshbotObjective(base)
	fresh.Goal = &DshbotObjectiveGoal{Phase: "active", Activation: "armed", RoundsStarted: 4, MaxGoalRounds: 16}
	if !DshbotObjectivePreflightMatches(DshbotObjectiveCommand{Action: "clear", Expected: base}, fresh) {
		t.Fatal("live progress invented a catalog conflict")
	}
	if DshbotObjectivePreflightMatches(DshbotObjectiveCommand{Action: "pause", Expected: base}, fresh) {
		t.Fatal("changed pause target passed")
	}
	fresh.Revision++
	if DshbotObjectivePreflightMatches(DshbotObjectiveCommand{Action: "clear", Expected: base}, fresh) {
		t.Fatal("clear ignored catalog conflict")
	}
}
