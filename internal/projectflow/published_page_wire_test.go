package projectflow

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type projectResultMemoryTransport func(*http.Request) (*http.Response, error)

func (f projectResultMemoryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestRequestRevertSendsDisplayedRevisionAndPreservesConflict(t *testing.T) {
	calls := 0
	client := Client{Origin: "http://127.0.0.1:1", HTTP: &http.Client{Transport: projectResultMemoryTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var request struct {
			RPCID   string `json:"rpcId"`
			Payload struct {
				ProjectID string `json:"projectId"`
				RequestID string `json:"requestId"`
				Revision  int    `json:"requestRevision"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if r.URL.Path != "/dsh-project/requests/revert" || request.Payload.ProjectID != "owned-project" || request.Payload.RequestID != "A" || request.Payload.Revision != 3 {
			t.Fatalf("Undo replaced the displayed identity/revision: %s %+v", r.URL.Path, request.Payload)
		}
		raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": false, "error": map[string]string{"code": "revert_conflict", "message": "owned A conflict"}}})
		if err != nil {
			t.Fatal(err)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
	})}}
	if err := client.RevertRequest(context.Background(), "owned-project", "A", 3); err == nil || !strings.Contains(err.Error(), "pluginrpc: revert_conflict: owned A conflict") || calls != 1 {
		t.Fatalf("conflict was lost or Undo silently refreshed/retried: err=%v calls=%d", err, calls)
	}
	if err := client.RevertRequest(context.Background(), "owned-project", "A", 0); err == nil || calls != 1 {
		t.Fatalf("an undisplayed revision reached the transport: err=%v calls=%d", err, calls)
	}
}

func TestPublishedPageObjectSurvivesAllProjectResultSnapshots(t *testing.T) {
	for _, height := range []int{80, 2000} {
		t.Run(fmt.Sprint(height), func(t *testing.T) {
			page := fmt.Sprintf(`{"sessionId":"owned-session","callId":"owned-call","title":"Owned result","height":%d,"file":{"attachmentId":"owned-attachment","name":"owned.html","bytes":42}}`, height)
			var card RequestCard
			var delivery HistoricalDelivery
			var activity RequestActivityCard
			for _, decode := range []struct {
				raw string
				out any
			}{{`{"id":"request","delivery":{"pages":[` + page + `]}}`, &card}, {`{"id":"delivery","pages":[` + page + `]}`, &delivery}, {`{"id":"request","delivery":{"pages":[` + page + `]}}`, &activity}} {
				if err := json.Unmarshal([]byte(decode.raw), decode.out); err != nil {
					t.Fatalf("real object-shaped page broke its containing snapshot: %v", err)
				}
			}
			want := DeliveryPageReference{SessionID: "owned-session", CallID: "owned-call", Title: "Owned result", Height: height, File: DeliveryPageFile{AttachmentID: "owned-attachment", Name: "owned.html", Bytes: 42}}
			for _, got := range []DeliveryPageReference{card.Delivery.Pages[0], delivery.Pages[0], activity.Delivery.Pages[0]} {
				if got != want {
					t.Fatalf("page identity/height was lost: got=%+v want=%+v", got, want)
				}
			}
		})
	}
}

func TestPublishedPageWireDoesNotCoerceLegacyStringOrFractionalHeight(t *testing.T) {
	for _, raw := range []string{
		`{"height":80,"file":"owned.html"}`,
		`{"height":80.5,"file":{"attachmentId":"owned","name":"owned.html","bytes":42}}`,
		`{"height":2000,"file":{"attachmentId":"owned","name":"owned.html","bytes":42.5}}`,
	} {
		var page DeliveryPageReference
		if err := json.Unmarshal([]byte(raw), &page); err == nil {
			t.Fatalf("invalid reference shape was silently coerced: %s", raw)
		}
	}
}
