package pluginrpc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type memoryTransport func(*http.Request) (*http.Response, error)

func (f memoryTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	response, err := f(r)
	return fixtureResponseIdentity(r, response, err)
}
func TestBotMemoryWireAndReceipt(t *testing.T) {
	calls := 0
	c := Client{BaseURL: "http://127.0.0.1", HTTPClient: &http.Client{Transport: memoryTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		var req clientRequest
		json.NewDecoder(r.Body).Decode(&req)
		p := req.Payload.(map[string]any)
		if p["id"] != "b" {
			t.Fatal("foreign Bot")
		}
		response := `{"botId":"b","bot":{"entries":["note"],"text":"- note","revision":"rev-b"},"user":{"entries":[],"text":"","revision":"rev-u"},"limits":{"bot":8000,"user":4000}}`
		switch req.Method {
		case "memory/get":
			if r.URL.Path != "/dshbot/memory/get" || len(p) != 1 {
				t.Fatal("read payload", p)
			}
		case "memory/replace":
			if r.URL.Path != "/dshbot/memory/replace" || len(p) != 4 || p["track"] != "user" || p["memoryRevision"] != "rev-u" || p["text"] != "- changed" {
				t.Fatal("write payload", p)
			}
			response = `{"botId":"b","track":"user","entries":["changed"],"text":"- changed","revision":"next-u","memoryRevision":"next-u","view":{"ns":"dsh-bot","revision":8,"value":{"items":[{"id":"b","sessionId":"s"}]}}}`
		default:
			t.Fatal("invented endpoint", req.Method)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"result":{"ok":true,"value":` + response + `}}`))}, nil
	})}}
	got, err := c.DshbotMemory(context.Background(), "b")
	if err != nil || got.Bot.Revision != "rev-b" || calls != 1 {
		t.Fatal(err, got)
	}
	cmd := DshbotMemoryCommand{BotID: "b", SessionID: "s", Track: "user", MemoryRevision: "rev-u", Text: "- changed", Limit: 4000}
	receipt, err := c.DshbotReplaceMemory(context.Background(), cmd)
	if err != nil || receipt.MemoryRevision != "next-u" || calls != 2 {
		t.Fatal(err)
	}
	for _, bad := range []DshbotMemoryCommand{{BotID: "b", SessionID: "s", Track: "other", MemoryRevision: "x", Limit: 8000}, {BotID: "b", SessionID: "s", Track: "bot", Limit: 8000}, {BotID: "b", SessionID: "s", Track: "bot", MemoryRevision: "x", Text: "😀", Limit: 1}} {
		if _, err := c.DshbotReplaceMemory(context.Background(), bad); err == nil {
			t.Fatal("invalid write accepted")
		}
	}
	if calls != 2 {
		t.Fatal("invalid request reached transport")
	}
	receipt.Track = "bot"
	if ValidateDshbotMemoryReceipt(cmd, receipt) == nil {
		t.Fatal("foreign track receipt accepted")
	}
}
func TestBotMemoryEntrySerializationAndLimits(t *testing.T) {
	if got := DshbotMemoryText([]string{" \ufefffirst  ", " ", "second\nline"}); got != "- first\n- second\nline" {
		t.Fatal(got)
	}
	if WhaleTextLength("😀") != 2 {
		t.Fatal("must count UTF-16")
	}
	v := DshbotMemoryResult{BotID: "b", Bot: DshbotMemoryTrack{Revision: "b"}, User: DshbotMemoryTrack{Revision: "u"}, Limits: DshbotMemoryLimits{Bot: 90000, User: 4000}}
	if err := ValidateDshbotMemory("b", v); err != nil || v.Limit("bot") != 64000 {
		t.Fatal(err)
	}
	v.User.Revision = ""
	if ValidateDshbotMemory("b", v) == nil {
		t.Fatal("missing CAS accepted")
	}
}
