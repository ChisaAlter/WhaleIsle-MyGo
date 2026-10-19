package schedule

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
)

func TestAmbientFactsSurviveRefreshFailureWithoutPartialReplacement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var fail atomic.Bool
		var lists atomic.Int32
		client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://ambient-schedule.fixture.invalid"}}
		client.HTTP = &http.Client{Transport: epochRoundTrip(func(r *http.Request) (*http.Response, error) {
			if strings.HasSuffix(r.URL.Path, "schedule/list") {
				lists.Add(1)
				return readbackReply(r, []any{})
			}
			if fail.Load() {
				return nil, errors.New("owned full-catalog failure")
			}
			return readbackReply(r, []map[string]any{{"id": "a", "sessionId": "owner-a", "status": "active", "title": "A"}, {"id": "b", "sessionId": "owner-b", "status": "active", "title": "B"}})
		})}
		s := New()
		s.Attach(client, nil)
		s.Refresh(context.Background(), "owner-a")
		synctest.Wait()
		if len(ActiveForSession(s.Snapshot(), "owner-b")) != 1 {
			t.Fatal("shared catalog did not expose the other owning Session")
		}
		fail.Store(true)
		s.Refresh(context.Background(), "owner-a")
		if len(ActiveForSession(s.Snapshot(), "owner-b")) != 1 {
			t.Fatal("refresh loading blanked known ambient facts")
		}
		synctest.Wait()
		snap := s.Snapshot()
		if snap.Err == "" || snap.CatalogReady || !snap.CatalogSettled || len(ActiveForSession(snap, "owner-b")) != 1 || lists.Load() != 0 {
			t.Fatalf("catalog failure replaced/cleared global facts: %+v partialReads=%d", snap, lists.Load())
		}
		s.Detach()
		s.Attach(client, nil)
		if len(ActiveForSession(s.Snapshot(), "owner-b")) != 0 {
			t.Fatal("reattached Host inherited ambient task marks")
		}
	})
}
