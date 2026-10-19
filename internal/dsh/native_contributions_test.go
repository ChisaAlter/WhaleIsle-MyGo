package dsh

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// These are declaration wires, not executable plugins. The graph deliberately
// points at undeliverable bundles: the native watcher must only read SSE.
type nativeTestEntry struct {
	ID       string   `json:"id"`
	URL      string   `json:"url"`
	Rev      string   `json:"rev"`
	Inject   []string `json:"inject,omitempty"`
	External []string `json:"external,omitempty"`
}

type nativeTestGraph struct {
	Rev     string            `json:"rev"`
	Release map[string]string `json:"release"`
	Entries []nativeTestEntry `json:"entries"`
	Batches []nativeTestBatch `json:"batches"`
}

type nativeTestBatch struct {
	Phase   string   `json:"phase"`
	URL     string   `json:"url"`
	Rev     string   `json:"rev"`
	Entries []string `json:"entries"`
}

func nativeTestFullGraph() nativeTestGraph {
	ids := []string{
		nativeConversation, nativeCommands, nativeModelSelection, nativePermissions, nativeFeedback, nativeLabel1017,
		"@deepseek-ai/dsh-client-connection", "@deepseek-ai/dsh-api-session-controller",
		"@deepseek-ai/dsh-client-file-upload", "@deepseek-ai/dsh-client-locale",
		"@deepseek-ai/dsh-client-ui-layout", "@deepseek-ai/dsh-client-ui-renderer",
		"@deepseek-ai/dsh-client-ui-session", "@deepseek-ai/dsh-client-ui-settings",
		"@deepseek-ai/dsh-client-ui-workspace", "@deepseek-ai/dsh-api-remotes",
		"@deepseek-ai/dsh-client-ui-input-trigger", "test-inject", "test-external", "test-leaf",
	}
	g := nativeTestGraph{Rev: "graph-one", Release: map[string]string{"version": "test-release", "commit": "test-commit"}}
	for _, id := range ids {
		g.Entries = append(g.Entries, nativeTestEntry{ID: id, URL: "https://bundles.invalid/not-fetched", Rev: "one"})
	}
	g.entry(nativeModelSelection).Inject = []string{"test-inject/client"}
	g.entry(nativeModelSelection).External = []string{"test-external/client", "react"}
	g.entry("test-inject").Inject = []string{"test-leaf/client"}
	g.entry("test-leaf").External = []string{"test-inject/client"} // A bounded dependency cycle.
	g.rebatch()
	return g
}

func (g *nativeTestGraph) entry(id string) *nativeTestEntry {
	for i := range g.Entries {
		if g.Entries[i].ID == id {
			return &g.Entries[i]
		}
	}
	panic("missing test entry: " + id)
}

func (g *nativeTestGraph) rebatch() {
	ids := make([]string, 0, len(g.Entries))
	for _, entry := range g.Entries {
		ids = append(ids, entry.ID)
	}
	g.Batches = []nativeTestBatch{{Phase: "application", URL: "/opaque-combo.js", Rev: g.Rev, Entries: ids}}
}

func nativeTestJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func nativeTestGraphEvent(g nativeTestGraph) string {
	return "data: " + nativeTestJSON(map[string]any{"type": "graph", "graph": g}) + "\n\n"
}

func nativeTestRebuilt(id, rev string) string {
	return "data: " + nativeTestJSON(map[string]string{"type": "rebuilt", "id": id, "rev": rev}) + "\n\n"
}

type nativeTestWatch struct {
	updates chan NativeContributions
	result  chan error
	stopped chan struct{}
	cancel  context.CancelFunc
}

