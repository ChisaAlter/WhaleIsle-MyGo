package accountlogin

import (
	"errors"
	"testing"
	"whaleisle/internal/dsh/api"
)

func TestExplicitLoginFixture(t *testing.T) {
	var f Flow
	link := "https://example.test/authorize"
	view := api.AccountView{Attempt: api.AccountViewAttempt{Id: "a", Phase: "waiting-browser", AuthorizeUrl: &link}}
	calls := 0
	open := func(string) error { calls++; return errors.New("no browser") }
	f.Observe(view, open)
	if calls != 0 {
		t.Fatal("passive state opened browser")
	}
	if !f.Begin() || f.Begin() {
		t.Fatal("duplicate start accepted")
	}
	f.Started(view, nil, open)
	f.Observe(view, open)
	if calls != 1 {
		t.Fatalf("opens = %d", calls)
	}
	if s := f.Snapshot(); s.URL != link || s.Error == "" || s.Busy {
		t.Fatalf("failed open lost recovery: %+v", s)
	}
	if !Active("waiting-browser") || Active("succeeded") {
		t.Fatal("phase projection")
	}
	if f.Begin() {
		t.Fatal("active attempt restarted")
	}
	view.Attempt.Phase = "cancelled"
	f.Observe(view, open)
	if !f.Begin() {
		t.Fatal("retry unavailable")
	}
	f.Started(api.AccountView{}, errors.New("mock start failed"), open)
	if s := f.Snapshot(); s.Error == "" || s.Busy {
		t.Fatal("start failure missing")
	}
}
