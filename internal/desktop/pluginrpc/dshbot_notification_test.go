package pluginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"whaleisle/internal/desktop/botnotify"
)

type botNotifyTransport func(*http.Request) (*http.Response, error)

func (f botNotifyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	return fixtureResponseIdentity(r, response, err)
}
func TestBotNotifyReadOnlyExactBatch(t *testing.T) {
	var methods []string
	c := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: botNotifyTransport(func(r *http.Request) (*http.Response, error) {
		var req struct {
			Method  string                     `json:"method"`
			Payload map[string]json.RawMessage `json:"payload"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		methods = append(methods, req.Method)
		value := `{"view":{"ns":"dsh-bot","revision":4,"value":{"items":[{"id":"b","kind":"bot","sessionId":"s","name":"Bot","notify":true},{"id":"other","kind":"room","sessionId":"s2","notify":false,"hidden":true}]}}}`
		if req.Method == "bot/activity" {
			if len(req.Payload) != 1 || string(req.Payload["ids"]) != `["b","other"]` {
				t.Fatalf("global/wrong activity request: %s", req.Payload)
			}
			value = `{"entries":[{"botId":"b","sessionId":"s","attention":true,"activitySeq":8,"goal":{"phase":"blocked"},"lastRun":{"routineId":"r","runId":"run","status":"interrupted","silent":false,"endedAt":123}},{"botId":"other","sessionId":"s2","attention":false,"activitySeq":0,"goal":null,"lastRun":null}]}`
		} else if req.Method != "describe" || len(req.Payload) != 0 {
			t.Fatalf("unrequested method %s", req.Method)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + value + `}}`))}, nil
	})}}
	out, err := c.DshbotNotificationSnapshot(context.Background())
	if err != nil || out.Revision != 4 || len(out.Rows) != 2 || !out.Rows[0].Notify || out.Rows[0].GoalPhase != "blocked" || out.Rows[0].Run.Status != "interrupted" {
		t.Fatalf("real fields lost: %#v %v", out, err)
	}
	if !out.Rows[1].Hidden || out.Rows[1].Kind != "room" || out.Rows[1].Notify {
		t.Fatal("hidden/room scope or per-item notify lost")
	}
	if !reflect.DeepEqual(methods, []string{"describe", "bot/activity"}) {
		t.Fatal(methods)
	}
}
func TestBotNotifyRejectsUnavailableOrChangedEvidence(t *testing.T) {
	for _, tc := range []struct{ name, profile, activity string }{
		{"missing flag", `{"id":"b","sessionId":"s"}`, `null`},
		{"null flag", `{"id":"b","sessionId":"s","notify":null}`, `null`},
		{"wrong flag", `{"id":"b","sessionId":"s","notify":"true"}`, `null`},
		{"wrong session", `{"id":"b","sessionId":"s","notify":true}`, `[{"botId":"b","sessionId":"other","goal":null,"lastRun":null}]`},
		{"missing status", `{"id":"b","sessionId":"s","notify":true}`, `[{"botId":"b","sessionId":"s"}]`},
		{"unrequested bot", `{"id":"b","sessionId":"s","notify":true}`, `[{"botId":"foreign","sessionId":"s","goal":null,"lastRun":null}]`},
		{"duplicate", `{"id":"b","sessionId":"s","notify":true}`, `[{"botId":"b","sessionId":"s","goal":null,"lastRun":null},{"botId":"b","sessionId":"s","goal":null,"lastRun":null}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: botNotifyTransport(func(r *http.Request) (*http.Response, error) {
				value := `{"view":{"ns":"dsh-bot","revision":1,"value":{"items":[` + tc.profile + `]}}}`
				if strings.HasSuffix(r.URL.Path, "bot/activity") {
					value = `{"entries":` + tc.activity + `}`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + value + `}}`))}, nil
			})}}
			if _, err := c.DshbotNotificationSnapshot(context.Background()); err == nil {
				t.Fatal("unavailable evidence accepted")
			}
		})
	}
}

func TestBotNotifyCatalogScopeAndBounds(t *testing.T) {
	for _, count := range []int{0, 201} {
		calls := 0
		c := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: botNotifyTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if !strings.HasSuffix(r.URL.Path, "describe") {
				t.Fatal("empty or oversized catalog sent an activity request")
			}
			items := make([]map[string]any, count)
			for i := range items {
				items[i] = map[string]any{"id": fmt.Sprint(i), "sessionId": fmt.Sprint("s", i), "notify": true}
			}
			raw, _ := json.Marshal(map[string]any{"result": map[string]any{"ok": true, "value": map[string]any{"view": map[string]any{"ns": "dsh-bot", "revision": 1, "value": map[string]any{"items": items}}}}})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
		})}}
		result, err := c.DshbotNotificationSnapshot(context.Background())
		if calls != 1 || (count == 0 && (err != nil || len(result.Rows) != 0)) || (count > 200 && (err == nil || !strings.Contains(err.Error(), "200"))) {
			t.Fatalf("count=%d calls=%d err=%v", count, calls, err)
		}
	}
}

func TestBotNotifyPartialActivityFailure(t *testing.T) {
	c := &Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: botNotifyTransport(func(r *http.Request) (*http.Response, error) {
		value := `{"view":{"ns":"dsh-bot","revision":4,"value":{"items":[{"id":"good","sessionId":"sg","notify":true,"title":" Display title "},{"id":"cold","sessionId":"sc","notify":true}]}}}`
		if strings.HasSuffix(r.URL.Path, "bot/activity") {
			value = `{"entries":[{"botId":"good","sessionId":"sg","attention":true,"activitySeq":2,"goal":null,"lastRun":null},{"botId":"cold","sessionId":"sc","error":"fixture inspection unavailable"}]}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + value + `}}`))}, nil
	})}}
	var tracker botnotify.Tracker
	tracker.Observe(botnotify.Snapshot{Revision: 4, Rows: []botnotify.Row{{ID: "good", SessionID: "sg", Notify: true, ActivitySeq: 1}, {ID: "cold", SessionID: "sc", Notify: true, ActivitySeq: 1}}}, false, true)
	out, err := c.DshbotNotificationSnapshot(context.Background())
	if err != nil || out.Unavailable != 1 || len(out.Rows) != 1 || out.Rows[0].ID != "good" || out.Rows[0].Name != "Display title" {
		t.Fatalf("partial evidence lost: %#v %v", out, err)
	}
	notices := tracker.Observe(out, false, true)
	if len(notices) != 1 || notices[0].Ticket.BotID != "good" {
		t.Fatal("cold item blocked healthy attention")
	}
	out.Unavailable = 0
	out.Rows = append(out.Rows, botnotify.Row{ID: "cold", SessionID: "sc", Notify: true, Attention: true, ActivitySeq: 1})
	if len(tracker.Observe(out, false, true)) != 0 {
		t.Fatal("recovered cold row skipped new baseline")
	}
}
