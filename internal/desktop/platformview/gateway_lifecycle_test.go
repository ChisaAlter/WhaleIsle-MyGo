package platformview

import (
	"net"
	"strings"
	"testing"
	"time"
)

// A Close that beats the Serve goroutine must still release the port. The
// account lease is cancelled in that window on every route exit, and a lease
// that keeps accepting connections is an open origin the app believes it
// stopped.
func TestClosedGatewayStopsAcceptingConnections(t *testing.T) {
	for i := 0; i < 200; i++ {
		g, err := New(Session{Origin: "https://account.example", Token: "fixture-only"}, ClientMetadata{Locale: "en", Platform: "desktop-win"})
		if err != nil {
			t.Fatalf("gateway %d: %v", i, err)
		}
		address := strings.TrimPrefix(g.origin, "http://")
		g.Close()
		connection, err := net.DialTimeout("tcp", address, 200*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			t.Fatalf("gateway %d still accepted a connection after Close", i)
		}
	}
}
