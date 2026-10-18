package chat

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"whaleisle/internal/store/session"
)

func TestChatStatusRestoration(t *testing.T) {
	for _, tc := range []struct{ reason, status, detail string }{
		{`{"kind":"max-tokens"}`, "Output limit reached", "token limit"},
		{`{"kind":"error","error":{"message":"upstream timeout","code":"TIMEOUT"}}`, "Failed", "upstream timeout [TIMEOUT]"},
		{`{"kind":"aborted","reason":{"kind":"hook","reason":"deepseek-account/signed-out"}}`, "Stopped", "ACCOUNT_SIGNED_OUT"},
	} {
		msg := session.Message{Raw: json.RawMessage(`{"data":{"reason":` + tc.reason + `}}`)}
		if !strings.Contains(completedTurnStatus(msg), tc.status) || !strings.Contains(terminalReasonDetail(msg), tc.detail) {
			t.Fatalf("reason %s: %s / %s", tc.reason, completedTurnStatus(msg), terminalReasonDetail(msg))
		}
	}
	auth := session.Message{Raw: json.RawMessage(`{"data":{"reason":{"kind":"error","error":{"message":"secret credential fragment","code":"AUTH"}}}}`)}
	if strings.Contains(terminalReasonDetail(auth), "secret") || !strings.Contains(terminalReasonDetail(auth), "AUTH") {
		t.Fatal("AUTH diagnostic leak")
	}
	now := time.Unix(0, 0)
	if retrySeconds(now.Add(1500*time.Millisecond), now) != 2 || retrySeconds(now, now.Add(time.Second)) != 0 {
		t.Fatal("countdown")
	}
}
