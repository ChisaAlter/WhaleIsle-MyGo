package chat

import (
	"encoding/json"
	"testing"

	"whaleisle/internal/store/session"
)

func TestCitationsFromToolResultJSON(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"tool/result",
		"data":{
			"message":{
				"content":[{"type":"text","text":"done"}]
			},
			"extra":{"results":[{"title":"Example","url":"https://example.com/a"}]}
		}
	}`)
	cites := CitationsForMessage(session.Message{Kind: session.MessageKindTool, Raw: raw})
	if len(cites) != 1 || cites[0].URL != "https://example.com/a" {
		t.Fatalf("%#v", cites)
	}
}

func TestCitationsFromMarkdownBody(t *testing.T) {
	body := "See [Docs](https://docs.example.com) and https://go.dev/dl/"
	cites := CitationsForMessage(session.Message{Role: "assistant", Body: body})
	if len(cites) < 2 {
		t.Fatalf("%#v", cites)
	}
}

func TestCitationsDedupe(t *testing.T) {
	raw := json.RawMessage(`{"url":"https://x.test","nested":{"url":"https://x.test"}}`)
	cites := CitationsForMessage(session.Message{Raw: raw})
	if len(cites) != 1 {
		t.Fatalf("%#v", cites)
	}
}
