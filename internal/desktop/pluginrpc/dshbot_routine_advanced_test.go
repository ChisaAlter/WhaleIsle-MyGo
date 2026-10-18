package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type advancedTransport func(*http.Request) (*http.Response, error)

func (f advancedTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func advancedBaseRow() DshbotRoutine {
	return DshbotRoutine{ID: "r", BotID: "b", Name: "Routine", Prompt: "Fixture prompt", Schedule: "every 1h", Timezone: "UTC", Enabled: true, ContextFrom: []string{"self"}, SilentAllowed: true, Watch: json.RawMessage(`{"kind":"command","value":"fixture-only-command","timeoutMs":15000,"futureField":"preserve"}`)}
}
func TestBotRoutineAdvancedExactPresence(t *testing.T) {
	cases := []struct {
		name  string
		patch DshbotRoutineAdvancedPatch
		keys  []string
	}{
		{"untouched", DshbotRoutineAdvancedPatch{}, nil},
		{"silent", DshbotRoutineAdvancedPatch{SilentSet: true, SilentAllowed: false}, []string{"silentAllowed"}},
		{"context-clear", DshbotRoutineAdvancedPatch{ContextSet: true, ContextFrom: []string{}}, []string{"contextFrom"}},
		{"watch-clear", DshbotRoutineAdvancedPatch{WatchSet: true}, []string{"watch"}},
		{"watch-command", DshbotRoutineAdvancedPatch{WatchSet: true, Watch: &DshbotRoutineWatch{Kind: "command", Value: "  fixture-only-command-new  ", TimeoutMS: 1}}, []string{"watch"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := advancedBaseRow()
			cmd := DshbotRoutineCommand{Action: "save", ID: "r", BotID: "b", Revision: 9, EnabledBefore: true, Edit: DshbotRoutineEdit{Name: r.Name, Prompt: r.Prompt, Schedule: r.Schedule, Timezone: r.Timezone, Enabled: r.Enabled}, Advanced: tc.patch}
			client := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: advancedTransport(func(req *http.Request) (*http.Response, error) {
				calls++
				var request clientRequest
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					t.Fatal(err)
				}
				p := request.Payload.(map[string]any)
				if request.Method != "routine/save" || p["id"] != "r" || p["botId"] != "b" || p["revision"] != float64(9) {
					t.Fatal(request)
				}
				for _, key := range []string{"contextFrom", "silentAllowed", "watch"} {
					_, present := p[key]
					want := false
					for _, k := range tc.keys {
						if k == key {
							want = true
						}
					}
					if present != want {
						t.Fatalf("%s presence=%t want=%t payload=%v", key, present, want, p)
					}
				}
				if tc.patch.ContextSet {
					if !reflect.DeepEqual(p["contextFrom"], []any{}) {
						t.Fatal("clear must send empty array", p)
					}
					r.ContextFrom = []string{}
				}
				if tc.patch.SilentSet {
					if p["silentAllowed"] != false {
						t.Fatal(p)
					}
					r.SilentAllowed = false
				}
				if tc.patch.WatchSet {
					if tc.patch.Watch == nil {
						if p["watch"] != nil {
							t.Fatal(p)
						}
						r.Watch = nil
					} else {
						w := p["watch"].(map[string]any)
						if w["kind"] != "command" || w["value"] != "fixture-only-command-new" || w["timeoutMs"] != float64(1) {
							t.Fatal(w)
						}
						r.Watch, _ = json.Marshal(w)
					}
				}
				for _, key := range []string{"run", "watchHash", "runHistory", "nextRunAt", "triggerToken"} {
					if _, ok := p[key]; ok {
						t.Fatal("unowned field sent", key)
					}
				}
				view := DshbotRoutineView{NS: "dsh-bot", Revision: 10}
				view.Value.Items = []DshbotCatalogItem{{ID: "b", SessionID: "s"}}
				view.Value.Routines = []DshbotRoutine{r}
				raw, _ := json.Marshal(map[string]any{"result": map[string]any{"ok": true, "value": map[string]any{"view": view}}})
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
			})}}
			view, err := client.DshbotControlRoutine(context.Background(), cmd)
			if err != nil || calls != 1 {
				t.Fatal(err, calls)
			}
			if !tc.patch.WatchSet && string(view.Value.Routines[0].Watch) != string(advancedBaseRow().Watch) {
				t.Fatal("untouched unknown watch data not preserved")
			}
		})
	}
}
func TestBotRoutineAdvancedValidationAndClone(t *testing.T) {
	for _, p := range []DshbotRoutineAdvancedPatch{
		{ContextSet: true, ContextFrom: []string{"a", "b", "c", "d"}},
		{ContextSet: true, ContextFrom: []string{"self", "self"}},
		{WatchSet: true, Watch: &DshbotRoutineWatch{Kind: "invented", Value: "x", TimeoutMS: 1}},
		{WatchSet: true, Watch: &DshbotRoutineWatch{Kind: "url", Value: "", TimeoutMS: 1}},
		{WatchSet: true, Watch: &DshbotRoutineWatch{Kind: "command", Value: "x", TimeoutMS: 0}},
		{WatchSet: true, Watch: &DshbotRoutineWatch{Kind: "command", Value: "x", TimeoutMS: 300001}},
		{WatchSet: true, Watch: &DshbotRoutineWatch{Kind: "url", Value: strings.Repeat("😀", 1001), TimeoutMS: 1}},
	} {
		if ValidateDshbotRoutineAdvanced(p) == nil {
			t.Fatal("invalid patch accepted", p)
		}
	}
	p := DshbotRoutineAdvancedPatch{ContextSet: true, ContextFrom: []string{"self"}, WatchSet: true, Watch: &DshbotRoutineWatch{Kind: "url", Value: "http://127.0.0.1", TimeoutMS: 300000}}
	if err := ValidateDshbotRoutineAdvanced(p); err != nil {
		t.Fatal(err)
	}
	clone := CloneDshbotRoutineAdvanced(p)
	clone.ContextFrom[0] = "foreign"
	clone.Watch.Value = "changed"
	if p.ContextFrom[0] != "self" || p.Watch.Value != "http://127.0.0.1" {
		t.Fatal("patch aliased")
	}
	if validateRoutineAdvancedReceipt(p, advancedBaseRow()) == nil {
		t.Fatal("mismatched watch accepted")
	}
}
