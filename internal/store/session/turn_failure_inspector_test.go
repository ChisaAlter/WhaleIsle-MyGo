package session

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestInspectorRetainsTheProviderTextOfAFailedTurn keeps the raw failure
// reachable after the shell stops printing it: the retained event is the detail
// affordance a person reports a bug from, and the sentence never replaces it.
func TestInspectorRetainsTheProviderTextOfAFailedTurn(t *testing.T) {
	s := New()
	raw := json.RawMessage(`{"type":"turn/end","seq":9,"data":{"reason":{"kind":"error","error":{"message":"llm-deepseek: no API key for provider route \"deepseek-official\"; store DEEPSEEK_API_KEY through the credentials service","code":"MISSING_CREDENTIAL"}}}}`)
	s.mu.Lock()
	s.captureInspectorEventWithHeadLocked(raw, "turn/end", 9)
	s.mu.Unlock()
	for _, record := range s.InspectorSnapshot().Records {
		if strings.Contains(string(record.Raw), "no API key for provider route") {
			return
		}
	}
	t.Fatal("the retained event no longer carries the provider's own text")
}
