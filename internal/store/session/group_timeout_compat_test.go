package session

import (
	"encoding/json"
	"testing"
)

// Fixtures follow the original groupMemberResultState timeout|timed.?out
// contract through actual room result projection, without a Host or model.
func TestGroupResultTimeoutSourceCompatibility(t *testing.T) {
	for _, tc := range []struct {
		status string
		want   string
	}{
		{"timeout", "timeout"},
		{"timedout", "timeout"},
		{"TIMED_OUT", "timeout"},
		{"timed-out", "timeout"},
		{"timed out", "timeout"},
		{"timed/out", "timeout"},
		{"timed--out", "settled"},
		{"timed  out", "settled"},
		{"timed\rout", "settled"},
		{"timed\nout", "settled"},
		{"timed\u2028out", "settled"},
		{"timed\u2029out", "settled"},
		{"timed😀out", "settled"},
		{"time out", "settled"},
		{"timingout", "settled"},
		{"limit timedout", "capped"},
		{"failed timedout", "timeout"},
		{"pending", "pending"},
	} {
		t.Run(tc.status, func(t *testing.T) {
			result, err := json.Marshal(map[string]any{
				"seq": 2, "type": "tool/result", "data": map[string]any{
					"status": tc.status,
					"message": map[string]any{
						"source":  map[string]any{"callId": "c1"},
						"content": []any{map[string]any{"type": "text", "text": "member reply"}},
					},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			threads := ProjectGroupThreads([]Message{
				groupFixture(`{"seq":1,"type":"tool/call","data":{"name":"ask_participant","callId":"c1","arguments":{"botId":"b1","threadId":"t1"}}}`),
				groupFixture(string(result)),
			}, nil)
			if len(threads) != 1 || threads[0].ID != "t1" || len(threads[0].Messages) != 1 {
				t.Fatalf("unexpected projected threads: %+v", threads)
			}
			message := threads[0].Messages[0]
			if message.State != tc.want {
				t.Fatalf("status %q projected as %q, want %q", tc.status, message.State, tc.want)
			}
			if tc.want == "settled" && message.Text != "member reply" {
				t.Fatal("unrecognized status lost its visible reply")
			}
			if tc.want != "settled" && message.Text != "" {
				t.Fatal("control status was projected as a visible member reply")
			}
		})
	}
}
