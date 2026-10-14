package phoneremote

import (
	"fmt"
	"net"
	"strings"

	"whaleisle/internal/config"
)

// Readiness is a config/credentials checklist for phone remote (no mobile client).
type Readiness string

const (
	// ReadinessDisabled means pairing is off.
	ReadinessDisabled Readiness = "disabled"
	// ReadinessIncomplete means required fields for phone pairing are missing.
	ReadinessIncomplete Readiness = "incomplete"
	// ReadinessReady means config/credentials look sufficient for phone pairing.
	ReadinessReady Readiness = "ready"
)

// Snapshot is config-derived phone remote stub status.
type Snapshot struct {
	RemoteEnabled       bool
	RemoteMode          string
	RemotePort          int
	RemoteBindAddress   string
	RemoteLanTLS        bool
	RemoteAppBaseURL    string
	RemoteRelayEndpoint string
	RemoteRelayUseTLS   bool
	HasRemoteToken      bool
	HasRemoteRelayToken bool
	MuxReady            bool
	DeviceCount         int
	DeviceNames         []string
	Readiness           Readiness
	Missing             []string
	ConnectURL          string
}

// FromConfig builds a snapshot from normalized config and runtime flags.
func FromConfig(cfg config.Config, hasRemoteToken, hasRemoteRelayToken, muxReady bool, deviceCount int) Snapshot {
	s := Snapshot{
		RemoteEnabled:       cfg.RemoteEnabled,
		RemoteMode:          cfg.RemoteMode,
		RemotePort:          cfg.RemotePort,
		RemoteBindAddress:   cfg.RemoteBindAddress,
		RemoteLanTLS:        cfg.RemoteLanTls,
		RemoteAppBaseURL:    cfg.RemoteAppBaseUrl,
		RemoteRelayEndpoint: cfg.RemoteRelayEndpoint,
		RemoteRelayUseTLS:   cfg.RemoteRelayUseTls,
		HasRemoteToken:      hasRemoteToken,
		HasRemoteRelayToken: hasRemoteRelayToken,
		MuxReady:            muxReady,
		DeviceCount:         deviceCount,
	}
	s.Readiness, s.Missing = Assess(s)
	s.ConnectURL = ConnectURL(s)
	return s
}

// Assess returns readiness and missing checklist items for phone remote.
func Assess(s Snapshot) (Readiness, []string) {
	var missing []string
	if !s.RemoteEnabled {
		missing = append(missing, "remoteEnabled")
	}
	if !s.HasRemoteToken {
		missing = append(missing, "remoteToken")
	}
	mode := normalizeMode(s.RemoteMode)
	switch mode {
	case "lan":
		if s.RemotePort < 1024 || s.RemotePort > 65535 {
			missing = append(missing, "remotePort")
		}
		if strings.TrimSpace(s.RemoteBindAddress) == "" {
			missing = append(missing, "remoteBindAddress")
		}
	case "relay":
		if !s.HasRemoteRelayToken {
			missing = append(missing, "remoteRelayToken")
		}
	}
	if !s.MuxReady {
		missing = append(missing, "mux")
	}
	if len(missing) > 0 {
		if !s.RemoteEnabled {
			return ReadinessDisabled, missing
		}
		return ReadinessIncomplete, missing
	}
	return ReadinessReady, nil
}

// ConnectURL returns a phone connect URL when constructable from config.
// Prefers remoteAppBaseUrl; otherwise builds a LAN http(s)://bind:port URL
// when mode is lan and the bind address is not loopback.
func ConnectURL(s Snapshot) string {
	if base := strings.TrimSpace(s.RemoteAppBaseURL); base != "" {
		return strings.TrimRight(base, "/")
	}
	if normalizeMode(s.RemoteMode) != "lan" {
		return ""
	}
	bind := strings.TrimSpace(s.RemoteBindAddress)
	if bind == "" || isLoopbackBind(bind) {
		return ""
	}
	if s.RemotePort < 1024 || s.RemotePort > 65535 {
		return ""
	}
	scheme := "http"
	if s.RemoteLanTLS {
		scheme = "https"
	}
	host := bind
	if ip := net.ParseIP(bind); ip != nil && ip.To4() == nil {
		host = "[" + bind + "]"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, s.RemotePort)
}

func normalizeMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "lan", "relay", "ssh":
		return m
	default:
		if strings.Contains(m, "ssh") {
			return "ssh"
		}
		if strings.Contains(m, "lan") {
			return "lan"
		}
		return "relay"
	}
}

func isLoopbackBind(bind string) bool {
	b := strings.TrimSpace(strings.ToLower(bind))
	switch b {
	case "127.0.0.1", "localhost", "::1", "0.0.0.0", "[::]", "::":
		return true
	}
	if ip := net.ParseIP(bind); ip != nil {
		return ip.IsLoopback() || ip.IsUnspecified()
	}
	return false
}
