package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDiscoverRetiredQueryCannotReplaceNewAnswerOrOpenedDetail(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	c := New()
	c.Attach(fixtureCall(func(_ context.Context, method string, args, out any) error {
		if method != "pluginMarket/search" {
			return errors.New("unexpected fixture endpoint " + method)
		}
		raw, _ := json.Marshal(args)
		var request struct {
			Query Query `json:"query"`
		}
		_ = json.Unmarshal(raw, &request)
		if request.Query.Text == "older" {
			close(started)
			<-release
		}
		return reply(out, SearchResult{FetchedAt: 1, Total: 1, Entries: []MarketEntry{{Name: request.Query.Text}}})
	}), nil)
	defer c.Detach()
	owner := c.Snapshot().Owner
	url := "https://fixture.invalid/index"
	c.Actions(owner).edit(func(s *State) { s.Discover.Selected = url; s.Discover.Status = "ready" })
	done := make(chan error, 1)
	go func() {
		done <- c.loadSearch(context.Background(), owner, url, Query{Text: "older", Sort: MarketSort{Kind: "relevance"}, Limit: 50}, false)
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("old query did not start")
	}
	c.Actions(owner).Search(Query{Text: "newer", Sort: MarketSort{Kind: "relevance"}})
	waitState(t, c, func(s State) bool {
		return !s.Discover.Loading && s.Discover.Listing != nil && len(s.Discover.Listing.Entries) == 1 && s.Discover.Listing.Entries[0].Name == "newer"
	})
	c.Actions(owner).OpenEntry("newer")
	unblock()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("retired query did not finish")
	}
	s := c.Snapshot()
	if s.Discover.Listing.Entries[0].Name != "newer" || s.Discover.Answered.Text != "newer" || s.Discover.Opened == nil || s.Discover.Opened.Name != "newer" {
		t.Fatalf("old query overwrote current listing/detail: %+v", s.Discover)
	}
}

func TestInstallSettlementKeepsPartialSuccessAndAuthoritativeAttempts(t *testing.T) {
	exit := 4
	s := State{Busy: true, Install: InstallState{Spec: "fixture", Phase: "running", Runs: []InstallRun{{JobID: "first", Output: "partial output"}}}}
	settleInstallState(&s, Change{Application: "failed", Bundle: "fixture", FailedAt: "registry", PackageResult: &PackageResult{ExitCode: &exit}, Registries: []*string{nil, registryPtr(NPMMirrorRegistry)}})
	if s.Busy || s.Install.Phase != "failed" || s.Install.Result.Bundle != "fixture" || !s.Install.Runs[0].Settled || s.Install.Runs[0].ExitCode == nil || *s.Install.Runs[0].ExitCode != 4 || len(s.Install.Attempts) != 2 || s.Install.Attempts[0].Registry != nil {
		t.Fatalf("partial result/log/attempts lost: %+v", s.Install)
	}
	settleInstallState(&s, Change{Application: "restart-required", Bundle: "fixture", RestartReason: "profile restart"})
	if s.Install.Phase != "done" || s.Install.Result.RestartReason != "profile restart" {
		t.Fatal("saved awaiting restart became an installation failure")
	}
}
