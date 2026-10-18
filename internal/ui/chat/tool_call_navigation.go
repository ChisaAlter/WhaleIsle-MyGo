package chat

import (
	"strings"
	"whaleisle/internal/store/session"
)

// TrajectoryToolCallTarget resolves a call identity through the same lifecycle
// projection as the visible tool card. It never searches display text or JSON
// substrings, and returns a raw ledger row rather than a chat aggregation ID.
func TrajectoryToolCallTarget(messages []session.Message, callID string) (string, int, bool) {
	if strings.TrimSpace(callID) == "" {
		return "", 0, false
	}
	for i, msg := range messages {
		for _, projected := range session.ProjectToolMessages([]session.Message{msg}) {
			stack := []*session.ToolCallView{projected.Tool}
			seen := map[*session.ToolCallView]bool{}
			for len(stack) > 0 {
				call := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if call == nil || seen[call] {
					continue
				}
				seen[call] = true
				if call.CallID == callID {
					return messageToTrajRow(msg).ID, i, true
				}
				stack = append(stack, call.Children...)
			}
		}
	}
	return "", 0, false
}
