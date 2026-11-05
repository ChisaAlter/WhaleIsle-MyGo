package sshremote

import (
	"context"
	"strings"
	"testing"
)

func TestTunnelForwardErrorPublishesAfterConnectReturns(t *testing.T) {
	tunnel := NewTunnelSession()
	tunnel.state = TunnelConnected
	var messages []string
	tunnel.OnChange(func() { _, _, message := tunnel.Snapshot(); messages = append(messages, message) })
	// Parsing fails before any socket/client use. The production async forward
	// worker must invalidate its consumer even though Connect already succeeded.
	tunnel.forward(context.Background(), nil, "invalid-forward")
	if len(messages) != 1 || !strings.Contains(messages[0], "forward spec:") {
		t.Fatalf("late error invisible: %v", messages)
	}
}
