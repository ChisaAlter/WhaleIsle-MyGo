package sshremote

import (
	"context"
	"fmt"
	"net"
	"strconv"
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

// ProbeTCP dials RemoteBindAddress:RemotePort with a short timeout.
func ProbeTCP(ctx context.Context, s Snapshot) ProbeResult {
	host := s.RemoteBindAddress
	if host == "" {
		host = "127.0.0.1"
	}
	addr := net.JoinHostPort(host, strconv.Itoa(s.RemotePort))
	out := ProbeResult{Address: addr}
	if s.RemotePort < 1 {
		out.Err = "invalid port"
		return out
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
