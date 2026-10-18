package sshremote

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// ProbeResult is a TCP reachability check against the configured bind/port
// (pairing daemon listen address). This is not an SSH handshake.
type ProbeResult struct {
	OK      bool
	Address string
	Err     string
	Latency time.Duration
}

// ProbeAddress performs a bounded TCP dial to host:port. It is shared by the
// pairing-port probe and the SSH connection form so both report the same
// address, latency and error shape.
func ProbeAddress(ctx context.Context, host string, port int) ProbeResult {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	out := ProbeResult{Address: addr}
	if port < 1 || port > 65535 {
		out.Err = "invalid port"
		return out
	}
	if ctx == nil {
		ctx = context.Background()
	}
	d := net.Dialer{Timeout: 800 * time.Millisecond}
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", addr)
	out.Latency = time.Since(start)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	_ = conn.Close()
	out.OK = true
	return out
}

// ProbeSSH checks only TCP reachability of an SSH profile. It deliberately
// does not authenticate or mutate known_hosts; Connect owns those actions.
func ProbeSSH(ctx context.Context, cfg TunnelConfig) ProbeResult {
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	return ProbeAddress(ctx, cfg.Host, port)
}

// ProbeTCP dials RemoteBindAddress:RemotePort with a short timeout.
func ProbeTCP(ctx context.Context, s Snapshot) ProbeResult {
	return ProbeAddress(ctx, s.RemoteBindAddress, s.RemotePort)
}

// FormatProbe is a one-line status for UI.
func FormatProbe(p ProbeResult) string {
	if p.OK {
		return fmt.Sprintf("tcp ok %s (%s)", p.Address, p.Latency.Round(time.Millisecond))
	}
	if p.Err != "" {
		return fmt.Sprintf("tcp fail %s: %s", p.Address, p.Err)
	}
	return "tcp not probed"
}
