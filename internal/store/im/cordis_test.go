package im

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseCordisBotsHealthAndLastMessageError(t *testing.T) {
	raw := json.RawMessage(`{
		"snapshot": {
			"bots": [{
				"botId": "b1",
				"connected": false,
				"state": "error",
				"bot": {"name": "Test Bot"},
				"health": {"summary": "长连接异常"},
				"lastMessageError": {"code": "x", "reason": "y", "message": "delivery failed", "referenceId": "r1", "at": 1}
			}]
		}
	}`)
	rows := parseCordisBots(raw, "feishu")
	if len(rows) != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	d := rows[0].Detail
	if !strings.Contains(d, "feishu") || !strings.Contains(d, "error") {
		t.Fatalf("detail missing channel/state: %q", d)
	}
	if !strings.Contains(d, "长连接异常") {
		t.Fatalf("detail missing health: %q", d)
	}
	if !strings.Contains(d, "lastMsg: delivery failed") {
		t.Fatalf("detail missing lastMessageError: %q", d)
	}
}

func TestParseCordisBotsAccountErrorFallback(t *testing.T) {
	raw := json.RawMessage(`{
		"bots": [{
			"botId": "b2",
			"connected": false,
			"state": "offline",
			"bot": {"name": "Slack"},
			"error": {"message": "Token expired"}
		}]
	}`)
	rows := parseCordisBots(raw, "slack")
	if len(rows) != 1 {
		t.Fatalf("rows = %+v", rows)
	}
	if !strings.Contains(rows[0].Detail, "Token expired") {
		t.Fatalf("detail = %q", rows[0].Detail)
	}
}
