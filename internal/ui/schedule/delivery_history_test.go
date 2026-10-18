package schedule

import (
	"errors"
	"testing"
	schedstore "whaleisle/internal/store/schedule"
)

func TestDeliveryHistoryRetryRefreshAndSavedPages(t *testing.T) {
	h := deliveryHistoryState{}
	first := schedstore.DeliveryPage{ID: "task", Records: []schedstore.DeliveryRecord{{MessageID: "new"}}, NextBefore: "cursor"}
	h.finish(first, "", nil)
	h.finish(schedstore.DeliveryPage{}, "cursor", errors.New("offline"))
	if h.RetryBefore != "cursor" || len(h.Page.Records) != 1 || h.RefreshRequired {
		t.Fatal("transport failure lost records or retry cursor")
	}
	h.finish(schedstore.DeliveryPage{}, "cursor", &schedstore.ResultError{Code: "delivery_cursor_not_found"})
	if !h.RefreshRequired || h.RetryBefore != "" || h.Page.Records[0].MessageID != "new" {
		t.Fatal("invalid cursor must restart from newest while retaining visible records")
	}
	older := schedstore.DeliveryPage{ID: "task", Records: []schedstore.DeliveryRecord{{MessageID: "new"}, {MessageID: "old"}}, EarlierRecordsPruned: true, Retention: &schedstore.DeliveryRetention{Days: 7, Records: 20}}
	h.finish(older, "cursor", nil)
	if len(h.Page.Records) != 2 || h.Error != "" || h.RefreshRequired || h.Page.NextBefore != "" {
		t.Fatal("page merge")
	}
	h.finish(first, "", nil)
	if len(h.Page.Records) != 1 || h.Page.EarlierRecordsPruned {
		t.Fatal("refresh must replace old pages")
	}
	h.finish(schedstore.DeliveryPage{}, "cursor", &schedstore.ResultError{Code: "schedule_not_found"})
	if h.RetryBefore != "" || h.Error == "" {
		t.Fatal("missing task retry must use first page")
	}
}
func TestDeliveryHistoryOwnTimezone(t *testing.T) {
	if got := deliveryTime("2026-10-16T01:00:00Z", "Asia/Shanghai"); got != "2026-10-16 09:00:00 CST" {
		t.Fatal(got)
	}
	if got := deliveryTime("unknown", "UTC"); got != "unknown" {
		t.Fatal("changed unknown time")
	}
}