func nativeTestStartWatch(t *testing.T, client *Client) *nativeTestWatch {
	t.Helper()
	// Keep even a regressed implementation inside this test's single origin.
	// Bundle URLs and accidental new network requests cannot leave the test.
	origin, err := url.Parse(client.Endpoint.Origin)
	if err != nil {
		t.Fatal(err)
	}
	localClient := *client
	httpClient := *client.http()
	transport := httpClient.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	httpClient.Transport = nativeTestRoundTrip(func(request *http.Request) (*http.Response, error) {
		if request.URL.Scheme != origin.Scheme || request.URL.Host != origin.Host || request.URL.Path != "/plugins/events" {
			t.Error("watch attempted a request outside its single SSE endpoint")
			return nil, errors.New("test refused non-SSE request")
		}
		return transport.RoundTrip(request)
	})
	localClient.HTTP = &httpClient
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	w := &nativeTestWatch{updates: make(chan NativeContributions, 32), result: make(chan error, 1), stopped: make(chan struct{}), cancel: cancel}
	go func() {
		defer close(w.stopped)
		w.result <- localClient.WatchNativeContributions(ctx, func(value NativeContributions) {
			select {
			case w.updates <- value:
			case <-ctx.Done():
			}
		})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-w.stopped:
		case <-time.After(2 * time.Second):
			t.Error("native contribution watch did not cancel")
		}
	})
	return w
}

func (w *nativeTestWatch) next(t *testing.T) NativeContributions {
	t.Helper()
	select {
	case value := <-w.updates:
		return value
	case <-time.After(6 * time.Second):
		t.Fatal("timed out waiting for native contribution publication")
		return NativeContributions{}
	}
}

func (w *nativeTestWatch) finish(t *testing.T) error {
	t.Helper()
	select {
	case err := <-w.result:
		return err
	case <-time.After(2 * time.Second):
		t.Fatal("native contribution watch did not finish")
		return nil
	}
}

// A channel drives real HTTP streaming without sleeps or a real Host process.
func nativeTestStream(t *testing.T) (*httptest.Server, chan<- string, <-chan http.Header) {
	t.Helper()
	frames := make(chan string, 32)
	requests := make(chan http.Header, 4)
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/plugins/events" || request.URL.RawQuery != "" {
			t.Errorf("unexpected native request: %s %s", request.Method, request.URL.Path)
			response.WriteHeader(http.StatusNotFound)
			return
		}
		requests <- request.Header.Clone()
		response.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		response.(http.Flusher).Flush()
		for {
			select {
			case frame := <-frames:
				if _, err := io.WriteString(response, frame); err != nil {
					return
				}
				response.(http.Flusher).Flush()
			case <-request.Context().Done():
				return
			}
		}
	}))
	t.Cleanup(server.Close)
	return server, frames, requests
}

func nativeTestAllPorts(t *testing.T, value NativeContributions) {
	t.Helper()
	if !value.Known || !value.FileCommand || !value.ModelCommand || !value.PermissionCommand || !value.FeedbackCommand || !value.ModelLabel1017 || value.Error != "" {
		t.Fatalf("expected five declared native adapters: %+v", value)
	}
	if value.Revision == "" || value.ReleaseVersion != "test-release" || value.ReleaseCommit != "test-commit" {
		t.Fatalf("missing graph/release identity: %+v", value)
	}
	for _, port := range []string{"file", "model", "permission", "feedback"} {
		if value.CommandRevision(port) == "" {
			t.Fatalf("missing %s owner identity", port)
		}
	}
	if value.CommandRevision("unknown") != "" {
		t.Fatal("unknown port acquired an identity")
	}
}

