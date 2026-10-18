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

type createTransport func(*http.Request) (*http.Response, error)

func (f createTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func createView(revision int) *DshbotRoutineView {
	v := &DshbotRoutineView{NS: "dsh-bot", Revision: revision}
	v.Value.Items = []DshbotCatalogItem{{ID: "b", SessionID: "s"}}
	return v
}
func createCommand() DshbotRoutineCreateCommand {
	return DshbotRoutineCreateCommand{BotID: "b", SessionID: "s", RequestID: "local-request", Revision: 7, BeforeIDs: []string{"old"}, Edit: DshbotRoutineEdit{Name: "New routine", Prompt: "Fixture prompt", Schedule: "0 9 * * *", Timezone: "Asia/Shanghai"}}
}
func TestBotRoutineCreateExactWireAndReceipt(t *testing.T) {
	calls := 0
	cmd := createCommand()
	c := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: createTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		p := req.Payload.(map[string]any)
		if req.Method != "routine/save" || r.URL.Path != "/dshbot/routine/save" {
			t.Fatal(req.Method)
		}
		for _, key := range []string{"id", "requestId", "operationId", "sessionId"} {
			if _, ok := p[key]; ok {
				t.Fatal("invented create payload", key)
			}
		}
		if p["botId"] != "b" || p["revision"] != float64(7) || p["enabled"] != false || p["schedule"] != "0 9 * * *" || p["silentAllowed"] != true || p["watch"] != nil || len(p["contextFrom"].([]any)) != 0 {
			t.Fatal(p)
		}
		v := createView(8)
		v.Value.Routines = []DshbotRoutine{{ID: "new", BotID: "b", Name: cmd.Edit.Name, Prompt: cmd.Edit.Prompt, Schedule: cmd.Edit.Schedule, Timezone: cmd.Edit.Timezone, SilentAllowed: true}}
		raw, _ := json.Marshal(map[string]any{"result": map[string]any{"ok": true, "value": map[string]any{"view": v}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	result, err := c.DshbotCreateRoutine(context.Background(), cmd)
	if err != nil || result.ID != "new" || calls != 1 {
		t.Fatalf("%v %#v %d", err, result, calls)
	}
	c.HTTPClient.Transport = createTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("lost response") })
	if _, err = c.DshbotCreateRoutine(context.Background(), cmd); err == nil || calls != 2 {
		t.Fatal("create was replayed or error hidden")
	}
}
func TestBotRoutineCreateRejectsAmbiguousIDs(t *testing.T) {
	cmd := createCommand()
	v := createView(8)
	r := DshbotRoutine{ID: "old", BotID: "b", Name: cmd.Edit.Name, Prompt: cmd.Edit.Prompt, Schedule: cmd.Edit.Schedule, Timezone: cmd.Edit.Timezone, SilentAllowed: true}
	v.Value.Routines = []DshbotRoutine{r}
	if _, err := DshbotRoutineCreatedID(cmd, v); err == nil {
		t.Fatal("existing row treated as created")
	}
	r.ID = "new"
	v.Value.Routines = append(v.Value.Routines, r)
	if id, err := DshbotRoutineCreatedID(cmd, v); err != nil || id != "new" {
		t.Fatal(id, err)
	}
	r.ID = "new2"
	v.Value.Routines = append(v.Value.Routines, r)
	if _, err := DshbotRoutineCreatedID(cmd, v); err == nil {
		t.Fatal("multiple matches attributed")
	}
	v.Value.Items[0].SessionID = "other"
	if _, err := DshbotRoutineCreatedID(cmd, v); err == nil {
		t.Fatal("canonical replacement accepted")
	}
}

func TestBotRoutineCreateKnownRejectionOnly(t *testing.T) {
	for _, message := range []string{"pluginrpc: failed: Invalid routine timezone.", "pluginrpc: failed: Bot catalog changed. Refresh and try again."} {
		if !routineCreatePreCommitRejection(errors.New(message)) {
			t.Fatal(message)
		}
	}
	for _, message := range []string{"lost response", "pluginrpc: failed: filesystem unavailable", "pluginrpc: HTTP 500: Invalid routine timezone."} {
		if routineCreatePreCommitRejection(errors.New(message)) {
			t.Fatal(message)
		}
	}
}
