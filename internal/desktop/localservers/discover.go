// Package localservers discovers already-running desktop development servers.
// It only attempts TCP handshakes on a fixed loopback allowlist; it never starts
// processes, sends HTTP requests, or scans remote/workspace network addresses.
package localservers

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	URL  string
	Port int
}

var ports = [...]int{3000, 3001, 3333, 4173, 4200, 4321, 5000, 5173, 5174, 5175, 5500, 8000, 8080, 8081, 8888, 9000}

// Discover matches the original preview's fixed-port TCP discovery contract.
// A reachable TCP port does not guarantee that the service speaks HTTP.
func Discover(ctx context.Context) []Server {
	return discover(ctx, func(ctx context.Context, port int) bool {
		dialer := net.Dialer{Timeout: 200 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	})
}

func discover(ctx context.Context, probe func(context.Context, int) bool) []Server {
	if ctx.Err() != nil {
		return nil
	}
	found := make([]bool, len(ports))
	var wg sync.WaitGroup
	for i, port := range ports {
		wg.Add(1)
		go func(i, port int) { defer wg.Done(); found[i] = probe(ctx, port) }(i, port)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return nil
	}
	var servers []Server
	for i, port := range ports {
		if found[i] {
			servers = append(servers, Server{URL: "http://127.0.0.1:" + strconv.Itoa(port), Port: port})
		}
	}
	return servers
}