func TestNativeContributionsAuthenticatedGraphLifecycle(t *testing.T) {
	server, frames, requests := nativeTestStream(t)
	w := nativeTestStartWatch(t, &Client{Endpoint: Endpoint{Origin: server.URL, Cookie: "dsh-auth-test=private-marker"}})
	select {
	case headers := <-requests:
		if headers.Get("Cookie") != "dsh-auth-test=private-marker" || headers.Get("Accept") != "text/event-stream" {
			t.Fatalf("missing authenticated SSE headers: %v", headers)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not request SSE")
	}
	g := nativeTestFullGraph()
	// BOM/comments, mixed line endings, ignored named events, and a multiline
	// message exercise EventSource framing before granting any authority.
	frames <- "\uFEFF: hello\r\nevent: diagnostic\rdata: not-json\r\r"
	frames <- nativeTestRebuilt(nativeModelSelection, "not-an-initial-graph")
	// Build the two data lines explicitly; blank lines must stay blank.
	data := nativeTestJSON(map[string]any{"type": "graph", "graph": g})
	frames <- "event: message\r\ndata: {\r\ndata: " + data[1:] + "\r\n\r\n"
	initial := w.next(t)
	nativeTestAllPorts(t, initial)
	frames <- nativeTestRebuilt("unknown-owner", "two")
	frames <- nativeTestRebuilt(nativeModelSelection, "one")
	frames <- nativeTestRebuilt(nativeModelSelection, "two")
	rebuilt := w.next(t)
	if rebuilt.CommandRevision("model") == initial.CommandRevision("model") || rebuilt.CommandRevision("file") != initial.CommandRevision("file") {
		t.Fatal("model rebuild did not fence only its own command")
	}
	for i, entry := range g.Entries {
		if entry.ID == nativeModelSelection {
			g.Entries = append(g.Entries[:i], g.Entries[i+1:]...)
			break
		}
	}
	g.Entries = append(g.Entries, nativeTestEntry{ID: nativeModelSelection + "-lookalike", URL: "/unused.js", Rev: "one"})
	g.Rev = "model-removed"
	g.rebatch()
	frames <- nativeTestGraphEvent(g)
	removed := w.next(t)
	if !removed.Known || removed.ModelCommand || removed.ModelLabel1017 || removed.CommandRevision("model") != "" || !removed.FileCommand || !removed.PermissionCommand || !removed.FeedbackCommand {
		t.Fatalf("owner removal did not withdraw its adapters: %+v", removed)
	}
	g = nativeTestFullGraph()
	g.Rev = "missing-permission-injection"
	g.entry(nativePermissions).Inject = []string{"missing-owner/client"}
	frames <- nativeTestGraphEvent(g)
	missing := w.next(t)
	if !missing.Known || missing.PermissionCommand || missing.CommandRevision("permission") != "" || !missing.ModelCommand || !missing.FileCommand || !missing.FeedbackCommand || !missing.ModelLabel1017 {
		t.Fatalf("missing inject dependency did not withdraw permission only: %+v", missing)
	}
	w.cancel()
	if err := w.finish(t); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
}

func TestNativeContributionsCommandRevisionScope(t *testing.T) {
	server, frames, _ := nativeTestStream(t)
	w := nativeTestStartWatch(t, &Client{Endpoint: Endpoint{Origin: server.URL}})
	frames <- nativeTestGraphEvent(nativeTestFullGraph())
	baseline := w.next(t)
	nativeTestAllPorts(t, baseline)
	cases := []struct {
		name    string
		change  func(*nativeTestGraph)
		changed string
	}{
		{"delivery-and-order", func(g *nativeTestGraph) {
			for left, right := 0, len(g.Entries)-1; left < right; left, right = left+1, right-1 {
				g.Entries[left], g.Entries[right] = g.Entries[right], g.Entries[left]
			}
			for i := range g.Entries {
				g.Entries[i].URL = "/different-delivery.js"
			}
			g.rebatch()
			g.Batches[0].URL = "/another-batch.js"
		}, ""},
		{"unrelated-entry", func(g *nativeTestGraph) {
			g.Entries = append(g.Entries, nativeTestEntry{ID: "unrelated", URL: "/unused.js", Rev: "two"})
			g.rebatch()
		}, ""},
		{"model-owner", func(g *nativeTestGraph) { g.entry(nativeModelSelection).Rev = "two" }, "model"},
		{"transitive-inject", func(g *nativeTestGraph) { g.entry("test-leaf").Rev = "two" }, "model"},
		{"external", func(g *nativeTestGraph) { g.entry("test-external").Rev = "two" }, "model"},
		{"permission-owner", func(g *nativeTestGraph) { g.entry(nativePermissions).Rev = "two" }, "permission"},
		{"feedback-owner", func(g *nativeTestGraph) { g.entry(nativeFeedback).Rev = "two" }, "feedback"},
		{"shared-base", func(g *nativeTestGraph) { g.entry("@deepseek-ai/dsh-client-locale").Rev = "two" }, "file model permission feedback"},
		{"label-only", func(g *nativeTestGraph) { g.entry(nativeLabel1017).Rev = "two" }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := nativeTestFullGraph()
			g.Rev = tc.name
			tc.change(&g)
			frames <- nativeTestGraphEvent(g)
			value := w.next(t)
			nativeTestAllPorts(t, value)
			if value.Revision == baseline.Revision {
				t.Fatal("diagnostic graph identity did not change")
			}
			for _, port := range []string{"file", "model", "permission", "feedback"} {
				wantChanged := strings.Contains(" "+tc.changed+" ", " "+port+" ")
				if gotChanged := value.CommandRevision(port) != baseline.CommandRevision(port); gotChanged != wantChanged {
					t.Errorf("%s identity changed = %v, want %v", port, gotChanged, wantChanged)
				}
			}
		})
	}
	g := nativeTestFullGraph()
	g.Entries = append(g.Entries, nativeTestEntry{ID: "unrelated", URL: "/unused.js", Rev: "one"})
	g.Rev = "before-unrelated-rebuild"
	g.rebatch()
	frames <- nativeTestGraphEvent(g)
	before := w.next(t)
	frames <- nativeTestRebuilt("unrelated", "two")
	after := w.next(t)
	if after.Revision == before.Revision {
		t.Fatal("unrelated rebuild did not update graph diagnostics")
	}
	for _, port := range []string{"file", "model", "permission", "feedback"} {
		if after.CommandRevision(port) != before.CommandRevision(port) {
			t.Errorf("unrelated rebuild fenced %s", port)
		}
	}
}

