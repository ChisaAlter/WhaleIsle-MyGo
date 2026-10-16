// Package localservers discovers already-running desktop development servers.
// It only attempts TCP handshakes on a fixed loopback allowlist; it never starts
// processes, sends HTTP requests, or scans remote/workspace network addresses.
package localservers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type Server struct {
	URL  string
	Port int
}

var ports = [...]int{3000, 3001, 3333, 4173, 4200, 4321, 5000, 5173, 5174, 5175, 5500, 8000, 8080, 8081, 8888, 9000}

// Discover matches the original preview's fixed-port TCP discovery contract.
// A reachable TCP port does not guarantee that the service speaks HTTP.
type Discovery struct {
	Servers []Server
	Error   string
}

func Discover(ctx context.Context) []Server { return DiscoverStatus(ctx).Servers }

// DiscoverStatus distinguishes ordinary absent listeners from failures such as
// denied socket access or exhausted resources. A partial list remains usable.
func DiscoverStatus(ctx context.Context) Discovery {
	return discoverWithStatus(ctx, func(ctx context.Context, port int) error {
		dialer := net.Dialer{Timeout: 200 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	})
}

func discover(ctx context.Context, probe func(context.Context, int) bool) []Server {
	return discoverWithStatus(ctx, func(ctx context.Context, port int) error {
		if probe(ctx, port) {
			return nil
		}
		return syscall.ECONNREFUSED
	}).Servers
}

func discoverWithStatus(ctx context.Context, probe func(context.Context, int) error) Discovery {
	if ctx.Err() != nil {
		return Discovery{}
	}
	results := make([]error, len(ports))
	var wg sync.WaitGroup
	for i, port := range ports {
		wg.Add(1)
		go func(i, port int) { defer wg.Done(); results[i] = probe(ctx, port) }(i, port)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return Discovery{}
	}
	var result Discovery
	for i, port := range ports {
		err := results[i]
		if err == nil {
			result.Servers = append(result.Servers, Server{URL: "http://127.0.0.1:" + strconv.Itoa(port), Port: port})
		} else if result.Error == "" && !ordinaryProbeMiss(err) {
			result.Error = fmt.Sprintf("127.0.0.1:%d: %v", port, err)
		}
	}
	return result
}

func ordinaryProbeMiss(err error) bool {
	// Windows socket errors are Winsock codes, not syscall's synthetic POSIX errno.
	if runtime.GOOS == "windows" && (errors.Is(err, syscall.Errno(10061)) || errors.Is(err, syscall.Errno(1225))) {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}
