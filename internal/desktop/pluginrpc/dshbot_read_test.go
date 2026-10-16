package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestBotMarkReadOriginalProtocol(t *testing.T) {
	for _, tc := range []struct {
		name, receipt string
		wantErr       bool
	}{
		{"confirmed", `{"view":{"ns":"dsh-bot","revision":9,"value":{"items":[{"id":"b","sessionId":"s","readInitialized":true,"lastSeenSeq":12}]}}}`, false},
		{"monotonic", `{"view":{"value":{"items":[{"id":"b","readInitialized":true,"lastSeenSeq":20}]}}}`, false},
		{"not acknowledged", `{"view":{"value":{"items":[{"id":"b","readInitialized":false,"lastSeenSeq":12}]}}}`, true},
		{"missing", `{"view":{"value":{"items":[]}}}`, true},
		{"behind", `{"view":{"value":{"items":[{"id":"b","readInitialized":true,"lastSeenSeq":11}]}}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			c := Client{BaseURL: "https://fixture.invalid", HTTPClient: &http.Client{Transport: notifyTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/dshbot/bot/mark-read" {
					t.Fatalf("unexpected route %s", r.URL.Path)
				}
				var req struct {
					RPCID   string                     `json:"rpcId"`
					Method  string                     `json:"method"`
					Payload map[string]json.RawMessage `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatal(err)
				}
				if req.Method != "bot/mark-read" || len(req.Payload) != 1 || string(req.Payload["marks"]) != `[{"id":"b","throughSeq":12}]` {
					t.Fatalf("wrong protocol %+v", req)
				}
				raw := `{"type":"server-response","rpcId":"` + req.RPCID + `","result":{"ok":true,"value":` + tc.receipt + `}}`
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(raw)), Header: make(http.Header)}, nil
			})}}
			_, err := c.DshbotMarkRead(context.Background(), []DshbotReadMark{{ID: "b", ThroughSeq: 12}})
			if (err != nil) != tc.wantErr || calls != 1 {
				t.Fatalf("calls=%d error=%v", calls, err)
			}
			for _, marks := range [][]DshbotReadMark{nil, {{ID: "", ThroughSeq: 0}}, {{ID: "b", ThroughSeq: -1}}, make([]DshbotReadMark, 201)} {
				if _, err := c.DshbotMarkRead(context.Background(), marks); err == nil {
					t.Fatal("accepted bad markers")
				}
			}
			if calls != 1 {
				t.Fatal("invalid request transmitted")
			}
		})
	}
}

func TestBotActivityReadCursorDecode(t *testing.T) {
	var a DshbotActivityEntry
	if err := json.Unmarshal([]byte(`{"botId":"b","sessionId":"s","activitySeq":19,"lastSeenSeq":12,"readInitialized":true,"attention":false,"goal":null,"lastRun":null}`), &a); err != nil {
		t.Fatal(err)
	}
	if a.ActivitySeq != 19 || a.LastSeenSeq != 12 || !a.ReadInitialized {
		t.Fatalf("lost host cursor: %+v", a)
	}
}