func TestNativeContributionsInvalidFramesRecover(t *testing.T) {
	server, frames, _ := nativeTestStream(t)
	w := nativeTestStartWatch(t, &Client{Endpoint: Endpoint{Origin: server.URL}})
	frames <- nativeTestGraphEvent(nativeTestFullGraph())
	baseline := w.next(t)
	nativeTestAllPorts(t, baseline)
	frames <- "data: {\"type\":\"future-protocol\",\"graph\":null}\n\n"
	g := nativeTestFullGraph()
	g.Rev = "after-unknown"
	frames <- nativeTestGraphEvent(g)
	nativeTestAllPorts(t, w.next(t)) // Unknown frames must not publish a withdrawal.
	for i, invalid := range []string{
		`{"secret":"private-marker","url":"https://private.invalid/",`,
		`{"type":"graph","graph":{"rev":"bad","entries":[],"batches":[{"phase":"application","url":"/bad","rev":"bad","entries":["missing"]}]}}`,
		`{"type":"rebuilt","id":12,"rev":"two"}`,
	} {
		frames <- "data: " + invalid + "\n\n"
		value := w.next(t)
		if value.Known || value.Error == "" || strings.Contains(value.Error, "private") || value.CommandRevision("model") != "" {
			t.Fatalf("invalid frame retained authority or echoed data: %+v", value)
		}
		frames <- nativeTestRebuilt(nativeModelSelection, "ignored-until-valid-graph")
		g.Rev = fmt.Sprintf("recovered-%d", i)
		frames <- nativeTestGraphEvent(g)
		recovered := w.next(t)
		nativeTestAllPorts(t, recovered)
		if recovered.CommandRevision("model") != baseline.CommandRevision("model") {
			t.Fatal("rebuilt while authority was withdrawn changed the recovered model")
		}
	}
}

