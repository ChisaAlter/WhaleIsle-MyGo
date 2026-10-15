package pet

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestQuickChatRequestRejectsCancelledAndDuplicateResults(t *testing.T) {
	var r quickChatRequest
	ctx, first := r.begin(context.Background(), time.Minute)
	if ctx == nil || first == 0 {
		t.Fatal("request not started")
	}
	if other, _ := r.begin(context.Background(), time.Minute); other != nil {
		t.Fatal("duplicate request admitted")
	}
	r.abort()
	if ctx.Err() == nil || r.finish(first) {
		t.Fatal("cancelled request completed")
	}
	_, next := r.begin(context.Background(), time.Minute)
	if next == first || r.finish(first) || !r.finish(next) || r.finish(next) {
		t.Fatal("stale or double completion accepted")
	}
}

func TestQuickChatInputUsesOriginalUTF16Limit(t *testing.T) {
	if _, err := quickChatText(" \n "); err == nil {
		t.Fatal("accepted empty prompt")
	}
	if _, err := quickChatText(strings.Repeat("鲸", 2000)); err != nil {
		t.Fatal(err)
	}
	if _, err := quickChatText(strings.Repeat("🐳", 1001)); err == nil {
		t.Fatal("did not enforce original JavaScript string length")
	}
}

func TestQuickChatOriginRejectsCredentialAndRouteInjection(t *testing.T) {
	for _, origin := range []string{"", "file:///tmp/host", "https://u:p@host", "https://host/api", "https://host?x=1", "https://host#x", "https://host/?"} {
		if _, err := newQuickChatClient(origin, ""); err == nil {
			t.Fatalf("accepted origin %q", origin)
		}
	}
	if _, err := newQuickChatClient("http://127.0.0.1:8787", ""); err != nil {
		t.Fatal(err)
	}
}

func TestQuickChatReplacementDropsOldHistoryAndCatalog(t *testing.T) {
	st := quickChatSnapshot{SessionID: "old", Groups: []quickChatGroup{{ID: "p"}}, History: []quickChatRow{{Role: "her", Text: "old"}}}
	if !st.replaceHistory(quickChatSnapshot{SessionID: "new"}) {
		t.Fatal("session replacement not reported")
	}
	if len(st.History) != 0 || st.Groups != nil {
		t.Fatal("old shared content retained")
	}
	st.Groups = []quickChatGroup{{ID: "fresh"}}
	if st.replaceHistory(quickChatSnapshot{SessionID: "new", History: []quickChatRow{{Text: "latest"}}}) || len(st.Groups) != 1 {
		t.Fatal("ordinary history poll reset model catalog")
	}
}

func TestQuickChatModelChoiceRequiresCatalogMembership(t *testing.T) {
	st := quickChatSnapshot{Groups: []quickChatGroup{{ID: "p", Models: []quickChatCatalogModel{{ID: "m", Efforts: []quickChatEffort{{ID: "high"}}}}}}}
	if !st.allows(quickChatModel{"p", "m", "high"}) || !st.allows(quickChatModel{"p", "m", ""}) {
		t.Fatal("server model choice rejected")
	}
	if st.allows(quickChatModel{"p", "m", "invented"}) || st.allows(quickChatModel{"other", "m", "high"}) {
		t.Fatal("invented model choice accepted")
	}
}
