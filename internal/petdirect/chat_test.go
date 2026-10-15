package petdirect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeTransport func(*http.Request) (*http.Response, error)

func (f fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func response(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}
}
func config() Config {
	return Config{APIKey: "fixture-key", BaseURL: "https://api.example.invalid/v1/", Prompt: Prompt("genki", Persona{Name: "小鲸", UserTitle: "朋友", PersonaText: "喜欢音乐"})}
}
func fixtureClient(load func() (Config, error), transport fakeTransport) *Client {
	return &Client{Load: load, HTTP: &http.Client{Transport: transport}}
}

func TestExplicitNativeRequestContract(t *testing.T) {
	reads, calls := 0, 0
	c := fixtureClient(func() (Config, error) { reads++; return config(), nil }, func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "POST" || r.URL.String() != "https://api.example.invalid/v1/chat/completions" {
			t.Fatal("wrong original direct endpoint")
		}
		if r.Header.Get("Authorization") != "Bearer fixture-key" || r.Header.Get("Cookie") != "" {
			t.Fatal("wrong credential boundary")
		}
		if _, ok := r.Context().Deadline(); !ok {
			t.Fatal("missing deadline")
		}
		var b struct {
			Model    string    `json:"model"`
			Messages []Message `json:"messages"`
			Tokens   int       `json:"max_tokens"`
			Stream   bool      `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			t.Fatal(err)
		}
		if b.Model != Model || b.Tokens != 360 || b.Stream || len(b.Messages) != 4 || b.Messages[1].Content != "先前" || b.Messages[3].Content != "你好" {
			t.Fatalf("wire mismatch: %#v", b)
		}
		if !strings.Contains(b.Messages[0].Content, "你是小鲸") || !strings.Contains(b.Messages[0].Content, "本鲸") {
			t.Fatal("persona missing")
		}
		return response(200, `{"choices":[{"message":{"role":"assistant","content":"  好呀  "}}]}`), nil
	})
	if reads != 0 || calls != 0 {
		t.Fatal("opening made a request")
	}
	reply, err := c.Send(context.Background(), " 你好 ", []Message{{"user", "先前"}, {"assistant", "收到"}})
	if err != nil || reply != "好呀" || reads != 1 || calls != 1 {
		t.Fatalf("explicit send: %q %v %d %d", reply, err, reads, calls)
	}
}

func TestEndpointAndInputBoundaries(t *testing.T) {
	for _, base := range []string{"http://remote.example", "http://127.0.0.2", "https://u:p@example.test", "https://example.test?a=b", "https://example.test#frag", "file:///tmp/keys", "https:///oops"} {
		if _, err := endpoint(base); err != ErrEndpoint {
			t.Fatalf("accepted %q", base)
		}
	}
	for _, base := range []string{"", "https://example.test/v1", "http://127.0.0.1:9999/v1", "http://[::1]:9999", "http://localhost:9999"} {
		if _, err := endpoint(base); err != nil {
			t.Fatalf("rejected %q", base)
		}
	}
	reads, calls := 0, 0
	c := fixtureClient(func() (Config, error) { reads++; return Config{}, nil }, func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("must not send") })
	for _, text := range []string{"  ", strings.Repeat("😀", 1001)} {
		if _, err := c.Send(context.Background(), text, nil); err != ErrInput {
			t.Fatal("input boundary")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Send(ctx, "hello", nil); err != ErrRequest {
		t.Fatal("cancelled request")
	}
	if reads != 0 || calls != 0 {
		t.Fatal("invalid/cancelled input loaded credentials")
	}
	if _, err := c.Send(context.Background(), "hello", nil); err != ErrCredentials || calls != 0 {
		t.Fatal("missing credentials sent")
	}
}

func TestNeverRetriesOrLeaksProviderErrors(t *testing.T) {
	for _, test := range []struct {
		name         string
		status       int
		body         string
		transportErr error
		want         error
	}{
		{"network", 0, "", errors.New("fixture-key in https://sensitive.example"), ErrRequest},
		{"http", 500, "fixture-key server stack", nil, ErrRequest},
		{"malformed", 200, "fixture-key not JSON", nil, ErrResponse},
		{"empty", 200, `{"choices":[]}`, nil, ErrResponse},
		{"oversized", 200, strings.Repeat("x", (1<<20)+1), nil, ErrResponse},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			c := fixtureClient(func() (Config, error) { return config(), nil }, func(*http.Request) (*http.Response, error) {
				calls++
				if test.transportErr != nil {
					return nil, test.transportErr
				}
				return response(test.status, test.body), nil
			})
			reply, err := c.Send(context.Background(), "hello", nil)
			if err != test.want || reply != "" || calls != 1 || strings.Contains(err.Error(), "fixture-key") {
				t.Fatalf("unsafe failure: %q %v calls=%d", reply, err, calls)
			}
		})
	}
	calls := 0
	c := fixtureClient(func() (Config, error) { return config(), nil }, func(*http.Request) (*http.Response, error) {
		calls++
		r := response(302, "")
		r.Header.Set("Location", "https://other.example.invalid/steal")
		return r, nil
	})
	if _, err := c.Send(context.Background(), "hello", nil); err != ErrRequest || calls != 1 {
		t.Fatal("redirect followed")
	}
}

func TestChangedProviderCredentialOrPersonaDoesNotReceiveMemory(t *testing.T) {
	for _, change := range []func(*Config){func(c *Config) { c.BaseURL = "https://another.invalid" }, func(c *Config) { c.APIKey = "other" }, func(c *Config) { c.Prompt = "changed" }} {
		cfg, calls := config(), 0
		c := fixtureClient(func() (Config, error) { return cfg, nil }, func(*http.Request) (*http.Response, error) {
			calls++
			return response(200, `{"choices":[{"message":{"content":"ok"}}]}`), nil
		})
		if _, err := c.Send(context.Background(), "one", nil); err != nil {
			t.Fatal(err)
		}
		change(&cfg)
		if _, err := c.Send(context.Background(), "two", []Message{{"user", "one"}, {"assistant", "ok"}}); err != ErrChanged || calls != 1 {
			t.Fatal("old memory crossed configuration change")
		}
	}
}

func TestSixTurnMemoryCancelCloseAndDraftIsolation(t *testing.T) {
	m := Memory{}
	var oldest Ticket
	for i := 0; i < 8; i++ {
		ticket, ok := m.Begin(fmt.Sprint(i))
		if !ok {
			t.Fatal("begin")
		}
		if i == 0 {
			oldest = ticket
		}
		if _, ok := m.Begin("duplicate click"); ok {
			t.Fatal("concurrent send")
		}
		if len(ticket.History) > 12 {
			t.Fatal("history limit")
		}
		if !m.Finish(ticket, "reply") {
			t.Fatal("finish")
		}
	}
	ticket, ok := m.Begin("cancel me")
	if !ok {
		t.Fatal("begin cancel")
	}
	if len(ticket.History) != 12 || ticket.History[0].Content != "2" {
		t.Fatal("six-turn rolling window")
	}
	ticket.History[0].Content = "mutated copy"
	m.Cancel()
	if m.Finish(ticket, "late") || m.Finish(oldest, "older") {
		t.Fatal("late callback accepted")
	}
	next, _ := m.Begin("new draft")
	if next.History[0].Content != "2" {
		t.Fatal("history alias")
	}
	m.Reset()
	if m.Finish(next, "after close") {
		t.Fatal("closed callback accepted")
	}
	reopened, _ := m.Begin("reopen")
	if len(reopened.History) != 0 {
		t.Fatal("close persisted memory")
	}
	if RetainDraft(false, true, "uncertain Host send") != "" || RetainDraft(true, false, "API draft") != "" || RetainDraft(true, true, "replaced API window") != "" || RetainDraft(false, false, "same mode") != "same mode" {
		t.Fatal("draft crossed mode")
	}
}

func TestInFlightCancellationAndUnicodeReplyLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	c := fixtureClient(func() (Config, error) { return config(), nil }, func(r *http.Request) (*http.Response, error) {
		calls++
		cancel()
		return response(200, `{"choices":[{"message":{"content":"late"}}]}`), nil
	})
	if _, err := c.Send(ctx, "hello", nil); err != ErrRequest || calls != 1 {
		t.Fatal("cancelled completion accepted")
	}
	long := strings.Repeat("😀", 121)
	got := truncate(long, ReplyLimit)
	if got != strings.Repeat("😀", 120) {
		t.Fatal("split surrogate/reply cap")
	}
}

func TestPersonaUsesOriginalSettingsAndExplicitPetOverride(t *testing.T) {
	settings := Persona{Name: "小鲸", UserTitle: "朋友", PersonaText: "喜欢音乐", Personality: "poison"}
	got := Prompt("", settings)
	for _, want := range []string{"你是小鲸", "毒舌", "你称呼用户为「朋友」。", "用户给你的额外人设：喜欢音乐"} {
		if !strings.Contains(got, want) {
			t.Fatal(want)
		}
	}
	if strings.Contains(Prompt("natural", settings), "毒舌") {
		t.Fatal("explicit personality lost")
	}
}