func TestNativeContributionsEOFReconnectAndReleaseGate(t *testing.T) {
	for _, changedRelease := range []bool{false, true} {
		t.Run(fmt.Sprintf("release-changed-%v", changedRelease), func(t *testing.T) {
			var calls, active, maximum atomic.Int32
			closed := make(chan time.Time, 1)
			arrival := make(chan time.Time, 1)
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				nowActive := active.Add(1)
				defer active.Add(-1)
				for previous := maximum.Load(); nowActive > previous; previous = maximum.Load() {
					if maximum.CompareAndSwap(previous, nowActive) {
						break
					}
				}
				call := calls.Add(1)
				response.Header().Set("Content-Type", "text/event-stream")
				g := nativeTestFullGraph()
				if call == 1 {
					io.WriteString(response, nativeTestGraphEvent(g))
					// An incomplete final message must never acquire authority.
					io.WriteString(response, `data: {"type":"rebuilt","id":"`+nativeModelSelection+`","rev":"partial"}`)
					response.(http.Flusher).Flush()
					closed <- time.Now()
					return
				}
				arrival <- time.Now()
				g.Rev = "reconnected"
				if changedRelease {
					g.Release["commit"] = "another-commit"
				}
				io.WriteString(response, nativeTestGraphEvent(g))
				response.(http.Flusher).Flush()
				<-request.Context().Done()
			}))
			t.Cleanup(server.Close)
			w := nativeTestStartWatch(t, &Client{Endpoint: Endpoint{Origin: server.URL}})
			initial := w.next(t)
			nativeTestAllPorts(t, initial)
			disconnected := w.next(t)
			if disconnected.Known || disconnected.Error == "" {
				t.Fatalf("EOF did not revoke declarations: %+v", disconnected)
			}
			value := w.next(t)
			select {
			case second := <-arrival:
				if elapsed := second.Sub(<-closed); elapsed < 3*time.Second {
					t.Fatalf("reconnect delay = %v, want at least EventSource's 3 seconds", elapsed)
				}
			default:
				t.Fatal("publication preceded the second connection")
			}
			if changedRelease {
				if value.Known || value.Error != errNativeContributionRelease.Error() || !errors.Is(w.finish(t), errNativeContributionRelease) {
					t.Fatalf("new release admitted after reconnect: %+v", value)
				}
			} else {
				nativeTestAllPorts(t, value)
				if value.CommandRevision("model") != initial.CommandRevision("model") {
					t.Fatal("unterminated final event changed the model revision")
				}
				w.cancel()
				if err := w.finish(t); !errors.Is(err, context.Canceled) {
					t.Fatalf("cancel while reading = %v", err)
				}
			}
			if calls.Load() != 2 || maximum.Load() != 1 {
				t.Fatalf("requests = %d, concurrent streams = %d", calls.Load(), maximum.Load())
			}
		})
	}
}

type nativeTestRoundTrip func(*http.Request) (*http.Response, error)

func (f nativeTestRoundTrip) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestNativeContributionsTransportFailureCancelsReconnect(t *testing.T) {
	var calls atomic.Int32
	client := &Client{
		Endpoint: Endpoint{Origin: "http://127.0.0.1:1", Cookie: "private-marker"},
		HTTP: &http.Client{Transport: nativeTestRoundTrip(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return nil, errors.New("private-marker https://private.invalid/")
		})},
	}
	w := nativeTestStartWatch(t, client)
	value := w.next(t)
	if value.Known || value.Error != "native contribution connection failed" {
		t.Fatalf("transport failure not sanitized: %+v", value)
	}
	w.cancel()
	if err := w.finish(t); !errors.Is(err, context.Canceled) || calls.Load() != 1 {
		t.Fatalf("cancel during reconnect: err=%v, requests=%d", err, calls.Load())
	}
}

