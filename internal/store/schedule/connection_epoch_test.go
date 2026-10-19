package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type epochRoundTrip func(*http.Request) (*http.Response, error)

func (f epochRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestScheduleRepliesCannotCrossReattachedClientEpoch(t *testing.T) {
	for _, operation := range []string{"update", "delete", "history"} {
		t.Run(operation, func(t *testing.T) {
			started, release := make(chan struct{}), make(chan struct{})
			client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "http://schedule-fixture.invalid"}, HTTP: &http.Client{Transport: epochRoundTrip(func(r *http.Request) (*http.Response, error) {
				var request struct {
					RPCID   string `json:"rpcId"`
					Payload struct {
						Args struct {
							Request struct{ ID, SessionID string } `json:"request"`
						} `json:"args"`
					} `json:"payload"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					return nil, err
				}
				if r.URL.Path != "/api/schedule/"+operation || request.Payload.Args.Request.ID != "task" || request.Payload.Args.Request.SessionID != "owner" {
					t.Errorf("unexpected schedule request: %s %+v", r.URL.Path, request.Payload.Args.Request)
					return nil, errors.New("unexpected request")
				}
				close(started)
				<-release
				value := map[string]any{"id": "task", "updated": true, "deleted": true, "record": map[string]any{"id": "task", "title": "old reply"}, "records": []any{}}
				raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
				if err != nil {
					return nil, err
				}
				return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
			})}}
			s := New()
			s.Attach(client, nil)
			row := Entry{ID: "task", SessionID: "owner", Origin: client.Endpoint.Origin, Raw: json.RawMessage(`{"id":"task","kind":"at","title":"before","prompt":"fixture","scheduledAt":"2026-12-01T01:00:00Z"}`)}
			done := make(chan error, 1)
			go func() {
				var err error
				switch operation {
				case "update":
					title := "old reply"
					err = s.Update(context.Background(), row, &title, nil, nil)
				case "delete":
					err = s.DeleteEntry(context.Background(), row)
				case "history":
					_, err = s.History(context.Background(), row.SessionID, row.ID)
				}
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("request did not start")
			}
			s.Detach()
			s.Attach(client, nil)
			s.mu.Lock()
			s.busy, s.lastAction, s.err = true, "replacement write", errors.New("replacement error")
			s.rows = []Entry{{ID: "task", SessionID: "owner", Title: "replacement row"}}
			s.mu.Unlock()
			before := s.Snapshot()
			close(release)
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "Host connection changed") {
					t.Fatalf("old reply was accepted: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("old reply did not settle")
			}
			if after := s.Snapshot(); !reflect.DeepEqual(before, after) {
				t.Fatalf("old reply changed replacement state: before=%+v after=%+v", before, after)
			}
		})
	}
}
