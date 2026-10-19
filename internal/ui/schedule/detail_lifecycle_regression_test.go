package schedule

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"

	"whaleisle/internal/dsh"
	schedstore "whaleisle/internal/store/schedule"
)

func TestLateSaveCannotReachReopenedOrDetachedDetail(t *testing.T) {
	for _, transition := range []string{"A/B/A", "unmount", "reattach"} {
		t.Run(transition, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				started, release := make(chan struct{}), make(chan struct{})
				row := draftFixture("2026-12-01T09:00:00Z", "A", "committed instruction")
				client := &dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://detail-owner.fixture.invalid"}}
				row.Origin = client.Endpoint.Origin
				client.HTTP = &http.Client{Transport: scheduleUITransport(func(r *http.Request) (*http.Response, error) {
					var request struct {
						RPCID string `json:"rpcId"`
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						return nil, err
					}
					var value any = []any{}
					if strings.HasSuffix(r.URL.Path, "schedule/update") {
						close(started)
						<-release
						value = map[string]any{"id": row.ID, "updated": true, "record": row.Raw}
					}
					raw, err := json.Marshal(map[string]any{"type": "server-response", "rpcId": request.RPCID, "result": map[string]any{"ok": true, "value": value}})
					if err != nil {
						return nil, err
					}
					return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
				})}
				store := schedstore.New()
				store.Attach(client, nil)
				p := &Panel{}
				p.OpenTask(row, store.Snapshot().ConnectionEpoch)
				draft := p.drafts[p.selected]
				draft.Prompt = "pending edit"
				p.saveDraft(Model{Store: store}, draft)
				<-started
				switch transition {
				case "A/B/A":
					other := row
					other.ID = "B"
					p.OpenTask(other, p.connectionEpoch)
					p.OpenTask(row, p.connectionEpoch)
				case "unmount":
					p.Deactivate()
				case "reattach":
					store.Detach()
					store.Attach(client, nil)
				}
				close(release)
				synctest.Wait()
				p.mu.Lock()
				if p.status != "" || len(p.committed) != 0 {
					t.Errorf("late save published into replacement detail: status=%q committed=%v", p.status, p.committed)
				}
				p.mu.Unlock()
				if transition == "A/B/A" && p.drafts[p.selected].Prompt != "committed instruction" {
					t.Fatal("late save changed the reopened draft")
				}
			})
		})
	}
}

func TestTaskSwitchDropsDraftButSameTaskKeepsFailure(t *testing.T) {
	a := draftFixture("2026-12-01T09:00:00Z", "A", "first")
	b := a
	b.ID, b.Title = "second", "B"
	p := &Panel{}
	p.OpenTask(a, 4)
	first := p.drafts[p.selected]
	first.Prompt = "failed edit"
	p.actionFailed, p.status = true, "conflict"
	p.OpenTask(a, 4)
	if p.drafts[p.selected] != first || !p.actionFailed || p.status != "conflict" {
		t.Fatal("same detail lost its failed draft")
	}
	p.OpenTask(b, 4)
	p.OpenTask(a, 4)
	if p.drafts[p.selected] == first || p.drafts[p.selected].Prompt != "first" || p.actionFailed || p.status != "" {
		t.Fatal("A/B/A reused the retired detail state")
	}
	p.Deactivate()
	if p.selected != "" || p.initialized || len(p.drafts) != 0 {
		t.Fatal("unmount retained an editable detail")
	}
}

func TestMissingDetailWaitsForAuthoritativeDeletionReadback(t *testing.T) {
	row := draftFixture("2026-12-01T09:00:00Z", "A", "first")
	p := &Panel{}
	p.OpenTask(row, 4)
	snap := schedstore.ViewState{CatalogReady: true, CatalogRevision: 12}
	p.actionFailed = true
	if p.reconcileDetail(snap) || p.selected == "" {
		t.Fatal("missing row hid a failed draft")
	}
	p.actionFailed = false
	p.actionPending = true
	if p.reconcileDetail(snap) || p.selected == "" {
		t.Fatal("missing row retired an in-flight mutation")
	}
	p.actionPending = false
	p.deleteAck = &deleteReceipt{key: taskKey(row), revision: 12}
	if p.reconcileDetail(snap) || p.selected == "" {
		t.Fatal("delete ack used a pre-ack catalog")
	}
	snap.CatalogRevision++
	snap.CatalogReady = false
	if p.reconcileDetail(snap) || p.selected == "" {
		t.Fatal("failed/fallback catalog confirmed global absence")
	}
	snap.CatalogReady = true
	if !p.reconcileDetail(snap) || p.selected != "" {
		t.Fatal("fresh authoritative absence did not complete own deletion")
	}
	p.OpenTask(row, 4)
	if p.reconcileDetail(snap) || p.selected != "" {
		t.Fatal("external absence must clear detail without closing a task tab")
	}
}

func TestTimingKindRestoreAndUntouchedMilliseconds(t *testing.T) {
	row := schedstore.Entry{Raw: json.RawMessage(`{"kind":"daily","time":"23:12:13.123","timeZone":"Asia/Shanghai","scheduledAt":"2026-12-06T15:12:13.123Z"}`)}
	d := newTaskDraft(row)
	d.Clock = "22:00:00"
	d.chooseKind("weekly")
	if d.Clock != "22:00:00" || d.Zone != "Asia/Shanghai" || d.Weekdays != "7" {
		t.Fatalf("clock choice did not carry local Sunday: %+v", d)
	}
	d.Weekdays = "1,3,5"
	d.chooseKind("daily")
	if d.Clock != "23:12:13.123" {
		t.Fatal("returning to stored daily kind did not restore committed clock")
	}
	d.chooseKind("weekly")
	if d.Weekdays != "1,3,5" {
		t.Fatal("clock detour discarded chosen weekdays")
	}
	d.chooseKind("every")
	if d.Seconds != "3600" {
		t.Fatal("new interval was not seeded to one hour")
	}
	d = newTaskDraft(schedstore.Entry{Raw: json.RawMessage(`{"kind":"at","scheduledAt":"2026-12-01T09:12:13.123Z"}`)})
	d.Zone = "UTC"
	date, clock := d.wallClock()
	if clock != "09:12:13.123" {
		t.Fatal("stored millisecond lost before clock editing")
	}
	if err := d.pickWallClock(date, clock, "Asia/Shanghai"); err != nil {
		t.Fatal(err)
	}
	raw, err := d.change()
	if err != nil || !strings.Contains(string(raw), "09:12:13.123+08:00") {
		t.Fatalf("zone edit discarded untouched fraction: %s %v", raw, err)
	}
}

func TestLeavingRecordsRetiresHistoryAndKeepsRuleDraft(t *testing.T) {
	row := draftFixture("2026-12-01T09:00:00Z", "A", "first")
	p := &Panel{}
	p.OpenTask(row, 4)
	d := p.drafts[p.selected]
	d.Prompt = "current draft"
	p.detailTab, p.historyKey, p.historyLoading = "records", taskKey(row), true
	request := p.historyRequest
	p.selectDetailTab("rule")
	if p.historyRequest <= request || p.historyLoading || p.historyKey != "" || p.drafts[p.selected] != d || d.Prompt != "current draft" {
		t.Fatal("records unmount did not retire just its own reader")
	}
	row.Raw = json.RawMessage(`{"lastDelivery":{"messageId":"receipt-new"}}`)
	if latestDelivery(row) != "receipt-new" {
		t.Fatal("catalog receipt identity not projected for history refresh")
	}
}
