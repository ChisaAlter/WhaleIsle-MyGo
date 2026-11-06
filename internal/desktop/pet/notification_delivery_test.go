package pet

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"whaleisle/internal/petwatch"
)

func stickerDeliveryFixture(t *testing.T) (*Companion, *QuickChat, petwatch.Event) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	p := &Companion{win: &mygo.Window{}, quickChatOpen: true}
	q := &QuickChat{owner: p, parent: p.win, win: &mygo.Window{}, life: ctx, cancel: cancel,
		options: QuickChatOptions{Current: func() bool { return true }},
		state:   quickChatSnapshot{SessionID: "first"}, stickerSeen: map[string]bool{}}
	p.QuickChat = q
	e := petwatch.Event{ID: "receipt", MediaPath: "sticker.png", Text: "hello"}
	p.notifications = notificationState{owner: "owner", mediaHome: "home", mediaRoot: "root",
		snapshot: petwatch.Snapshot{Events: []petwatch.Event{e}}, chatImages: map[string]bool{e.ID: true}}
	return p, q, e
}

func notificationReceiptCount(t *testing.T, p *Companion) int {
	t.Helper()
	rec := httptest.NewRecorder()
	p.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "whale-pet://localhost/notifications.json", nil))
	var body struct {
		Events []json.RawMessage `json:"events"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil {
		t.Fatalf("notification response: %d %s", rec.Code, rec.Body.String())
	}
	return len(body.Events)
}

func TestQuickChatStickerDecodeFailureRestoresNotification(t *testing.T) {
	p, q, e := stickerDeliveryFixture(t)
	if got := notificationReceiptCount(t, p); got != 0 {
		t.Fatalf("pending chat receipt unexpectedly visible: %d", got)
	}
	q.finishSticker(0, "owner", e, "home", "root", nil, errors.New("decode failed"), func(string) { t.Fatal("failed image acknowledged") })
	if got := notificationReceiptCount(t, p); got != 1 || len(q.state.History) != 0 {
		t.Fatalf("failed image lost instead of returning to notification: events=%d history=%d", got, len(q.state.History))
	}
}

func TestQuickChatStaleConnectionRestoresPendingNotification(t *testing.T) {
	p, q, _ := stickerDeliveryFixture(t)
	q.options.Current = func() bool { return false }
	if q.check() || notificationReceiptCount(t, p) != 1 {
		t.Fatal("stale open card swallowed its pending sticker")
	}
}

func TestQuickChatStickerCompletionRequiresCurrentSessionAndReceipt(t *testing.T) {
	for _, change := range []string{"session", "receipt", "source"} {
		t.Run(change, func(t *testing.T) {
			p, q, e := stickerDeliveryFixture(t)
			switch change {
			case "session":
				q.stickerGeneration++
			case "receipt":
				p.notifications.snapshot.Events = nil
			case "source":
				p.notifications.mediaHome = "replacement"
			}
			q.finishSticker(0, "owner", e, "home", "root", &ui.Bitmap{}, nil, func(string) { t.Fatal("stale image acknowledged") })
			if len(q.stickers) != 0 || len(q.state.History) != 0 {
				t.Fatal("late image entered a replacement session or source")
			}
		})
	}
}

func TestQuickChatStickerAcknowledgesOnlyCurrentVisibleReceipt(t *testing.T) {
	p, q, e := stickerDeliveryFixture(t)
	acks := 0
	q.finishSticker(0, "owner", e, "home", "root", &ui.Bitmap{}, nil, func(string) { acks++ })
	if len(q.state.History) != 1 || acks != 0 {
		t.Fatalf("decode must append without acknowledging: history=%d acks=%d", len(q.state.History), acks)
	}
	ack := q.state.History[0].Acknowledge
	p.notifications.snapshot.Events = nil
	ack()
	if acks != 0 {
		t.Fatal("removed receipt acknowledged by late paint")
	}
	p.notifications.snapshot.Events = []petwatch.Event{e}
	ack()
	ack()
	if acks != 1 {
		t.Fatalf("current visible receipt acknowledged %d times", acks)
	}
}