func TestNativeContributionsTerminalHTTPResponses(t *testing.T) {
	for _, tc := range []struct {
		name        string
		status      int
		contentType string
	}{
		{"unauthorized", 401, "text/event-stream"},
		{"forbidden", 403, "text/event-stream"},
		{"no-content", 204, "text/event-stream"},
		{"server-refusal", 500, "text/event-stream"},
		{"not-sse", 200, "application/json"},
		{"missing-type", 200, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				if request.Header.Get("Cookie") != "" {
					t.Error("empty Endpoint cookie was replaced")
				}
				response.Header().Set("Content-Type", tc.contentType)
				response.WriteHeader(tc.status)
				io.WriteString(response, "private-marker https://private.invalid/")
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var values []NativeContributions
			err := (&Client{Endpoint: Endpoint{Origin: server.URL}}).WatchNativeContributions(ctx, func(value NativeContributions) { values = append(values, value) })
			if err == nil || errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "private") || calls.Load() != 1 || len(values) != 1 || values[0].Known || values[0].Error != err.Error() {
				t.Fatalf("response was not a sanitized terminal refusal: err=%v calls=%d values=%+v", err, calls.Load(), values)
			}
		})
	}
}

func TestNativeContributionsRedirectCannotSendForeignCookie(t *testing.T) {
	for _, rewrite := range []bool{false, true} {
		t.Run(fmt.Sprintf("callback-rewrites-%v", rewrite), func(t *testing.T) {
			var foreignCalls atomic.Int32
			foreign := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				foreignCalls.Add(1)
				response.WriteHeader(http.StatusNoContent)
			}))
			defer foreign.Close()
			source := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Header.Get("Cookie") != "dsh-auth-test=private-marker" {
					t.Error("origin did not receive Endpoint cookie")
				}
				location := foreign.URL + "/private-marker"
				if rewrite {
					location = "http://" + request.Host + "/same-origin"
				}
				http.Redirect(response, request, location, http.StatusFound)
			}))
			defer source.Close()
			client := &Client{Endpoint: Endpoint{Origin: source.URL, Cookie: "dsh-auth-test=private-marker"}, HTTP: &http.Client{}}
			if rewrite {
				client.HTTP.CheckRedirect = func(request *http.Request, _ []*http.Request) error {
					request.URL.Host = strings.TrimPrefix(foreign.URL, "http://")
					return nil
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var values []NativeContributions
			err := client.WatchNativeContributions(ctx, func(value NativeContributions) { values = append(values, value) })
			if err == nil || err.Error() != "native contribution redirect refused" || foreignCalls.Load() != 0 || len(values) != 1 || values[0].Known || values[0].Error != err.Error() {
				t.Fatalf("foreign origin was not fenced: err=%v foreignRequests=%d values=%+v", err, foreignCalls.Load(), values)
			}
		})
	}
}

func TestNativeContributionsOriginAdmissionBeforeDial(t *testing.T) {
	for _, origin := range []string{
		"https://remote.invalid", "http://192.0.2.1", "http://[2001:db8::1]",
		"http://private-marker@127.0.0.1", "http://127.0.0.1?private-marker", "http://127.0.0.1/not-origin",
	} {
		t.Run(origin, func(t *testing.T) {
			var calls int
			client := &Client{Endpoint: Endpoint{Origin: origin}, HTTP: &http.Client{Transport: nativeTestRoundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				return nil, errors.New("unexpected dial")
			})}}
			var values []NativeContributions
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			err := client.WatchNativeContributions(ctx, func(value NativeContributions) { values = append(values, value) })
			if err == nil || strings.Contains(err.Error(), "private-marker") || calls != 0 || len(values) != 1 || values[0].Known {
				t.Fatalf("inadmissible origin reached transport: err=%v calls=%d values=%+v", err, calls, values)
			}
		})
	}
	for _, origin := range []string{"http://localhost:1", "http://LOCALHOST:1", "http://127.42.1.2:1", "http://[::1]:1"} {
		t.Run(origin, func(t *testing.T) {
			var calls int
			client := &Client{Endpoint: Endpoint{Origin: origin}, HTTP: &http.Client{Transport: nativeTestRoundTrip(func(request *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: request}, nil
			})}}
			err := client.WatchNativeContributions(context.Background(), func(NativeContributions) {})
			if calls != 1 || err == nil || err.Error() != "Host refused the native contribution stream" {
				t.Fatalf("loopback spelling rejected before HTTP: calls=%d err=%v", calls, err)
			}
		})
	}
}
