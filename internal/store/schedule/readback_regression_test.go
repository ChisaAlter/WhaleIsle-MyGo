package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
)

// These fixtures use a RoundTripper only: no listener, real Host, or scheduler.
func readbackReply(r *http.Request, value any) (*http.Response, error) {
	var body struct {
		RPCID string `json:"rpcId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": body.RPCID, "result": map[string]any{"ok": true, "value": value}})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
}

func TestUpdatePublishesCanonicalInstantAndKeepsNewerReadback(t *testing.T) {
	for _, newerCatalog := range []bool{false, true} {
		t.Run(map[bool]string{false: "canonical metadata", true: "newer catalog wins"}[newerCatalog], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				before := json.RawMessage(`{"id":"task","kind":"daily","title":"Before","prompt":"fixture","time":"09:00:00","timeZone":"UTC","scheduledAt":"2026-12-01T09:00:00Z","lastDelivery":{"messageId":"saved-receipt"}}`)
				committed := json.RawMessage(`{"id":"task","kind":"daily","title":"Committed","prompt":"fixture","time":"19:30:00","timeZone":"Asia/Shanghai","scheduledAt":"2026-12-01T11:30:00Z"}`)
				newer := json.RawMessage(`{"id":"task","kind":"daily","title":"Newer catalog","prompt":"fixture","time":"20:00:00","timeZone":"Asia/Shanghai","scheduledAt":"2026-12-02T12:00:00Z"}`)
				s := New()
				release := make(chan struct{})
				client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://schedule-readback.fixture.invalid"}}
				client.HTTP = &http.Client{Transport: epochRoundTrip(func(r *http.Request) (*http.Response, error) {
					if strings.HasSuffix(r.URL.Path, "schedule/update") {
						if newerCatalog {
							s.mu.Lock()
							fresh := parseScheduleList([]json.RawMessage{newer})[0]
							fresh.Origin, fresh.SessionID, fresh.Status = client.Endpoint.Origin, "owner", "active"
							s.rows = []Entry{fresh}
							s.mu.Unlock()
						}
						return readbackReply(r, map[string]any{"id": "task", "updated": true, "record": committed})
					}
					select {
					case <-release:
					case <-r.Context().Done():
						return nil, r.Context().Err()
					}
					return readbackReply(r, []any{})
				})}
				s.Attach(client, nil)
				row := parseScheduleList([]json.RawMessage{before})[0]
				row.Origin, row.SessionID, row.Status = client.Endpoint.Origin, "owner", "active"
				s.rows = []Entry{row}
				title := "Committed"
				if err := s.Update(ctx, row, &title, nil, nil); err != nil {
					t.Fatal(err)
				}
				shown := s.Snapshot().Rows[0]
				if newerCatalog {
					if shown.Title != "Newer catalog" || shown.ScheduledAt != "2026-12-02T12:00:00Z" {
						t.Fatalf("late save regressed authoritative rule: %+v", shown)
					}
				} else {
					if shown.Title != "Committed" || shown.ScheduledAt != "2026-12-01T11:30:00Z" || shown.TimeZone != "Asia/Shanghai" || !strings.Contains(string(shown.Raw), "saved-receipt") {
						t.Fatalf("committed metadata inconsistent: %+v", shown)
					}
				}
				close(release)
				synctest.Wait()
			})
		})
	}
}

func TestReceiptAbsenceRequiresSuccessfulPostReceiptCatalog(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		type answer struct {
			value any
			err   error
		}
		queries := make(chan chan answer, 8)
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://receipt-ordinal.fixture.invalid"}}
		client.HTTP = &http.Client{Transport: epochRoundTrip(func(r *http.Request) (*http.Response, error) {
			if !strings.HasSuffix(r.URL.Path, "schedule/catalog") {
				return nil, errors.New("isolated catalog unavailable")
			}
			reply := make(chan answer, 1)
			queries <- reply
			var next answer
			select {
			case next = <-reply:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
			if next.err != nil {
				return nil, next.err
			}
			return readbackReply(r, next.value)
		})}
		s := New()
		s.Attach(client, nil)
		s.Refresh(ctx, "viewer")
		before := <-queries
		s.EnsureContext(ctx, "viewer", []string{"created"})
		after := <-queries
		row := Entry{ID: "created", SessionID: "viewer", Origin: client.Endpoint.Origin}
		before <- answer{value: []any{}}
		synctest.Wait()
		if registered, settled := s.ReceiptCatalogState(row); !registered || settled {
			t.Fatal("pre-receipt catalog proved absence")
		}
		after <- answer{err: errors.New("owned read failed")}
		synctest.Wait()
		if _, settled := s.ReceiptCatalogState(row); settled {
			t.Fatal("failed post-receipt read erased creation identity")
		}
		s.Refresh(ctx, "viewer")
		success := <-queries
		success <- answer{value: []any{}}
		synctest.Wait()
		if _, settled := s.ReceiptCatalogState(row); !settled {
			t.Fatal("fresh full catalog did not establish absence")
		}
		s.Detach()
		s.Attach(client, nil)
		if registered, _ := s.ReceiptCatalogState(row); registered {
			t.Fatal("new Host attachment inherited old receipt authority")
		}
	})
}

func TestDeleteOutcomeNoticeSurvivesDetailAndRetiresWithHost(t *testing.T) {
	for _, tc := range []struct {
		name    string
		result  any
		outcome string
		failed  bool
	}{
		{"deleted", map[string]any{"deleted": true}, "deleted", false},
		{"already absent", map[string]any{"deleted": false, "code": "schedule_not_found"}, "alreadyAbsent", false},
		{"refused", map[string]any{"deleted": false, "code": "internal_error"}, "failed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://delete-notice.fixture.invalid"}}
				client.HTTP = &http.Client{Transport: epochRoundTrip(func(r *http.Request) (*http.Response, error) {
					if strings.HasSuffix(r.URL.Path, "schedule/delete") {
						return readbackReply(r, tc.result)
					}
					return readbackReply(r, []any{})
				})}
				s := New()
				s.Attach(client, nil)
				err := s.DeleteEntry(context.Background(), Entry{ID: "owned", SessionID: "owner", Origin: client.Endpoint.Origin})
				if (err != nil) != tc.failed {
					t.Fatalf("unexpected outcome: %v", err)
				}
				notice, fresh := s.ConsumeDeletionNotice()
				if !fresh || notice.Sequence == 0 || notice.Outcome != tc.outcome || notice.ConnectionEpoch != s.Snapshot().ConnectionEpoch {
					t.Fatalf("wrong notice: %+v fresh=%v", notice, fresh)
				}
				if again, fresh := s.ConsumeDeletionNotice(); fresh || again != notice {
					t.Fatal("repainting replayed or dropped the global outcome")
				}
				if err := s.DeleteEntry(context.Background(), Entry{ID: "owned", SessionID: "owner", Origin: client.Endpoint.Origin}); (err != nil) != tc.failed {
					t.Fatal("replacement deletion did not settle")
				}
				s.DismissDeletionNotice(notice)
				if latest, fresh := s.ConsumeDeletionNotice(); !fresh || latest.Sequence <= notice.Sequence {
					t.Fatal("old toast close dismissed a newer deletion outcome")
				}
				synctest.Wait()
				s.Detach()
				s.Attach(client, nil)
				if stale, fresh := s.ConsumeDeletionNotice(); fresh || stale.Sequence != 0 {
					t.Fatal("reattached Host inherited prior deletion outcome")
				}
			})
		})
	}
}
