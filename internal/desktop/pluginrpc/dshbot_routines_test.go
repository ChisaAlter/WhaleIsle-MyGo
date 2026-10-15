package pluginrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type routineTransport func(*http.Request) (*http.Response, error)

func (f routineTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func routineFixtureView() *DshbotRoutineView {
	v := &DshbotRoutineView{NS: "dsh-bot", Revision: 8}
	v.Value.Items = []DshbotCatalogItem{{ID: "b", Name: "Bot B", SessionID: "s"}}
	v.Value.Routines = []DshbotRoutine{{ID: "r", BotID: "b", Name: "Check", Prompt: "Inspect fixture", Schedule: "every 1h", Timezone: "UTC", Enabled: true, RunHistory: []DshbotRoutineRun{}}}
	return v
}
func TestBotRoutineWireContract(t *testing.T) {
	calls := 0
	view := routineFixtureView()
	c := &Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: routineTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req clientRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		want := map[string]any{"id": "r", "botId": "b", "revision": float64(7), "name": "Check", "prompt": "Inspect fixture", "schedule": "every 1h", "timezone": "UTC", "maxRuns": float64(0), "enabled": true}
		if req.Method != "routine/save" || r.URL.Path != "/dshbot/routine/save" || !reflect.DeepEqual(req.Payload, want) {
			t.Fatalf("wrong wire: %s %#v", req.Method, req.Payload)
		}
		raw, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": req.RPCID, "result": map[string]any{"ok": true, "value": map[string]any{"view": view}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})}}
	command := DshbotRoutineCommand{Action: "save", ID: "r", BotID: "b", Revision: 7, Edit: DshbotRoutineEdit{Name: "Check", Prompt: "Inspect fixture", Schedule: "every 1h", Timezone: "UTC", Enabled: true}}
	if _, err := c.DshbotControlRoutine(context.Background(), command); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	for _, action := range []string{"create", "trigger-token", "unsupported"} {
		command.Action = action
		if _, err := c.DshbotControlRoutine(context.Background(), command); err == nil {
			t.Fatal(action)
		}
	}
	if calls != 1 {
		t.Fatal("invalid operation reached transport")
	}
}
func TestBotRoutineReadAndNoRetry(t *testing.T) {
	calls := 0
	c := &Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: routineTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req clientRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Method != "describe" {
			t.Fatal("read performed mutation", req.Method)
		}
		v := routineFixtureView()
		raw, _ := json.Marshal(map[string]any{"result": map[string]any{"ok": true, "value": map[string]any{"view": v}}})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	})}}
	if v, err := c.DshbotRoutines(context.Background(), "b"); err != nil || len(v.Value.Routines) != 1 {
		t.Fatalf("%v %#v", err, v)
	}
	c.HTTPClient.Transport = routineTransport(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("ambiguous response") })
	if _, err := c.DshbotControlRoutine(context.Background(), DshbotRoutineCommand{Action: "run", ID: "r", BotID: "b", Revision: 8}); err == nil {
		t.Fatal("transport failure swallowed")
	}
	if calls != 2 {
		t.Fatal("automatic retry", calls)
	}
}
func TestBotRoutineReceiptValidation(t *testing.T) {
	command := DshbotRoutineCommand{Action: "toggle", ID: "r", BotID: "b", Revision: 7, EnabledBefore: true}
	v := routineFixtureView()
	if err := ValidateDshbotRoutineReceipt(command, v); err == nil {
		t.Fatal("unchanged toggle accepted")
	}
	v.Value.Routines[0].Enabled = false
	if err := ValidateDshbotRoutineReceipt(command, v); err != nil {
		t.Fatal(err)
	}
	command.Action = "delete"
	if err := ValidateDshbotRoutineReceipt(command, v); err == nil {
		t.Fatal("undeleted row accepted")
	}
	v.Value.Routines = nil
	if err := ValidateDshbotRoutineReceipt(command, v); err != nil {
		t.Fatal(err)
	}
	v.Value.Items[0].Kind = "room"
	if err := ValidateDshbotRoutineReceipt(command, v); err == nil {
		t.Fatal("room accepted as routine Bot")
	}
}

func TestBotRoutineEditAndScheduleReceipt(t *testing.T) {
	e := DshbotRoutineEdit{Name: "Fixture", Prompt: "Inspect", Schedule: "EVERY 001H", Timezone: "UTC", Enabled: true}
	if err := ValidateDshbotRoutineEdit(e); err != nil {
		t.Fatal(err)
	}
	v := routineFixtureView()
	v.Value.Routines[0].Name = e.Name
	v.Value.Routines[0].Prompt = e.Prompt
	cmd := DshbotRoutineCommand{Action: "save", ID: "r", BotID: "b", Revision: 7, Edit: e}
	if err := ValidateDshbotRoutineReceipt(cmd, v); err != nil {
		t.Fatal(err)
	}
	v.Value.Routines[0].Schedule = "every 2h"
	if err := ValidateDshbotRoutineReceipt(cmd, v); err == nil {
		t.Fatal("wrong schedule accepted")
	}
	e.MaxRuns = -1
	if err := ValidateDshbotRoutineEdit(e); err == nil {
		t.Fatal("negative run limit accepted")
	}
	e.MaxRuns = 0
	e.Prompt = " "
	if err := ValidateDshbotRoutineEdit(e); err == nil {
		t.Fatal("empty prompt accepted")
	}
}
