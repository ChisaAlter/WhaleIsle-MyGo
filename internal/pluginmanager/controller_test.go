package pluginmanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"whaleisle/internal/dsh"
)

type fixtureCall func(context.Context, string, any, any) error

func TestWireLocalizedTextUsesHostStringAndDictionary(t *testing.T) {
	var bundles []Bundle
	if err := json.Unmarshal([]byte(`[{"name":"fixture","meta":{"title":{"en":"Fixture title","zh":"示例标题"},"description":"Literal description"},"rows":[]}]`), &bundles); err != nil {
		t.Fatal(err)
	}
	if bundles[0].Meta.Title.Resolve("zh-CN") != "示例标题" || bundles[0].Meta.Description.Resolve("en") != "Literal description" {
		t.Fatal("localized wire union did not decode Host JSON")
	}
	raw, err := json.Marshal(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"description":"Literal description"`) {
		t.Fatalf("localized strings became base64: %s", raw)
	}
}

func (f fixtureCall) Call(ctx context.Context, method string, args, out any) error {
	return f(ctx, method, args, out)
}
func reply(out, value any) error {
	if out == nil {
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
func fixtureBundle() Bundle {
	return Bundle{Name: "fixture-plugin", Version: "1.0.0", Availability: "profile", Installed: true, Removable: true, Enabled: true, Meta: &Meta{Title: Text(`"Fixture plugin"`), Description: Text(`"A fixture plugin with configurable parts."`)}, Rows: []BundleRow{{RowID: "patch-id", ModuleName: "fixture-module", EntryID: "loader-42", Meta: &Meta{Title: Text(`"Fixture part"`)}}}}
}
func basicFixture(write func(string, any) error) fixtureCall {
	return func(_ context.Context, method string, args, out any) error {
		switch method {
		case "pluginInventory/list":
			return reply(out, map[string]any{"managementAvailable": true, "entries": []any{}})
		case "pluginManager/listBundles":
			return reply(out, []Bundle{fixtureBundle()})
		case "pluginManager/listPlugins":
			return reply(out, []Plugin{{EntryID: "loader-42", Enabled: true, Phase: "active"}})
		case "pluginManager/registries":
			return reply(out, RegistryInfo{})
		case "pluginManager/inspect":
			yes := true
			return reply(out, Inspection{Status: "accepted", Name: "fixture-plugin", Version: "2.0.0", Bundle: &yes})
		case "pluginManager/installBundle", "pluginManager/setBundleEnabled", "pluginManager/setPluginEnabled", "pluginManager/removeBundle", "pluginManager/setVersionExemption":
			if write != nil {
				if err := write(method, args); err != nil {
					return err
				}
			}
			return reply(out, Change{Application: "applied", Changed: true, Target: "fixture-plugin"})
		default:
			return fmt.Errorf("fixture has no method %s", method)
		}
	}
}
func waitState(t *testing.T, c *Controller, predicate func(State) bool) State {
	t.Helper()
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		snapshot := c.Snapshot()
		if predicate(snapshot) {
			return snapshot
		}
		select {
		case <-deadline.C:
			t.Fatalf("state did not settle: %+v", snapshot)
		case <-ticker.C:
		}
	}
}

func TestManagerJoinsLoaderIdentityAndKeepsCardsOnReadFailure(t *testing.T) {
	fail := false
	fixture := basicFixture(nil)
	c := New()
	c.Attach(fixtureCall(func(ctx context.Context, method string, args, out any) error {
		if fail && method == "pluginManager/listBundles" {
			return errors.New("bundle read failed")
		}
		return fixture(ctx, method, args, out)
	}), nil)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	state := c.Snapshot()
	if len(state.Packages) != 1 || state.Packages[0].Parts[0].Phase != "active" || !state.Packages[0].Parts[0].Enabled {
		t.Fatalf("Loader identity join failed: %+v", state.Packages)
	}
	fail = true
	if c.Load(context.Background()) == nil {
		t.Fatal("failed refresh reported success")
	}
	state = c.Snapshot()
	if state.Status != "error" || len(state.Packages) != 1 || state.Error != "bundle read failed" {
		t.Fatal("same-owner refresh erased its cards or diagnostic")
	}
}

func TestManagerCapturedControlsCannotMutateReplacementHost(t *testing.T) {
	var mu sync.Mutex
	writes := []string{}
	write := func(method string, _ any) error {
		mu.Lock()
		defer mu.Unlock()
		writes = append(writes, method)
		return nil
	}
	c := New()
	c.Attach(basicFixture(write), nil)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	old := c.Actions(c.Snapshot().Owner)
	old.Remove("fixture-plugin")
	if c.Snapshot().Confirmation == nil {
		t.Fatal("remove had no confirmation")
	}
	c.Attach(basicFixture(write), nil)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	old.Confirm()
	old.Enable("fixture-plugin", false)
	mu.Lock()
	n := len(writes)
	mu.Unlock()
	if n != 0 {
		t.Fatal("stale controls called the replacement Host")
	}
	current := c.Actions(c.Snapshot().Owner)
	current.EnableRow("loader-42", false)
	waitState(t, c, func(s State) bool { return !s.Busy && s.Notice == "applied" })
	mu.Lock()
	defer mu.Unlock()
	if len(writes) != 1 || writes[0] != "pluginManager/setPluginEnabled" {
		t.Fatalf("wrong entry action: %v", writes)
	}
}

func TestManagerUnavailableCapabilityNeverCallsMutationOrBundleEndpoints(t *testing.T) {
	calls := []string{}
	c := New()
	c.Attach(fixtureCall(func(_ context.Context, method string, _, out any) error {
		calls = append(calls, method)
		if method != "pluginInventory/list" {
			t.Fatalf("unmanaged profile called %s", method)
		}
		return reply(out, map[string]any{"managementAvailable": false, "entries": []any{}})
	}), nil)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.Actions(c.Snapshot().Owner).Enable("fixture-plugin", true)
	if c.Snapshot().Status != "unavailable" || len(calls) != 1 {
		t.Fatal("management capability was ignored")
	}
}

func TestManagerInstallSingleGestureUsesAcceptedRegistryAndDisabledSelection(t *testing.T) {
	writes := make(chan map[string]any, 2)
	c := New()
	c.Attach(basicFixture(func(method string, args any) error {
		if method == "pluginManager/installBundle" {
			raw, _ := json.Marshal(args)
			var v map[string]any
			_ = json.Unmarshal(raw, &v)
			writes <- v
		}
		return nil
	}), nil)
	t.Cleanup(c.Detach)
	if err := c.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	a := c.Actions(c.Snapshot().Owner)
	a.OpenInstall()
	a.EditInstall("fixture-plugin@2.0.0", "")
	a.Inspect()
	select {
	case v := <-writes:
		options := v["options"].(map[string]any)
		registry, present := options["registry"]
		if v["spec"] != "fixture-plugin@2.0.0" || options["requestId"] == "" || options["enabled"] != false || !present || registry != nil || options["approvedBuilds"] != nil {
			t.Fatalf("incorrect reviewed request: %v", v)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("one Install gesture did not inspect and install")
	}
	waitState(t, c, func(s State) bool { return !s.Busy && s.Install.Result != nil })
	if c.Snapshot().Confirmation != nil {
		t.Fatal("extra confirmation was inserted after Install")
	}
	select {
	case v := <-writes:
		t.Fatalf("duplicate install: %v", v)
	default:
	}
}

func TestManagerUnknownInstallCanReturnToInputWithoutInventedRefreshGate(t *testing.T) {
	for _, leave := range []string{"cancel", "close"} {
		t.Run(leave, func(t *testing.T) {
			base := basicFixture(nil)
			c := New()
			c.Attach(fixtureCall(func(ctx context.Context, method string, args, out any) error {
				if method == "pluginManager/waitForInstall" {
					return reply(out, nil)
				}
				return base(ctx, method, args, out)
			}), nil)
			t.Cleanup(c.Detach)
			a := c.Actions(c.Snapshot().Owner)
			a.edit(func(s *State) {
				s.Install = InstallState{Open: true, Spec: "retained-spec", RequestID: "uncertain-request", Phase: "unconfirmed"}
			})
			a.ReconcileInstall()
			waitState(t, c, func(s State) bool { return s.Install.Phase == "unknown" })
			if err := c.Load(context.Background()); err != nil {
				t.Fatal(err)
			}
			if c.Snapshot().Install.Phase != "unknown" {
				t.Fatal("inventory refresh rewrote the Host's unknown result")
			}
			if leave == "cancel" {
				a.CancelInstall()
			} else {
				a.CloseInstall()
			}
			s := c.Snapshot()
			if s.Install.Phase != "idle" || s.Install.RequestID != "" || s.Busy {
				t.Fatalf("retained unknown request was not released: %+v", s.Install)
			}
			if leave == "cancel" && s.Install.Spec != "retained-spec" {
				t.Fatal("return to input discarded the typed spec")
			}
		})
	}
}
func TestDiscoverMoreReloadsChangedDocumentWithoutMixingPages(t *testing.T) {
	c := New()
	base := basicFixture(nil)
	requests := make(chan Query, 4)
	c.Attach(fixtureCall(func(ctx context.Context, method string, args, out any) error {
		if method == "pluginMarket/listIndexes" {
			return reply(out, []Index{{URL: "https://fixture.invalid/index.json"}})
		}
		if method != "pluginMarket/search" {
			return base(ctx, method, args, out)
		}
		raw, _ := json.Marshal(args)
		var request struct {
			URL   string `json:"url"`
			Query Query  `json:"query"`
		}
		_ = json.Unmarshal(raw, &request)
		requests <- request.Query
		version, prefix, count := int64(10), "old", 50
		if request.Query.Offset > 0 || request.Query.Limit == 100 {
			version, prefix = 11, "new"
		}
		if request.Query.Limit == 100 {
			count = 100
		}
		entries := make([]MarketEntry, count)
		for n := range entries {
			entries[n].Name = fmt.Sprintf("%s-%03d", prefix, request.Query.Offset+n)
		}
		result := SearchResult{URL: request.URL, FetchedAt: version, Entries: entries, Total: 100}
		if request.Query.Limit != 100 {
			next := request.Query.Offset + 50
			result.NextOffset = &next
		}
		return reply(out, result)
	}), nil)
	defer c.Detach()
	if err := c.LoadIndexes(context.Background(), c.Snapshot().Owner); err != nil {
		t.Fatal(err)
	}
	nextQuery := func() Query {
		t.Helper()
		select {
		case q := <-requests:
			return q
		case <-time.After(3 * time.Second):
			t.Fatal("expected Discover page request did not arrive")
			return Query{}
		}
	}
	nextQuery()
	c.Actions(c.Snapshot().Owner).More()
	s := waitState(t, c, func(s State) bool {
		return !s.Discover.Loading && !s.Discover.MoreLoading && s.Discover.Listing != nil && s.Discover.Listing.FetchedAt == 11
	})
	more, reload := nextQuery(), nextQuery()
	if more.Offset != 50 || more.FetchedAt != 10 || more.Limit != 50 || reload.Offset != 0 || reload.FetchedAt != 0 || reload.Limit != 100 {
		t.Fatalf("incorrect pinned More/reload: %+v %+v", more, reload)
	}
	if s.Discover.Error != "" || len(s.Discover.Listing.Entries) != 100 {
		t.Fatal(s.Discover.Error)
	}
	for _, entry := range s.Discover.Listing.Entries {
		if !strings.HasPrefix(entry.Name, "new-") {
			t.Fatal("mixed index documents", entry.Name)
		}
	}
}
func TestDiscoverMissingEndpointStaysDistinctFromCuratedMarket(t *testing.T) {
	c := New()
	c.Attach(fixtureCall(func(_ context.Context, method string, _, _ any) error {
		return &dsh.Error{Code: "method-not-found", Message: method + " is unavailable"}
	}), nil)
	err := c.LoadIndexes(context.Background(), c.Snapshot().Owner)
	state := c.Snapshot()
	if err == nil || state.Discover.Status != "error" || !strings.Contains(state.Discover.IndexesError, "pluginMarket/listIndexes") || state.Discover.Listing != nil {
		t.Fatal("missing generic market endpoint was hidden by fallback data")
	}
}

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestGenericMarketTypertEnvelopeWithoutNetwork(t *testing.T) {
	calls := 0
	transport := fixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var envelope struct {
			Type    string `json:"type"`
			ID      string `json:"rpcId"`
			Method  string `json:"method"`
			Payload struct {
				Args map[string]json.RawMessage `json:"args"`
			} `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Type != "client-request" || envelope.ID == "" || envelope.Method != "pluginMarket/search" || r.URL.Path != "/api/pluginMarket/search" {
			t.Fatalf("incorrect Typert envelope: %+v", envelope)
		}
		var url string
		_ = json.Unmarshal(envelope.Payload.Args["url"], &url)
		var q Query
		_ = json.Unmarshal(envelope.Payload.Args["query"], &q)
		if url != "https://fixture.invalid/index.json" || q.Offset != 50 || q.Limit != 50 || q.Sort.Kind != "index" {
			t.Fatal("named arguments lost their contract")
		}
		body, _ := json.Marshal(map[string]any{"type": "server-response", "rpcId": envelope.ID, "result": map[string]any{"ok": true, "value": SearchResult{URL: url, FetchedAt: 12, Total: 0}}})
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
	})
	client := Client{&dsh.Client{Endpoint: dsh.Endpoint{Origin: "https://host.fixture.invalid"}, HTTP: &http.Client{Transport: transport}}}
	result, err := client.Search(context.Background(), "https://fixture.invalid/index.json", Query{Sort: MarketSort{Kind: "index"}, Offset: 50, Limit: 50})
	if err != nil || calls != 1 || result.FetchedAt != 12 {
		t.Fatalf("result=%+v error=%v calls=%d", result, err, calls)
	}
}
