package planpreview

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPlanResourceIdentityAndNavigation(t *testing.T) {
	a := PlanAddress("session/a 中文", "call/#1")
	target, ok := ParseAddress(a)
	if !ok || target.SessionID != "session/a 中文" || target.ID != "call/#1" || target.Temporary {
		t.Fatalf("bad roundtrip: %s %+v", a, target)
	}
	for _, bad := range []string{"dsh-resource://plan/a", "dsh-resource://plan/a/b?x=1", "dsh-resource://plan/a/%ZZ", "dsh-resource://plan//b", "file:///tmp/a"} {
		if _, ok := ParseAddress(bad); ok {
			t.Fatalf("accepted %q", bad)
		}
	}
	var r Resources
	doc := Document{Address: PlanAddress("a", "same"), Title: "first", Markdown: "# first\n\ncomplete\nlast"}
	r.Observe(doc)
	r.Observe(Document{Address: doc.Address, Title: "rewrite", Markdown: "wrong"})
	if err := r.Navigate("a", doc.Address); err != "" {
		t.Fatal(err)
	}
	if got, ok := r.Current("a"); !ok || got.Markdown != doc.Markdown {
		t.Fatalf("lost immutable content: %+v %v", got, ok)
	}
	if _, ok := r.Current("b"); ok {
		t.Fatal("plan leaked to another session")
	}
	if err := r.Navigate("b", doc.Address); err == "" {
		t.Fatal("accepted other session address")
	}
	other := Document{Address: PlanAddress("b", "same"), Title: "other", Markdown: "# other"}
	r.Observe(other)
	r.Navigate("b", other.Address)
	if got, _ := r.Current("a"); got.Markdown != doc.Markdown {
		t.Fatal("same call ID collision")
	}
	if err := r.Navigate("a", "dsh-resource://plan/a/%73ame"); err != "" {
		t.Fatal(err)
	}
	if got, ok := r.Current("a"); !ok || got.Markdown != doc.Markdown {
		t.Fatal("equivalent encoded address did not resolve")
	}
	r.Close("a", doc.Address)
	if r.ActiveAddress("a") != "" {
		t.Fatal("close did not dismiss")
	}
	r.Navigate("a", doc.Address)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "complete") {
		t.Fatal("plan body leaked into layout")
	}
	var restored Resources
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.ActiveAddress("a") != doc.Address {
		t.Fatal("lost durable resource address")
	}
	if _, ok := restored.Current("a"); ok {
		t.Fatal("restored stale body")
	}
	restored.Observe(doc)
	if got, ok := restored.Current("a"); !ok || got.Markdown != doc.Markdown {
		t.Fatal("history did not restore addressed body")
	}
	review := Document{Address: ReviewAddress("a", "request/one"), Title: "review", Markdown: "# pending"}
	r.Observe(review)
	r.Navigate("a", review.Address)
	if got, ok := r.Current("a"); !ok || got.Markdown != review.Markdown {
		t.Fatal("review identity lost")
	}
}

func TestPlanResourceHostOwnerIsolation(t *testing.T) {
	var r Resources
	address := PlanAddress("same-session", "same-call")
	r.BindOwner(HostOwner("host-A\x00session-one"))
	r.Observe(Document{Address: address, Markdown: "# Host A secret"})
	r.Navigate("same-session", address)
	review := ReviewAddress("same-session", "same-request")
	r.Observe(Document{Address: review, Markdown: "# Old review"})
	r.Navigate("same-session", review)
	r.BindOwner(HostOwner("host-A\x00session-two"))
	if _, ok := r.Current("same-session"); !ok {
		t.Fatal("same Host lost session tab")
	}
	r.BindOwner("host-B")
	if len(r.Tabs) != 0 || r.ActiveAddress("same-session") != "" || len(r.Documents) != 0 {
		t.Fatal("Host replacement retained prior resources")
	}
	r.Navigate("same-session", address)
	if _, ok := r.Current("same-session"); ok {
		t.Fatal("same sid/call reused previous Host body")
	}
	r.Observe(Document{Address: address, Markdown: "# Host B"})
	if doc, ok := r.Current("same-session"); !ok || doc.Markdown != "# Host B" {
		t.Fatal("new Host could not resolve same identity")
	}
	raw, _ := json.Marshal(r)
	var restored Resources
	json.Unmarshal(raw, &restored)
	restored.BindOwner("host-C")
	if restored.ActiveAddress("same-session") != address {
		t.Fatal("initial recovery lost durable address")
	}
	if _, ok := restored.Current("same-session"); ok {
		t.Fatal("layout restored a foreign Host body")
	}
}

func TestPlanHistoryCannotRepopulateChangedHost(t *testing.T) {
	for _, pair := range [][2]string{{"", "host-A\x00same"}, {"host-A\x00same", "host-B\x00same"}, {"host-A\x00old-session", "host-A\x00new-session"}} {
		if HistoryMatchesBinding(pair[0], pair[1]) {
			t.Fatalf("accepted stale history: %q", pair)
		}
	}
	if !HistoryMatchesBinding("host-B\x00same", "host-B\x00same") {
		t.Fatal("rejected authoritative current snapshot")
	}
}
